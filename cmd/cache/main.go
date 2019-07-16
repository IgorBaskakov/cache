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
	"github.com/IgorBaskakov/service/redis"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type storage interface {
	Set(string, string, time.Duration) error
	Get(string) (string, error)
}

type config struct {
	urls             []string
	minTimeout       uint
	maxTimeout       uint
	numberOfRequests int
}

const (
	port     = ":50051"
	storAddr = "localhost:6379"
)

var _ pb.CacherServer = &cacherServer{}

// cacherServer is used to implement cache.CacherServer.
type cacherServer struct {
	config
	storage storage
}

type response struct {
	url, content string
	cached       bool
}

// GetRandomDataStream implements cache.CacherServer.
func (cs *cacherServer) GetRandomDataStream(in *pb.Nothing, stream pb.Cacher_GetRandomDataStreamServer) error {
	wg := &sync.WaitGroup{}
	result := make(chan response)

	for i := 0; i < cs.numberOfRequests; i++ {
		url := getRandomURL(cs.urls)
		wg.Add(1)
		go func() {
			defer wg.Done()
			cont, err := cs.storage.Get(url)
			if err != nil {
				log.Printf("get by url %q from redis error: %+v", url, err)
				getContent(url, result)
				return
			}

			result <- response{url: url, content: cont, cached: true}
		}()
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	for resp := range result {
		cd := &pb.CacheData{Str: resp.content}
		if err := stream.Send(cd); err != nil {
			return err
		}
		if !resp.cached {
			timeout := getRandomTimeout(cs.minTimeout, cs.maxTimeout)
			dur := time.Duration(int64(timeout)) * time.Second
			fmt.Printf("set url %q to redis\n", resp.url)
			cs.storage.Set(resp.url, resp.content, dur)
		}
	}

	return nil
}

func getRandomURL(urls []string) string {
	i := rand.Intn(len(urls))
	return urls[i]
}

func getRandomTimeout(min, max uint) uint {
	var border uint
	if max < min {
		min, max = max, min
	}
	border = max - min

	timeout := rand.Intn(int(border))
	return uint(timeout) + min
}

func getContent(url string, out chan response) {
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

	out <- response{url: url, content: string(data), cached: false}
}

func getStorage() storage {
	cl, err := redis.NewClient(storAddr)
	if err != nil {
		log.Fatalf("get redis client error: %+v", err)
	}
	return cl
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

	st := getStorage()
	fmt.Printf("storage = %+v\n", st)

	// flag.Parse()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterCacherServer(s, &cacherServer{config: conf, storage: st})
	fmt.Printf("Start gRPC server at port %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
