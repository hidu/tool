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
var version = "0.1.5 20161125"

var conc = flag.Uint("c", 10, "Concurrent Num [conc]")
var timeout = flag.Int64("t", 10000, "Timeout,ms")
var method = flag.String("method", "GET", "HTTP Method")
var noResp = flag.Bool("nr", false, "No Respose (default false)")
var ua = flag.String("ua", "url_call_conc/"+version, "User-Agent")
var logPath = flag.String("log", "", "log file prex,default stderr")
var start = flag.Int("start", 0, "start item num")

var complex = flag.Bool("complex", false, `Complex Input (default false)`)

var v = flag.Bool("version", false, "Version:"+version)

var flagRetry = flag.Int("retry", 3, "retry times")
var flagConf = flag.String("conf", "url_call_conc.conf", "dynamic conf")

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
	FromFile bool   `json:"-"`
	Done     bool   `json:"-"`
	ConcMax  uint   `json:"conc"`
	Start    uint64 `json:"start"`
	Retry    int    `json:"retry"`
	EndID    uint64 `json:"end_id"`
}

var confCur *UrlCallConcConf
var confLast *UrlCallConcConf

func (c *UrlCallConcConf) String() string {
	bf, _ := json.Marshal(c)
	return string(bf)
}

func (c *UrlCallConcConf) TryWrite() {
	if !c.FromFile {
		return
	}
	conf := getConf()
	if idx > conf.Start {
		conf.Start = idx
	}
	if c.Done {
		conf.EndID = idx
		conf.Start = 0
	}
	bf, _ := json.MarshalIndent(conf, "", "  ")
	ioutil.WriteFile(*flagConf, bf, 0666)
}

func getConf() *UrlCallConcConf {
	conf := &UrlCallConcConf{
		ConcMax: *conc,
		Start:   uint64(*start),
	}
	bs, err := ioutil.ReadFile(*flagConf)
	if err != nil {
		//log.Println("[wf] read_conf failed", fname, "skip,", err.Error())
	} else {
		err := json.Unmarshal(bs, &conf)
		if err != nil {
			log.Println("[wf] parse_conf ", *flagConf, "failed,skip,", err.Error())
		} else {
			conf.FromFile = true
		}
	}
	if conf.Retry < 1 {
		conf.Retry = *flagRetry
	}
	log.Println("[trace] now_conf is:", conf)
	return conf
}

func getCurConf() *UrlCallConcConf {
	rw.RLock()
	defer rw.RUnlock()
	return confCur
}

func startWorkers() {
	if confCur.ConcMax > confLast.ConcMax {
		log.Println("[pid=", pid, "][trace] startWorkers,last_conc=", confLast.ConcMax, "cur_conc=", confCur.ConcMax)
		for i := confLast.ConcMax; i < confCur.ConcMax; i++ {
			go urlCallWorker(jobs, i)
		}
	}
}

var workerRunning int64 = 0

var pid = 0
var exitErr error

func main() {
	flag.Parse()
	if *v {
		fmt.Println(version)
		return
	}
	pid = os.Getpid()

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

	log.Println("url_call_conc_start-------------------------->")

	confCur = getConf()
	confLast = &UrlCallConcConf{}

	startTime := time.Now()
	timeOut = time.Duration(*timeout) * time.Millisecond

	jobsBufSize := 20

	if int(confCur.ConcMax) > jobsBufSize {
		jobsBufSize = int(confCur.ConcMax)
	}
	jobs = make(chan *http.Request, jobsBufSize)

	speedData = speed.NewSpeed("url_call_cocc", 5, func(msg string) {
		n := atomic.LoadInt64(&workerRunning)
		log.Println("speed", msg, "running_workers=", n, "jobs_buf=", len(jobs))
	})

	var urlStr = strings.TrimSpace(flag.Arg(0))

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
				confCur.TryWrite()
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
		log.Println("[pid=", pid, "][trace] running_worker_total=", workerRunning, "waiting...")
		time.Sleep(1 * time.Second)
	}
	speedData.Stop()

	timeUsed := time.Now().Sub(startTime)

	log.Println("[pid=", pid, "][info]done total:", idx, "used:", timeUsed)
	confCur.Done = true
	confCur.TryWrite()
	log.Println("url_call_conc_finish<=========================")

	if exitErr != nil {
		log.Fatalln("exit_with_error:", exitErr)
	}
}

