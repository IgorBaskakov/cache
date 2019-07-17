package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/IgorBaskakov/service/cache"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	address       = "localhost:50051"
	maxConnection = 500
	maxLen        = 100
)

func sendGRPCRequest(wg *sync.WaitGroup, ops *uint64) {
	defer wg.Done()

	opt1 := grpc.WithInsecure()
	kep := keepalive.ClientParameters{
		Time: 2 * time.Minute,
	}
	opt2 := grpc.WithKeepaliveParams(kep)
	options := []grpc.DialOption{opt1, opt2}
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, options...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	client := pb.NewCacherClient(conn)
	stream, err := client.GetRandomDataStream(context.Background(), &pb.Nothing{})
	if err != nil {
		// log.Printf("error get random data stream: %v", err)
		return
	}

	for {
		_, err := stream.Recv()
		if err != nil { //io.EOF
			return
		}
		atomic.AddUint64(ops, 1)
	}
}

func main() {
	var ops uint64
	defer func(now time.Time) {
		fmt.Printf("spent %s\n", time.Since(now))
	}(time.Now())
	wg := &sync.WaitGroup{}
	for i := 0; i < maxConnection; i++ {
		wg.Add(1)
		go sendGRPCRequest(wg, &ops)
	}
	wg.Wait()
	opsFinal := atomic.LoadUint64(&ops)
	fmt.Println("count read from stream:", opsFinal)
}
