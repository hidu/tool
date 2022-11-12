package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/beefsack/go-rate"
)

var concurrentMax uint = 200

var inputPath = flag.String("f", "./url_list.txt", "Input File Path,first row is the url")
var outPath = flag.String("o", "", "Output File Path")
var cps = flag.Uint("cps", 10000, "Limit Number per Second")
var cpm = flag.Uint("cpm", 9999999, "Limit Number per Minute")
var concurrent = flag.Uint("c", 1, "Number of multiple requests to make at a time,max "+fmt.Sprint(concurrentMax))
var timeOut = flag.Int("s", 30, "Seconds to max. wait for each response")

var outFile *os.File

var rateDefault *rate.RateLimiter
var rateMinute *rate.RateLimiter

var lineReg = regexp.MustCompile(`\s`)
var historyUrlMap = make(map[string]int)

var taskTotal int64
var taskDoneTotal int64 = 0

var client *http.Client

var writeChan = make(chan bool, 1)

var jobs = make(chan string, 100)

func main() {
	flag.Parse()

	rateDefault = rate.New(int(*cps), 1*time.Second)
	rateMinute = rate.New(int(*cpm), 1*time.Minute)

	if *concurrent > concurrentMax {
		log.Fatalln("-c concurrent max value is 10")
	}

	rd, err := os.Open(*inputPath)
	if err != nil {
		log.Fatalln("open input file failed:", err)
	}
	defer rd.Close()
	if len(*outPath) == 0 {
		log.Fatalln("output file path is empty")
	}
	client = &http.Client{Timeout: time.Duration(*timeOut) * time.Second}

	loadHistoryFile(*outPath)

	outFile, err = os.OpenFile(*outPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalln("open output file failed:", err)
	}

	defer (func() {
		outFile.Sync()
		outFile.Close()
	})()

	done := make(chan bool)
	go (func() {
		buf := bufio.NewReaderSize(rd, 81920)
		for {
			line, err := buf.ReadBytes('\n')

			lineStr := strings.TrimSpace(string(line))
			if len(lineStr) != 0 {
				addTask(lineStr)
			}
			if err == io.EOF {
				close(jobs)
				done <- true
				log.Println("read file eof")
				break
			}
			if err != nil {
				log.Println(err)
			}
		}
	})()

	var i uint
	for i = 0; i < *concurrent; i++ {
		go parseWorker(i, jobs)
	}

	log.Println("parse input file done,wating...")
	<-done
	log.Println("wait for done")

	for taskTotal > taskDoneTotal {
		time.Sleep(1 * time.Second)
		log.Println("wait...... taskTotal:", taskTotal, ",taskDone:", taskDoneTotal)
	}

	log.Println("all finish")
}

func addTask(lineStr string) {
	atomic.AddInt64(&taskTotal, 1)
	jobs <- lineStr
}

func loadHistoryFile(outPath string) {
	rd, err := os.Open(outPath)
	if err != nil {
		log.Println("start load history file failed")
		return
	}
	defer rd.Close()
	log.Println("start load history output file")
	buf := bufio.NewReaderSize(rd, 8192)
	for {
		line, err := buf.ReadBytes('\n')

		lineStr := strings.TrimSpace(string(line))
		if len(lineStr) != 0 {
			strs := lineReg.Split(lineStr, 3)
			if len(strs) > 2 {
				code, _ := strconv.ParseInt(strs[1], 10, 64)
				if code == 200 {
					historyUrlMap[strs[0]] = 1
				}
			}
		}
		if err == io.EOF {
			break
		}
	}
	log.Println("finish load history output file")
}

func saveResult(urlRaw string, lineRaw string, err error, response *http.Response) {
	status := 0
	var respLen int64
	var respHeader string
	var contentType string
	if err == nil && response != nil {
		status = response.StatusCode
		respLen = response.ContentLength
		rh, _ := json.Marshal(response.Header)
		respHeader = string(rh)
		historyUrlMap[urlRaw] = 1
		contentType = response.Header.Get("Content-Type")
		tmp := strings.Split(contentType, ";")
		contentType = tmp[0]
	}
	var vals []string
	vals = append(vals, urlRaw)
	vals = append(vals, fmt.Sprint(status))
	vals = append(vals, fmt.Sprint(respLen))
	vals = append(vals, fmt.Sprint(contentType))
	vals = append(vals, respHeader)

	ss := lineReg.Split(lineRaw, -1)

	if len(ss) > 1 {
		vals = append(vals, ss[1:]...)
	}
	str := strings.Join(vals, "\t") + "\n"
	fmt.Print(str)

	writeChan <- true

	n, err := outFile.WriteString(str)
	log.Println("write:", n, err)

	<-writeChan
}

func parseWorker(id uint, job <-chan string) {
	for line := range job {
		parseLine(line, 0)
	}
}

func parseLine(line string, times int) {
	if times == 0 {
		defer atomic.AddInt64(&taskDoneTotal, 1)
	}

	ss := lineReg.Split(string(line), 2)
	urlRaw := ss[0]
	if _, has := historyUrlMap[urlRaw]; has {
		log.Println("skip", urlRaw)
		return
	}
	_, err := url.Parse(urlRaw)
	if err != nil {
		log.Println("parse url failed,url:", urlRaw, err)
		// 		saveResult(urlRaw, line, err, nil)
		return
	}

	rateDefault.Wait()
	rateMinute.Wait()

	resp, err := client.Head(urlRaw)
	saveResult(urlRaw, line, err, resp)

	if times < 1 && (err != nil || resp.StatusCode != 200) {
		parseLine(line, times+1)
	}
}
