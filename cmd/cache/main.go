package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	pb "github.com/IgorBaskakov/service/cache"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type config struct {
	urls             []string
	minTimeout       uint
	maxTimeout       uint
	numberOfRequests int
}

const (
	port = ":50051"
)

var _ pb.CacherServer = &cacherServer{}

// cacherServer is used to implement cache.CacherServer.
type cacherServer struct {
	config config
}

// GetRandomDataStream implements cache.CacherServer.
func (cs *cacherServer) GetRandomDataStream(in *pb.Nothing, stream pb.Cacher_GetRandomDataStreamServer) error {
	wg := &sync.WaitGroup{}
	result := make(chan []byte)

	for i := 0; i < cs.config.numberOfRequests; i++ {
		url := getRandomURL(cs.config.urls)
		wg.Add(1)
		go func() {
			getContent(url, result)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	for data := range result {
		cd := &pb.CacheData{Str: string(data)}
		if err := stream.Send(cd); err != nil {
			return err
		}
		// setToCache(string(data))
	}
	return nil
}

func getRandomURL(urls []string) string {
	i := rand.Intn(len(urls))
	return urls[i]
}

func getContent(url string, out chan []byte) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    3 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("http get error: %+v", err)
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error read data from response: %+v", err)
		return
	}

	out <- data
}

func main() {
	rand.Seed(time.Now().UnixNano())
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("fatal error of read config file: %v", err)
	}

	conf := config{
		urls:             viper.GetStringSlice("URLs"),
		minTimeout:       viper.GetUint("MinTimeout"),
		maxTimeout:       viper.GetUint("MaxTimeout"),
		numberOfRequests: viper.GetInt("NumberOfRequests"),
	}
	fmt.Printf("config = %+v\n", conf)

	// flag.Parse()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterCacherServer(s, &cacherServer{config: conf})
	fmt.Printf("Start gRPC server at port %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
