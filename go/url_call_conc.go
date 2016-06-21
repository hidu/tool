package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	"io"
	"os"
	"sync/atomic"
)

var conc = flag.Uint("c", 10, "Concurrent Num")
var timeout = flag.Int64("t", 800, "timeout,ms")
var client *http.Client
var idx uint64

func main() {
	flag.Parse()
	timeOut:=time.Duration(*timeout) * time.Millisecond
	client = &http.Client{
		Timeout: timeOut,
	}

	var urlStr = strings.TrimSpace(flag.Arg(0))
	
	jobs := make(chan string, *conc)
    
	for i := 0; i < int(*conc); i++ {
		go urlCallWorker(jobs)
	}
	if urlStr == "" {
		buf := bufio.NewReaderSize(os.Stdin, 81920)
		for {
			line, err := buf.ReadBytes('\n')
			if len(line) > 0 {
				uStr := strings.TrimSpace(string(line))
				if uStr == "" {
					continue
				}
				jobs <- uStr
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("wf:", string(line), "err:", err)
			}
		}
	} else {
		jobs <- urlStr
	}
	close(jobs)
	time.Sleep(2*timeOut)
	fmt.Println("done,total:",idx)
}

func urlCallWorker(jobs <-chan string) {
	for u := range jobs {
		res, err := urlCall(u)
		id:=atomic.AddUint64(&idx,1)
		log.Println("idx=",id,u)
		if err != nil {
			log.Println(id,"wf", u, "err:", err)
		} else {
			fmt.Println("res",id,res)
		}
	}
}

func urlCall(urlStr string) (res string, err error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bd, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bd), nil
}