func parseSimpleStdIn() {
	var urlStr string
	buf := bufio.NewReaderSize(os.Stdin, 3000)
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
	buf := bufio.NewReaderSize(os.Stdin, 1024)
	for {
		headLen, err := buf.ReadString('|')
		if err == io.EOF {
			break
		}
		hlStr := headLen[0 : len(headLen)-1]
		hl, err := strconv.Atoi(hlStr)
		if err != nil {
			exitErr = fmt.Errorf("[fat] read head length faild,not int,header_len_str=[%s]", hlStr)
			log.Println(exitErr)
			break
		}
		headBf, err := bufPeed(buf, hl)
		if err != nil {
			exitErr = fmt.Errorf("[fat] read header faild,len=%d,head=[%s],err=%s", hl, string(headBf), err.Error())
			log.Println(exitErr)
			break
		}

		var hdObj *Head
		err = json.Unmarshal(headBf, &hdObj)
		if err != nil {
			exitErr = fmt.Errorf("[fat] parse head faild:%s", err.Error())
			log.Println(exitErr)
			break
		}

		bodyLen, err := buf.ReadString('|')
		if err != nil {
			exitErr = fmt.Errorf("[fat] read body length faild:%s", err.Error())
			log.Println(exitErr)
			break
		}

		blStr := bodyLen[0 : len(bodyLen)-1]
		bl, err := strconv.Atoi(blStr)
		if err != nil {
			exitErr = fmt.Errorf("[fat] parse body length faild,str=[%s]", blStr)
			log.Println(exitErr)
			break
		}

		var body io.Reader

		if bl > 0 {
			bodyBf, err := bufPeed(buf, bl)
			if err != nil {
				exitErr = fmt.Errorf("[fat] read body faild:%s", err.Error())
				log.Println(exitErr)
				break
			}
			body = bytes.NewReader(bodyBf)
		}
		if hdObj.Method == "" {
			hdObj.Method = "GET"
		}
		req, err := http.NewRequest(strings.ToUpper(hdObj.Method), hdObj.Url, body)
		if err != nil {
			exitErr = fmt.Errorf("[fat] build request faild:%s,%v", err.Error(), hdObj)
			break
		}
		req.Header.Set("User-Agent", *ua)
		if hdObj.Header != nil {
			for k, v := range hdObj.Header {
				req.Header.Add(k, v)
			}
		}

		buf.ReadByte() //the last \n

		jobs <- req
	}
}

func bufPeed(reader *bufio.Reader, n int) ([]byte, error) {
	bs := make([]byte, n)
	var err error
	for i := 0; i < n; i++ {
		bs[i], err = reader.ReadByte()
		if err != nil {
			return bs, err
		}
	}
	return bs, nil

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
	lr := NewLogRequest()
	isSkip := false
	isOutRange := false
	var errOutOfRange = fmt.Errorf("urlCallWorker_OutOfRange")
	tryTimes := 0
	var conf *UrlCallConcConf
	var dealRequest = func(req *http.Request) error {
		tryTimes = 0

		id := atomic.AddUint64(&idx, 1)
		urlStr := req.URL.String()

		lr.reset()

		lr.addNotice("id", id)
		lr.addNotice("worker_id", workerId)
		lr.addNotice("method", req.Method)
		lr.addNotice("url", urlStr)

		conf = getCurConf()

		isSkip = conf.Start >= id
		isOutRange = workerId >= conf.ConcMax

		if isSkip {
			if id%100 == 0 {
				lr.print("[skip]")
			}
			return nil
		}

	httpClientTry:
		tryTimes++
		conf = getCurConf()
		startTime := time.Now()
		resp, err := client.Do(req)
		timeUsed := time.Now().Sub(startTime)

		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}

		lr.addNotice("try", fmt.Sprintf("%d/%d", tryTimes, conf.Retry))

		lr.addNotice("used_ms", fmt.Sprintf("%.3f", float64(timeUsed.Nanoseconds())/1e6))

		lr.addNotice("client_err", err)
		sucNum = 0
		if err == nil {
			if resp.StatusCode == 200 {
				sucNum = 1
			}

			lr.addNotice("http_code", resp.StatusCode)
			bd, r_err := ioutil.ReadAll(resp.Body)
			lr.addNotice("resp_len", len(bd))
			lr.addNotice("resp_err", r_err)
			respStr = string(bd)
			if !*noResp {
				lr.addNotice("resp_body", respStr)
			}
		} else {
			lr.addNotice("http_code", nil)
			lr.addNotice("resp_len", nil)
			lr.addNotice("resp_err", nil)
		}

		lr.addNotice("is_suc", sucNum)

		if sucNum > 0 {
			speedData.Success("request", 1)
			speedData.Success("resp_size", len(respStr))
			lr.print("info")
		} else {
			speedData.Fail("request", 1)
			lr.print("wf")
		}

		if sucNum < 1 && tryTimes < conf.Retry {
			time.Sleep(1 * time.Second)
			goto httpClientTry
		}

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
	buf  *bytes.Buffer
	data map[string]interface{}
	keys []string
}

func NewLogRequest() *logRequest {
	lr := &logRequest{
		buf: new(bytes.Buffer),
	}
	lr.reset()
	return lr
}

func (lr *logRequest) reset() {
	lr.buf.Reset()
	lr.data = make(map[string]interface{})
	lr.keys = make([]string, 0, 10)
}

func (lr *logRequest) addNotice(key string, val interface{}) {
	if val == nil {
		if _, has := lr.data[key]; has {
			delete(lr.data, key)
		}
		return
	}
	if _, has := lr.data[key]; !has {
		lr.keys = append(lr.keys, key)
	}
	lr.data[key] = val
}

func (lr *logRequest) print(ty string) {
	lr.buf.Reset()
	for _, key := range lr.keys {
		val, has := lr.data[key]
		if !has {
			continue
		}
		lr.buf.WriteString(key)
		lr.buf.WriteString("=")
		lr.buf.WriteString(url.QueryEscape(fmt.Sprintf("%v", val)))
		lr.buf.WriteString(" ")
	}
	log.Printf("[pid=%d][%s] %s \n", pid, ty, lr.buf.String())
}
