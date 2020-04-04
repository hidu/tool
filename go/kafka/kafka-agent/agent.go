package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	// "os"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	pb "github.com/hidu/tool/go/kafka/kafka-agent/kafka"
)

var addr = flag.String("addr", ":50051", "agent listen addr")
var brokerList = flag.String("brokers", "127.0.0.1:9092", "The comma separated list of brokers in the Kafka cluster. You can also set the KAFKA_PEERS environment variable")

type server struct {
	Producers map[string]sarama.SyncProducer
	mu        sync.Mutex
}

func NewServer() *server {
	s := &server{
		Producers: make(map[string]sarama.SyncProducer),
	}
	return s
}

func (s *server) Send(ctx context.Context, in *pb.Request) (*pb.Reply, error) {
	pr, _ := peer.FromContext(ctx)

	re := &pb.Reply{}
	logData := []string{
		fmt.Sprintf("logid=%s", in.Logid),
		fmt.Sprintf("remote=%s", pr.Addr.String()),
		fmt.Sprintf("topic=%s", in.Topic),
		fmt.Sprintf("key=%s", string(in.Key)),
		fmt.Sprintf("value_len=%d", len(in.Value)),
	}
	defer (func() {
		logData = append(logData, fmt.Sprintf("errno=%d", re.Errno))
		logData = append(logData, fmt.Sprintf("offset=%d", re.Offset))
		log.Println(logData)
	})()

	producer, err := s.getProducer(in.Topic)

	if err != nil {
		re.Errno = 502404
		re.Error = "get producer failed:" + err.Error()
		return re, nil
	}
	log.Println("input:", in, in.Partition)
	message := &sarama.ProducerMessage{
		Topic: in.Topic,
		Key:   sarama.ByteEncoder(in.Key),
		Value: sarama.ByteEncoder(in.Value),
	}
	partition, offset, err := producer.SendMessage(message)

	if err != nil {
		re.Errno = 500
		re.Error = err.Error()
	} else {
		re.Offset = offset
		re.Partition = partition
	}
	return re, nil
}

func (s *server) getProducer(topic string) (sarama.SyncProducer, error) {
	if sp, has := s.Producers[topic]; has {
		return sp, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewHashPartitioner
	sp, err := sarama.NewSyncProducer(strings.Split(*brokerList, ","), config)
	if err == nil {
		s.Producers[topic] = sp
	}
	return sp, err
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAgentServer(s, NewServer())
	s.Serve(lis)
}
