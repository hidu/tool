package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hidu/go-speed"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/hidu/tool/go/kafka/kafka-agent/kafka"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	speedData := speed.NewSpeed("call", 5, func(msg string) {
		log.Println("speed", msg)
	})

	defer conn.Close()
	c := pb.NewAgentClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	req := &pb.Request{
		Topic: "test",
		Key:   []byte("hello" + name),
		Value: []byte("value"),
		Logid: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	for i := 0; i < 10000; i++ {
		r, err := c.Send(context.Background(), req)
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Println("Greeting", r)
		speedData.Success("send", 1)
	}
	speedData.Stop()
}
