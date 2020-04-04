package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"strings"
	"time"
)

var urlStr = flag.String("url", "", "url to test")
var count = flag.Int("c", 10, "count")
var debug = flag.Bool("debug", false, "show debug detail")
var method = flag.String("method", "", "http method,eg:GET,POST,default is GET")
var post = flag.String("post", "", "post body string,eg a=1&b=1")
var post_body = flag.String("post_body", "", "post body in file")
var content_type = flag.String("ct", "application/x-www-form-urlencoded", "Content-Type")

const (
	METHOD_CONNECT string = "connect"
	METHOD_WRITE          = "write"
	METHOD_READ           = "read"
	METHOD_TOTAL          = "total"
)

func main() {
	flag.Parse()
	http_method := ""

	body := ""
	if *post != "" {
		http_method = "POST"
		body = *post
	}

	if *post_body != "" {
		ioutil.ReadFile(*post_body)
	}

	if *method != "" {
		http_method = *method
	}

	if http_method == "" {
		http_method = "GET"
	}
	if *urlStr == "" {
		fmt.Fprint(os.Stderr, "empty url to test\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	req, err := http.NewRequest(http_method, *urlStr, strings.NewReader(body))
	checkErr(err)
	if err != nil {
		os.Exit(2)
	}

	if body != "" {
		req.ContentLength = int64(len(body))
		if *content_type != "" {
			req.Header.Set("Content-Type", *content_type)
		}
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}

	tsArr := make([]*TimeMaps, 0)
	for index := 0; index < *count; index++ {
		log.Println("-------------------------", index, "----------------------------------")
		ts := TestCallUrl(req)
		tsArr = append(tsArr, ts)
	}
	printTsResult(tsArr)
}
func printTsResult(tsArr []*TimeMaps) {
	line := strings.Repeat("=", 80)

	fmt.Println("")
	fmt.Println(line)
	fmt.Println("Detail:")
	fmt.Println(line)
	header := "index\tconnect(ms)\twrite(ms)\tread(ms)\ttotal(ms)\thttp_code\n" + strings.Repeat("-", 80)
	fmt.Println(header)
	stpl := "%-5s\t %-11s\t %-9s\t %-8s\t %-9s\t %-9s\n"

	methods := []string{METHOD_CONNECT, METHOD_WRITE, METHOD_READ, METHOD_TOTAL}
	res := make(map[string][]float64)
	for _, method := range methods {
		res[method] = make([]float64, 0)
	}

	for index, ts := range tsArr {
		for _, method := range methods {
			us := ts.Used(method)
			if us > 0 {
				res[method] = append(res[method], us)
			}
		}
		fmt.Printf(stpl, fmt.Sprintf("%d", index), ts.UsedStr(METHOD_CONNECT), ts.UsedStr(METHOD_WRITE), ts.UsedStr(METHOD_READ), ts.UsedStr(METHOD_TOTAL), fmt.Sprintf("%d", ts.statusCode))
	}

	fmt.Println("")
	fmt.Println(line)
	fmt.Println("Statis:")
	fmt.Println(line)
	fmt.Println(header)

	fns := []*MyFn{
		NewMyFn("suc", func(fl []float64) float64 {
			return float64(len(fl))
		}),
		NewMyFn("failed", func(fl []float64) float64 {
			return float64(*count - len(fl))
		}),
		NewMyFn("empty", Print_Empty),
		NewMyFn("min", Math_Min),
		NewMyFn("min", Math_Min),
		NewMyFn("max", Math_Max),
		NewMyFn("avg", Math_Avg),
		NewMyFn("empty", Print_Empty),
	}

	pers := []int{60, 70, 80, 90, 95, 99}
	for _, n := range pers {
		fn := NewMyFn(fmt.Sprintf("per_%d", n), func(fl []float64) float64 {
			return Math_Percent(fl, n)
		})
		fns = append(fns, fn)
	}

	for _, myfn := range fns {
		if myfn.name == "empty" {
			myfn.fn([]float64{0})
			continue
		}
		fn := func(fl []float64) string {
			return fmt.Sprintf("%.2f", myfn.fn(fl))
		}
		fmt.Printf(stpl, myfn.name, fn(res[METHOD_CONNECT]), fn(res[METHOD_WRITE]), fn(res[METHOD_READ]), fn(res[METHOD_TOTAL]), "-1")
	}

	fmt.Println("")
}

func TestCallUrl(req *http.Request) *TimeMaps {
	splitLine := strings.Repeat("=", 80)
	hostInfo := strings.Split(req.Host, ":")

	host := hostInfo[0]
	port := "80"
	if len(hostInfo) == 2 {
		port = hostInfo[1]
	}

	addr := host + ":" + port
	bf := BuildReq(req)
	// 	dumpS:=bf.String()
	if *debug {
		// 		dumpS= strings.Replace(strings.Replace(dumpS, "\r", "\\r", -1), "\n", "\\n", -1)
		fmt.Printf("http request:\n"+splitLine+"\n%s\n"+splitLine+"\n", bf.String())
	}

	ts := NewTimeMaps()

	log.Println("addr :", addr)

	t0 := ts.Start(METHOD_TOTAL)

	t1 := ts.Start(METHOD_CONNECT)
	conn, err := net.Dial("tcp", addr)
	t1.Stop(err)
	if err != nil {
		return ts
	}
	defer conn.Close()

	t2 := ts.Start(METHOD_WRITE)
	n, err := conn.Write(bf.Bytes())
	t2.Stop(err)
	log.Println("write:", n, "byte")
	if err != nil {
		return ts
	}

	t3 := ts.Start(METHOD_READ)
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		defer resp.Body.Close()
	}
	if *debug {
		dumpRes, _ := httputil.DumpResponse(resp, true)
		log.Printf("response\n"+splitLine+"\n%s\n"+splitLine+"\n", string(dumpRes))
	}
	bd, err := ioutil.ReadAll(resp.Body)
	t3.Stop(err)
	if err != nil {
		return ts
	}

	ts.statusCode = resp.StatusCode
	ts.bodyLen = len(bd)

	log.Println("resp :", len(bd), "byte,status:", resp.Status)

	t0.Stop(nil)
	return ts
}

func BuildReq(req *http.Request) *bytes.Buffer {
	rn := "\r\n"
	bf := bytes.NewBufferString(req.Method + " " + req.URL.String() + " HTTP/1.1" + rn)
	bf.WriteString(fmt.Sprintf("%s: %s%s", "Host", req.Host, rn))
	for key, vs := range req.Header {
		for _, v := range vs {
			bf.WriteString(fmt.Sprintf("%s: %s%s", key, v, rn))
		}
	}
	bf.WriteString(rn)
	if req.Body != nil {
		bd, _ := ioutil.ReadAll(req.Body)
		bf.Write(bd)
	}
	return bf
}

type TimeMaps struct {
	tms        map[string]*MyTimer
	statusCode int
	bodyLen    int
}

func NewTimeMaps() *TimeMaps {
	t := &TimeMaps{
		tms: make(map[string]*MyTimer),
	}
	return t
}

func (t TimeMaps) Start(name string) *MyTimer {
	log.Println("start:", name)
	myt := &MyTimer{
		start: time.Now(),
		name:  name,
	}
	t.tms[name] = myt
	return myt
}

func (t TimeMaps) Used(name string) float64 {
	if ts, has := t.tms[name]; has {
		return ts.Used()
	}
	return -1
}

func (t TimeMaps) UsedStr(name string) string {
	u := t.Used(name)
	return fmt.Sprintf("%.2f", u)
}

type MyTimer struct {
	start time.Time
	end   time.Time
	name  string
	suc   bool
}

func (m *MyTimer) Stop(err error) {
	m.suc = err == nil
	m.end = time.Now()
	log.Println("stop :", m.name, "used:", m.Used(), "ms", "err:", err)
}

func (m *MyTimer) Used() float64 {
	if !m.suc {
		return -1
	}
	return float64(m.end.Sub(m.start).Nanoseconds()) / 1e6
}

func checkErr(err error) {
	if err != nil {
		log.Println("error:", err)
	}
}

type MyFn struct {
	name string
	fn   func(fl []float64) float64
}

func NewMyFn(name string, fn func(fl []float64) float64) *MyFn {
	return &MyFn{
		name: name,
		fn:   fn,
	}
}

func Math_Max(fl []float64) float64 {
	var m float64 = -1
	for _, v := range fl {
		if m < 0 {
			m = v
		} else if v > m {
			m = v
		}
	}
	return m
}

func Math_Min(fl []float64) float64 {
	var m float64 = -1
	for _, v := range fl {
		if m < 0 {
			m = v
		} else if v < m {
			m = v
		}
	}
	return m
}
func Math_Avg(fl []float64) float64 {
	var t float64 = 0
	for _, v := range fl {
		t += v
	}
	return t / float64(len(fl))
}

func Math_Percent(fl []float64, percent int) float64 {
	if len(fl) == 0 {
		return 0
	}
	sort.Float64s(fl)
	n := int(float64(percent) / 100.0 * float64(len(fl)))
	if n < 1 {
		n = 1
	}
	return fl[n-1]
}

func Print_Empty(fl []float64) float64 {
	fmt.Println("")
	return -1
}
