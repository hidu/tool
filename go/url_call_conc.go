package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hidu/go-speed"
	"github.com/hidu/goutils/log_util"
	"io"
	"io/ioutil"
	glog "log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var version = "0.1.2 20160724"

var conc = flag.Uint("c", 10, "Concurrent Num")
var timeout = flag.Int64("t", 10000, "Timeout,ms")
var method = flag.String("method", "GET", "HTTP Method")
var noResp = flag.Bool("nr", false, "No Respose (default false)")
var ua = flag.String("ua", "url_call_conc/"+version, "User-Agent")
var logPath = flag.String("log", "", "log file prex,default stderr")

var complex = flag.Bool("complex", false, `Complex Input (default false)`)

var v = flag.Bool("version", false, "Version:"+version)

var idx uint64

var jobs chan *http.Request

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

var timeOut time.Duration
var log *glog.Logger

func main() {
	flag.Parse()
	if *v {
		fmt.Println(version)
		return
	}
	log = glog.New(os.Stderr, "", glog.LstdFlags)
	if *logPath != "" {
		log_util.SetLogFile(log, *logPath, log_util.LOG_TYPE_HOUR)
	}

	startTime := time.Now()
	timeOut = time.Duration(*timeout) * time.Millisecond

	speedData = speed.NewSpeed("call", 5, func(msg string, sp *speed.Speed) {
		log.Println("speed", msg)
	})

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
			req.Header.Set("User-Agent", *ua)
			jobs <- req
		}
	}
	close(jobs)
	time.Sleep(timeOut + 500*time.Millisecond)
	speedData.Stop()

	timeUsed := time.Now().Sub(startTime)

	log.Println("[info]done total:", idx, "used:", timeUsed)
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
				log.Println("[wf]", urlStr, "err:", err)
				continue
			}
			req.Header.Set("User-Agent", *ua)
			jobs <- req
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("[wf]", string(line), "err:", err)
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
			log.Fatalln("[fat] read head length faild:", headLen)
		}
		headBf := make([]byte, hl)
		buf.Read(headBf)

		var hdObj *Head
		err = json.Unmarshal(headBf, &hdObj)
		if err != nil {
			log.Fatalln("[fat] parse head faild:", headLen)
		}

		bodyLen, err := buf.ReadString('|')
		if err != nil {
			log.Fatalln("[fat] read body length faild:", err)
		}
		bl, err := strconv.Atoi(bodyLen[0 : len(bodyLen)-1])
		if err != nil {
			log.Fatalln("[fat] parse body length faild:", headLen)
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
			log.Fatalln("[wf]", hdObj, "err:", err)
		}
		req.Header.Set("User-Agent", *ua)
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
	client := &http.Client{
		Timeout: timeOut,
	}
	var respStr string
	var sucNum int
	lr := &logRequest{
		buf: new(bytes.Buffer),
	}
	for req := range jobs {
		id := atomic.AddUint64(&idx, 1)
		urlStr := req.URL.String()
		lr.reset()

		lr.addNotice("id", id)
		lr.addNotice("method", req.Method)
		lr.addNotice("url", urlStr)

		startTime := time.Now()
		resp, err := client.Do(req)
		timeUsed := time.Now().Sub(startTime)

		lr.addNotice("used_ms", float64(timeUsed.Nanoseconds())/1e6)

		lr.addNotice("client_err", err)
		if err == nil {
			if resp.StatusCode == 200 {
				sucNum = 1
			} else {
				sucNum = 0
			}
			defer resp.Body.Close()
			bd, r_err := ioutil.ReadAll(resp.Body)
			lr.addNotice("http_code", resp.StatusCode)
			lr.addNotice("resp_len", len(bd))
			lr.addNotice("resp_err", r_err)
			respStr = string(bd)
			if !*noResp {
				lr.addNotice("resp_body", respStr)
			}
			lr.print("info")
		} else {
			lr.addNotice("http_code", 0)
			lr.addNotice("resp_len", 0)
			lr.addNotice("resp_err", err)
			lr.print("wf")
		}
		speedData.Inc(1, len(respStr), sucNum)
	}
}

type logRequest struct {
	buf *bytes.Buffer
}

func (lr *logRequest) reset() {
	lr.buf.Reset()
}

func (lr *logRequest) addNotice(key string, val interface{}) {
	lr.buf.WriteString(key)
	lr.buf.WriteString("=")
	if val == nil {
		lr.buf.WriteString("nil")
	} else {
		lr.buf.WriteString(url.QueryEscape(fmt.Sprintf("%v", val)))
	}
	lr.buf.WriteString(" ")
}

func (lr *logRequest) print(ty string) {
	log.Printf("[%s] %s \n", ty, lr.buf.String())
}
