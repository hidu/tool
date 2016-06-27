package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"github.com/hidu/go-speed"
)

var conc = flag.Uint("c", 10, "Concurrent Num")
var timeout = flag.Int64("t", 800, "Timeout,ms")
var method = flag.String("method", "GET", "HTTP Method")
var noResp = flag.Bool("nr", false, "No respose (default false)")

var complex = flag.Bool("complex", false, `Complex input (default false)`)

var client *http.Client
var idx uint64

var jobs chan *http.Request
var version = "0.1 20160627"

func init() {
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\ncat urllist.txt|url_call_conc -c 100")
		fmt.Println("\n site: https://github.com/hidu/tool")
		fmt.Println(" version:", version)
	}
}

var speedData *speed.Speed

func main() {
	flag.Parse()
	timeOut := time.Duration(*timeout) * time.Millisecond
	client = &http.Client{
		Timeout: timeOut,
	}
	speedData=speed.NewSpeed("call",5,func(msg string,sp *speed.Speed){
		log.Println("speed",msg)
	});
	
	var urlStr = strings.TrimSpace(flag.Arg(0))

	jobs = make(chan *http.Request, *conc)

	for i := 0; i < int(*conc); i++ {
		go urlCallWorker(jobs)
	}
	if urlStr == "" {
		if *complex {
			parseComplexStdIn()
		} else {
			parseSimpleStdIn()
		}
	} else {
		req, err := http.NewRequest(strings.ToUpper(*method), urlStr, nil)
		if err != nil {
			log.Println(err)
		} else {
			jobs <- req
		}
	}
	close(jobs)
	time.Sleep(timeOut+500*time.Millisecond)
	speedData.Stop()
	fmt.Println("done,total:", idx)
}

func parseSimpleStdIn() {
	var urlStr string
	buf := bufio.NewReaderSize(os.Stdin, 81920)
	for {
		line, err := buf.ReadBytes('\n')
		if len(line) > 0 {
			urlStr = strings.TrimSpace(string(line))
			if urlStr == "" {
				continue
			}
			req, err := http.NewRequest(strings.ToUpper(*method), urlStr, nil)
			if err != nil {
				log.Println("wf:", urlStr, "err:", err)
				continue
			}
			jobs <- req
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("wf:", string(line), "err:", err)
		}
	}
}

func parseComplexStdIn() {
	buf := bufio.NewReaderSize(os.Stdin, 81920)
	for {
		headLen, err := buf.ReadString('|')
		if err == io.EOF {
			break
		}
		hl, err := strconv.Atoi(headLen[0 : len(headLen)-1])
		if err != nil {
			log.Fatalln("read head length faild:", headLen)
		}
		headBf := make([]byte, hl)
		buf.Read(headBf)

		var hdObj *Head
		err = json.Unmarshal(headBf, &hdObj)
		if err != nil {
			log.Fatalln("parse head faild:", headLen)
		}

		bodyLen, err := buf.ReadString('|')
		if err != nil {
			log.Fatalln("read body length faild:", err)
		}
		bl, err := strconv.Atoi(bodyLen[0 : len(bodyLen)-1])
		if err != nil {
			log.Fatalln("parse body length faild:", headLen)
		}

		var body io.Reader

		if bl > 0 {
			bodyBf := make([]byte, bl)
			buf.Read(bodyBf)
			body = bytes.NewReader(bodyBf)
		}
		if hdObj.Method == "" {
			hdObj.Method = "GET"
		}
		req, err := http.NewRequest(strings.ToUpper(hdObj.Method), hdObj.Url, body)
		if err != nil {
			log.Fatalln("wf:", hdObj, "err:", err)
		}

		if hdObj.Header != nil {
			for k, v := range hdObj.Header {
				req.Header.Add(k, v)
			}
		}

		jobs <- req
	}
}

type Head struct {
	Url    string            `json:"url"`
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
}

func urlCallWorker(jobs <-chan *http.Request) {
	var respStr string
	for req := range jobs {
		id := atomic.AddUint64(&idx, 1)
		urlStr := req.URL.String()
		
		logData:=[]string{}
		logData=append(logData,fmt.Sprintf("id=%d %s %s",id,req.Method,urlStr))
		resp, err := client.Do(req)
		logData=append(logData,fmt.Sprintf("client_err=%v",err))
		if(err==nil){
			defer resp.Body.Close()
			bd, err := ioutil.ReadAll(resp.Body)
			logData=append(logData,fmt.Sprintf("http_code=%d blen=%d bd_err=%v",resp.StatusCode,len(bd),err))
			respStr=string(bd)
		}
		log.Println(logData)
		if(!*noResp){
			fmt.Printf("id=%d\t%d\t%s\n", id, len(respStr),respStr)
		}
		speedData.Inc(1,0)
	}
}

