// @see https://github.com/Shopify/sarama/blob/master/tools/kafka-console-consumer/kafka-console-consumer.go

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/hidu/go-speed"
	"github.com/hidu/goutils/object"
)

var (
	brokerList = flag.String("brokers", os.Getenv("PUSHER_BROKERS"), "The comma separated list of brokers in the Kafka cluster")
	topic      = flag.String("topic", os.Getenv("PUSHER_TOPIC"), "REQUIRED: the topic to consume")
	partitions = flag.String("partitions", "all", "The partitions to consume, can be 'all' or comma-separated numbers")
	offset     = flag.Int64("offset", -1, "The offset to start with. Can be -2:oldest, -1:newest")
	verbose    = flag.Bool("verbose", false, "Whether to turn on sarama logging")

	// 	bufferSize = flag.Int("buffer-size", 256, "The buffer size of the message channel.")
	httpConsumerUrl      = flag.String("http-con-url", os.Getenv("PUSHER_HTTP_CON_URL"), "http consumer url")
	httpConsumerTimeout  = flag.Int("http-con-timeout", 10000, "http consumer timeout,ms")
	httpConsumerReTryNum = flag.Int("rt", 3, "http consumer retry times")
	httpConsumerJsonSuc  = flag.String("http-check-json", "/errno=0", `check http consumer response json {"errno":0}`)

	con = flag.Int("con", 10, "Concurrent Num")

	logger = log.New(os.Stderr, "", log.LstdFlags)
)

const User_Agent = "go-kafka-pusher/0.1/hidu"

var speedData *speed.Speed

var HttpConsumerUrls []string

var dealId uint64

var currentOffset int64

func main() {
	flag.Parse()

	if len(*brokerList) == 0 {
		printUsageErrorAndExit("You have to provide -brokers as a comma-separated list, or set the KAFKA_PEERS environment variable.")
	}

	if len(*topic) == 0 {
		printUsageErrorAndExit("-topic is required")
	}

	HttpConsumerUrls = parseConUrlFlag()

	if *verbose {
		sarama.Logger = logger
	}

	if len(*httpConsumerJsonSuc) != 0 && !strings.Contains(*httpConsumerJsonSuc, "=") {
		printUsageErrorAndExit(`-http-check-json must contains "="`)
	}

	speedData = speed.NewSpeed("call", 5, func(msg string) {
		logger.Println("[speed]", msg)
	})

	currentOffset = *offset

start:
	c, err := sarama.NewConsumer(strings.Split(*brokerList, ","), nil)
	if err != nil {
		logger.Printf("Failed to start consumer: %s\n", err)
		time.Sleep(500 * time.Millisecond)
		goto start
	}
	logger.Println("start consumer success")
	partitionList, err := getPartitions(c)
	if err != nil {
		logger.Printf("Failed to get the list of partitions: %s\n", err)

		c.Close()
		time.Sleep(500 * time.Millisecond)
		goto start
	}

	var (
		messages = make(chan *sarama.ConsumerMessage, *con)
		closing  = make(chan struct{})
		wg       sync.WaitGroup
	)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
		<-signals
		logger.Println("Initiating shutdown of consumer...")
		close(closing)
	}()

	for _, partition := range partitionList {
		pc, err := c.ConsumePartition(*topic, partition, currentOffset)
		if err != nil {
			logger.Printf("Failed to start consumer for partition %d: %s\n", partition, err)
			c.Close()
			time.Sleep(500 * time.Millisecond)
			goto start
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
				currentOffset = msg.Offset
				subNum := 0
				for i := 0; i < *httpConsumerReTryNum; i++ {
					if dealMessage(client, msg, i) {
						subNum = 1
						break
					}
				}
				speedData.Success("send", subNum)
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

func parseConUrlFlag() []string {
	if len(*httpConsumerUrl) == 0 {
		printUsageErrorAndExit("-http-con-url is required")
	}
	str := strings.ReplaceAll(strings.TrimSpace(*httpConsumerUrl), "\n", ";")
	arr := strings.Split(str, ";")
	var urls []string
	for _, line := range arr {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		_, err := url.Parse(line)
		if err != nil {
			printUsageErrorAndExit("invalid consumer url:" + line)
		}
		urls = append(urls, line)
	}

	if len(urls) < 1 {
		printUsageErrorAndExit("-http-con-url is empty")
	}

	return urls
}

func dealMessage(client *http.Client, msg *sarama.ConsumerMessage, try int) bool {
	curDealId := atomic.AddUint64(&dealId, 1)
	val := string(msg.Value)
	key := string(msg.Key)
	logData := []string{
		fmt.Sprintf("id=%d", curDealId),
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
		fmt.Printf("Partition:\t%d\n", msg.Partition)
		fmt.Printf("Offset:\t%d\n", msg.Offset)
		fmt.Printf("Key:\t%s\n", key)
		fmt.Printf("Value:\t%s\n", val)
		fmt.Println()
	}
	vs := make(url.Values)
	vs.Add("kafka_key", key)
	vs.Add("kafka_partition", fmt.Sprintf("%d", msg.Partition))
	vs.Add("kafka_offset", fmt.Sprintf("%d", msg.Offset))
	qs := vs.Encode()
	vs.Add("kafka_value", val)

	start := time.Now()
	urlNew := HttpConsumerUrls[int(curDealId%uint64(len(HttpConsumerUrls)))]
	if strings.Contains(urlNew, "?") {
		urlNew = urlNew + "&" + qs
	} else {
		urlNew = urlNew + "?" + qs
	}

	req, reqErr := http.NewRequest("POST", urlNew, bytes.NewReader([]byte(vs.Encode())))

	if reqErr != nil {
		logData[1] = "error"
		logData = append(logData, "http_build_req_error:", reqErr.Error())
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", User_Agent)

	resp, err := client.Do(req)

	used := time.Since(start)

	logData = append(logData, fmt.Sprintf("used_ms=%d", used.Nanoseconds()/1e6))
	if err != nil {
		logData[1] = "error"
		logData = append(logData, "http_client_error:", err.Error())
		return false
	}
	defer resp.Body.Close()
	bd, respErr := io.ReadAll(resp.Body)

	if respErr != nil {
		logData[1] = "error"
		logData = append(logData, "http_read_resp_error:", respErr.Error())
		return false
	}
	logData = append(logData, fmt.Sprintf("http_code=%d", resp.StatusCode))

	isSuc := true
	if resp.StatusCode != http.StatusOK {
		isSuc = false
	}

	if isSuc && len(*httpConsumerJsonSuc) != 0 {
		var obj any
		jsonErr := json.Unmarshal(bd, &obj)
		if jsonErr != nil {
			logData = append(logData, "resp_errno=unknow")
			logData = append(logData, "resp_json_err:", jsonErr.Error())
			isSuc = false
		} else {
			_info := strings.SplitN(*httpConsumerJsonSuc, "=", 2)
			errno, _ := object.NewInterfaceWalker(obj).GetString(_info[0])
			logData = append(logData, "resp_errno="+errno)
			isSuc = errno == _info[1]
		}
	}

	if isSuc {
		logData[3] = "is_suc=suc"
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

// func printErrorAndExit(code int, format string, values ...any) {
// 	fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
// 	fmt.Fprintln(os.Stderr)
// 	logger.Printf("ERROR: %s\n", fmt.Sprintf(format, values...))
// 	os.Exit(code)
// }

func printUsageErrorAndExit(format string, values ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Available command line options:")
	flag.PrintDefaults()
	os.Exit(64)
}
