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
	"sync"
	"sync/atomic"
	"time"
)

//import _ "net/http/pprof"

var version = "0.1.4 20161115"

var conc = flag.Uint("c", 10, "Concurrent Num [conc]")
var timeout = flag.Int64("t", 10000, "Timeout,ms")
var method = flag.String("method", "GET", "HTTP Method")
var noResp = flag.Bool("nr", false, "No Respose (default false)")
var ua = flag.String("ua", "url_call_conc/"+version, "User-Agent")
var logPath = flag.String("log", "", "log file prex,default stderr")
var start = flag.Int("start", 0, "start item num")

var complex = flag.Bool("complex", false, `Complex Input (default false)`)

var v = flag.Bool("version", false, "Version:"+version)

var idx uint64

var jobs chan *http.Request

func init() {
	usage := flag.Usage
	flag.Usage = func() {
		usage()
		fmt.Println("\ncat urllist.txt|url_call_conc -c 100")
		fmt.Println("\n use dynamic conf [ ./url_call_conc.conf ] to change runtime params :")
		fmt.Println(`{"conc":10}`)
		fmt.Println("\n site: https://github.com/hidu/tool")
		fmt.Println(" version:", version)
	}
}

var speedData *speed.Speed

var timeOut time.Duration
var log *glog.Logger

var rw sync.RWMutex

type UrlCallConcConf struct {
	FromFile bool   `json:"from_file"`
	ConcMax  uint   `json:"conc"`
	Start    uint64 `json:"start"`
	//    QpsMax  uint `json:"qps"`
}

var confCur *UrlCallConcConf
var confLast *UrlCallConcConf

func (c *UrlCallConcConf) String() string {
	bf, _ := json.Marshal(c)
	return string(bf)
}

func getConf() *UrlCallConcConf {
	conf := &UrlCallConcConf{
		ConcMax: *conc,
		Start:   uint64(*start),
	}
	fname := "url_call_conc.conf"
	bs, err := ioutil.ReadFile(fname)
	if err != nil {
		//log.Println("[wf] read_conf failed", fname, "skip,", err.Error())
	} else {
		err := json.Unmarshal(bs, &conf)
		if err != nil {
			log.Println("[wf] parse_conf ", fname, "failed,skip,", err.Error())
		} else {
			conf.FromFile = true
		}
	}
	log.Println("[trace] now_conf is:", conf)
	return conf
}

func startWorkers() {
	if confCur.ConcMax > confLast.ConcMax {
		log.Println("[trace] startWorkers,last_conc=", confLast.ConcMax, "cur_conc=", confCur.ConcMax)
		for i := confLast.ConcMax; i < confCur.ConcMax; i++ {
			go urlCallWorker(jobs, i)
		}
	}
}

var workerRunning int64 = 0

func main() {
	flag.Parse()
	if *v {
		fmt.Println(version)
		return
	}

	if *ua == "mac" {
		*ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.1103.21"
	}

	//	go func() {
	//		http.ListenAndServe("localhost:6060", nil)
	//	}()

	log = glog.New(os.Stderr, "", glog.LstdFlags)
	if *logPath != "" {
		log_util.SetLogFile(log, *logPath, log_util.LOG_TYPE_HOUR)
	}
	startTime := time.Now()
	timeOut = time.Duration(*timeout) * time.Millisecond

	speedData = speed.NewSpeed("call", 5, func(msg string, sp *speed.Speed) {
		n := atomic.LoadInt64(&workerRunning)
		log.Println("speed", msg, "running_workers=", n)
	})

	confCur = getConf()
	confLast = &UrlCallConcConf{}

	var urlStr = strings.TrimSpace(flag.Arg(0))

	jobs = make(chan *http.Request, 1024)

	startWorkers()

	ticker := time.NewTicker(30 * time.Second)

	go func() {
		for range ticker.C {
			(func() {
				confTmp := getConf()
				if confTmp.String() != confCur.String() {
					rw.Lock()
					defer rw.Unlock()
					confLast = confCur
					confCur = confTmp
					startWorkers()
				}
			})()
		}
	}()

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
	var num int64 = 1
	for num > 0 {
		num = atomic.LoadInt64(&workerRunning)
		log.Println("[trace] e_running_worker_total=", workerRunning, "waiting...")
		time.Sleep(1 * time.Second)
	}
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

func urlCallWorker(jobs <-chan *http.Request, workerId uint) {
	_runningNum := atomic.AddInt64(&workerRunning, 1)
	defer func() {
		num := atomic.AddInt64(&workerRunning, -1)
		log.Println("[trace] this worker closed,running_worker_total=", num)
	}()

	log.Println("[trace] urlCallWorker_start,worker_id=", workerId, "running_worker_total=", _runningNum)
	client := &http.Client{
		Timeout: timeOut,
	}
	var respStr string
	var sucNum int
	lr := &logRequest{
		buf: new(bytes.Buffer),
	}
	isSkip := false
	isOutRange := false
	var errOutOfRange = fmt.Errorf("urlCallWorker_OutOfRange")

	var dealRequest = func(req *http.Request) error {
		id := atomic.AddUint64(&idx, 1)
		urlStr := req.URL.String()
		lr.reset()

		lr.addNotice("id", id)
		lr.addNotice("worker_id", workerId)
		lr.addNotice("method", req.Method)
		lr.addNotice("url", urlStr)

		(func() {
			rw.RLock()
			defer rw.RUnlock()
			isSkip = confCur.Start >= id
			isOutRange = workerId >= confCur.ConcMax
		})()

		if isSkip {
			lr.print("[skip]")
			return nil
		}

		startTime := time.Now()
		resp, err := client.Do(req)
		timeUsed := time.Now().Sub(startTime)

		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}

		lr.addNotice("used_ms", float64(timeUsed.Nanoseconds())/1e6)

		lr.addNotice("client_err", err)

		if err == nil {
			if resp.StatusCode == 200 {
				sucNum = 1
			} else {
				sucNum = 0
			}
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

		if isOutRange {
			return errOutOfRange
		}
		return nil
	}

	for req := range jobs {
		err := dealRequest(req)

		if err == errOutOfRange {
			log.Println("[trace]", err.Error(), "close_worker,workerId=", workerId)
			break
		}
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
