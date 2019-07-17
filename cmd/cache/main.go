package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/IgorBaskakov/service/cache"
	"github.com/IgorBaskakov/service/redis"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
}

type chans struct {
	sync.Mutex
	urls map[string]chan struct{}
}

var urlChans = chans{urls: make(map[string]chan struct{})}
var counterStart, counterEnd uint64

// GetRandomDataStream implements cache.CacherServer.
func (cs *cacherServer) GetRandomDataStream(in *pb.Nothing, stream pb.Cacher_GetRandomDataStreamServer) error {
	// atomic.AddUint64(&counterStart, 1)
	// defer func() {
	// 	atomic.AddUint64(&counterEnd, 1)
	// }()
	wg := &sync.WaitGroup{}
	result := make(chan response)

	for i := 0; i < cs.numberOfRequests; i++ {
		url := getRandomURL(cs.urls)
		wg.Add(1)
		go func() {
			defer wg.Done()

			urlChans.Lock()
			ch, ok := urlChans.urls[url]
			urlChans.Unlock()
			if !ok {
				return
			}
			ch <- struct{}{}

			defer func() {
				<-ch
			}()

			var resp *response
			cont, err := cs.storage.Get(url)
			if err != nil {
				if resp, err = getContent(url); err != nil {
					return
				}

				timeout := getRandomTimeout(cs.minTimeout, cs.maxTimeout)
				dur := time.Duration(int64(timeout)) * time.Second
				cs.storage.Set(resp.url, resp.content, dur)
				cont = resp.content
			}

			result <- response{url: url, content: cont}
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

func getContent(url string) (*response, error) {
	tr := &http.Transport{
		MaxIdleConns:       30,
		IdleConnTimeout:    1 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("http get error: %+v", err)
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error read data from response: %+v", err)
		return nil, err
	}

	return &response{url: url, content: string(data)}, nil
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
	for _, url := range conf.urls {
		urlChans.urls[url] = make(chan struct{}, 1)
	}

	st := getStorage()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opt1 := grpc.MaxConcurrentStreams(2000)
	opt2 := grpc.ConnectionTimeout(1 * time.Minute)
	kep := keepalive.EnforcementPolicy{
		MinTime:             1 * time.Minute,
		PermitWithoutStream: true,
	}
	opt3 := grpc.KeepaliveEnforcementPolicy(kep)
	options := []grpc.ServerOption{opt1, opt2, opt3}
	s := grpc.NewServer(options...)

	pb.RegisterCacherServer(s, &cacherServer{config: conf, storage: st})
	fmt.Printf("Start gRPC server at port %s\n", port)

	// go printCounter()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func printCounter() {
	for {
		time.Sleep(1 * time.Second)
		opsStart := atomic.LoadUint64(&counterStart)
		fmt.Println("counter start:", opsStart)
		opsEnd := atomic.LoadUint64(&counterEnd)
		fmt.Println("counter end:", opsEnd)
	}
}
