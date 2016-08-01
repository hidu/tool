/**
* @see https://github.com/Shopify/sarama/blob/master/tools/kafka-console-consumer/kafka-console-consumer.go
 */
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"bytes"
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/hidu/go-speed"
	"github.com/hidu/goutils/object"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var (
	brokerList = flag.String("brokers", os.Getenv("KAFKA_PEERS"), "The comma separated list of brokers in the Kafka cluster")
	topic      = flag.String("topic", "", "REQUIRED: the topic to consume")
	partitions = flag.String("partitions", "all", "The partitions to consume, can be 'all' or comma-separated numbers")
	offset     = flag.String("offset", "newest", "The offset to start with. Can be `oldest`, `newest`")
	verbose    = flag.Bool("verbose", false, "Whether to turn on sarama logging")
	//	bufferSize = flag.Int("buffer-size", 256, "The buffer size of the message channel.")
	httpConsumerUrl      = flag.String("http-con-url", "", "http consumer url")
	httpConsumerTimeout  = flag.Int("http-con-timeout", 10000, "http consumer timeout,ms")
	httpConsumerReTryNum = flag.Int("rt", 3, "http consumer retry times")
	httpConsumerJsonSuc  = flag.Bool("http-check-json", true, `check http consumer response json {"errno":0}`)

	con = flag.Int("con", 10, "Concurrent Num")

	logger = log.New(os.Stderr, "", log.LstdFlags)
)

const User_Agent = "go-kafka-pusher/0.1/hidu"

var speedData *speed.Speed

func main() {
	flag.Parse()

	if *brokerList == "" {
		printUsageErrorAndExit("You have to provide -brokers as a comma-separated list, or set the KAFKA_PEERS environment variable.")
	}

	if *topic == "" {
		printUsageErrorAndExit("-topic is required")
	}
	if *httpConsumerUrl == "" {
		printUsageErrorAndExit("-http-con-url is required")
	}

	_, err := url.Parse(*httpConsumerUrl)
	if err != nil {
		printUsageErrorAndExit("invalid consumer url")
	}

	if *verbose {
		sarama.Logger = logger
	}

	speedData = speed.NewSpeed("call", 5, func(msg string, sp *speed.Speed) {
		logger.Println("[speed]", msg)
	})

	var initialOffset int64
	switch *offset {
	case "oldest":
		initialOffset = sarama.OffsetOldest
	case "newest":
		initialOffset = sarama.OffsetNewest
	default:
		printUsageErrorAndExit("-offset should be `oldest` or `newest`")
	}

	c, err := sarama.NewConsumer(strings.Split(*brokerList, ","), nil)
	if err != nil {
		printErrorAndExit(69, "Failed to start consumer: %s", err)
	}

	partitionList, err := getPartitions(c)
	if err != nil {
		printErrorAndExit(69, "Failed to get the list of partitions: %s", err)
	}

	var (
		messages = make(chan *sarama.ConsumerMessage, *con)
		closing  = make(chan struct{})
		wg       sync.WaitGroup
	)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Kill, os.Interrupt)
		<-signals
		logger.Println("Initiating shutdown of consumer...")
		close(closing)
	}()

	for _, partition := range partitionList {
		pc, err := c.ConsumePartition(*topic, partition, initialOffset)
		if err != nil {
			printErrorAndExit(69, "Failed to start consumer for partition %d: %s", partition, err)
		}

		go func(pc sarama.PartitionConsumer) {
			<-closing
			pc.AsyncClose()
		}(pc)

		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for message := range pc.Messages() {
				messages <- message
			}
		}(pc)
	}

	for num := 0; num < *con; num++ {
		go func() {
			client := &http.Client{
				Timeout: time.Duration(*httpConsumerTimeout) * time.Millisecond,
			}
			for msg := range messages {
				subNum := 0
				for i := 0; i < *httpConsumerReTryNum; i++ {
					if dealMessage(client, msg, i) {
						subNum = 1
						break
					}
				}
				speedData.Inc(1, len(msg.Value), subNum)
			}
		}()
	}

	wg.Wait()
	logger.Println("Done consuming topic", *topic)
	close(messages)
	speedData.Stop()

	if err := c.Close(); err != nil {
		logger.Println("Failed to close consumer: ", err)
	}
}

