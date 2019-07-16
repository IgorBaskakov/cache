package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	pb "github.com/IgorBaskakov/service/cache"

	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewCacherClient(conn)

	stream, err := client.GetRandomDataStream(context.Background(), &pb.Nothing{})
	if err != nil {
		log.Fatalf("error get random data stream: %v", err)
	}

	i := 0
	for {
		i++
		cdata, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.GetRandomDataStream(_) = _, %v", client, err)
		}
		fmt.Printf("get result from %d url with content:\n %s\n\n", i, cutContent(cdata.Str))
		// fmt.Println(cdata.Str)
	}
}

const maxLen = 360

func cutContent(indata string) string {
	data := strings.TrimSpace(indata)
	r := []rune(data)
	return string(r[:maxLen]) + "..."
}