func dealMessage(client *http.Client, msg *sarama.ConsumerMessage, try int) bool {
	val := string(msg.Value)
	key := string(msg.Key)
	logData := []string{
		"info",
		fmt.Sprintf("try=%d", try),
		"is_suc=failed",
		fmt.Sprintf("offset=%d", msg.Offset),
		fmt.Sprintf("partition=%d", msg.Partition),
		fmt.Sprintf("key=%s", url.QueryEscape(key)),
		fmt.Sprintf("value_len=%d", len(msg.Value)),
	}
	defer (func() {
		logger.Println(logData)
	})()

	if *verbose {
		fmt.Printf("Key:\t%s\n", key)
		fmt.Printf("Value:\t%s\n", val)
		fmt.Println()
	}
	vs := make(url.Values)
	vs.Add("kafka_key", key)
	vs.Add("kafka_value", val)
	vs.Add("kafka_partition", fmt.Sprintf("%d", msg.Partition))
	vs.Add("kafka_offset", fmt.Sprintf("%d", msg.Offset))

	start := time.Now()

	req, reqErr := http.NewRequest("POST", *httpConsumerUrl, bytes.NewReader([]byte(vs.Encode())))
	if reqErr != nil {
		logData[0] = "error"
		logData = append(logData, "http_build_req_error:", reqErr.Error())
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", User_Agent)

	resp, err := client.Do(req)

	used := time.Now().Sub(start)

	logData = append(logData, fmt.Sprintf("used_ms=%d", used.Nanoseconds()/1e6))
	if err != nil {
		logData[0] = "error"
		logData = append(logData, "http_client_error:", err.Error())
		return false
	}
	defer resp.Body.Close()
	bd, respErr := ioutil.ReadAll(resp.Body)

	if respErr != nil {
		logData[0] = "error"
		logData = append(logData, "http_read_resp_error:", respErr.Error())
		return false
	}
	logData = append(logData, fmt.Sprintf("http_code=%d", resp.StatusCode))

	isSuc := true
	if resp.StatusCode != http.StatusOK {
		isSuc = false
	}

	if isSuc && *httpConsumerJsonSuc {
		var obj interface{}
		jsonErr := json.Unmarshal(bd, &obj)
		if jsonErr != nil {
			logData = append(logData, "resp_errno=unknow")
			logData = append(logData, "resp_json_err:", jsonErr.Error())
			isSuc = false
		} else {
			errno, _ := object.NewInterfaceWalker(obj).GetString("/errno")
			logData = append(logData, "resp_errno="+errno)
			isSuc = errno == "0"
		}
	}

	if isSuc {
		logData[2] = "is_suc=suc"
	}

	logData = append(logData, fmt.Sprintf("resp_len=%d", len(bd)), fmt.Sprintf("resp=%s", url.QueryEscape(string(bd))))

	return isSuc
}

func getPartitions(c sarama.Consumer) ([]int32, error) {
	if *partitions == "all" {
		return c.Partitions(*topic)
	}

	tmp := strings.Split(*partitions, ",")
	var pList []int32
	for i := range tmp {
		val, err := strconv.ParseInt(tmp[i], 10, 32)
		if err != nil {
			return nil, err
		}
		pList = append(pList, int32(val))
	}

	return pList, nil
}

func printErrorAndExit(code int, format string, values ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
	fmt.Fprintln(os.Stderr)
	os.Exit(code)
}

func printUsageErrorAndExit(format string, values ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Available command line options:")
	flag.PrintDefaults()
	os.Exit(64)
}
