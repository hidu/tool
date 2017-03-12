/**
*for http network test
 */
package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var reqid uint64

type resData struct {
	ID      uint64 `json:"id"`
	Request string `json:"request'`
}
type Datas struct {
	ResData []*resData
}

func HelloServer(w http.ResponseWriter, req *http.Request) {
	id := atomic.AddUint64(&reqid, 1)
	item := new(resData)
	item.ID = id
	dump, err := httputil.DumpRequest(req, true)

	if err != nil {
		item.Request = "error:" + err.Error()
	} else {
		item.Request = string(dump)
	}

	sleep := getIntVal(req, "sleep")
	if sleep > 0 {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}

	http_code := getIntVal(req, "http_code")
	if http_code > 0 {
		w.WriteHeader(http_code)
	}
	content_type := req.FormValue("content_type")

	repeat_num := getIntVal(req, "repeat")
	if repeat_num == 0 {
		repeat_num = 1
	}

	datas := new(Datas)

	for i := 0; i < repeat_num; i++ {
		datas.ResData = append(datas.ResData, item)
	}

	if req.FormValue("broken") != "" {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
	}

	var dataBf []byte
	switch req.FormValue("type") {
	case "json":
		content_type = "application/json"
		dataBf, _ = json.MarshalIndent(datas, "", " ")
	case "xml":
		content_type = "text/xml"
		dataBf, _ = xml.MarshalIndent(datas, "", " ")
	default:
		dataBf = []byte(fmt.Sprintf("%q", datas))
	}

	if content_type != "" {
		w.Header().Set("Content-Type", content_type)
	}

	w.Write(dataBf)

}

func getIntVal(req *http.Request, key string) int {
	val := req.FormValue(key)
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return n
}

var addr = flag.String("addr", ":8088", "http server listen at")

func main() {
	flag.Parse()
	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/help", HelpServer)
	fmt.Println("start http server at:", *addr)
	err := http.ListenAndServe(*addr, nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func HelpServer(w http.ResponseWriter, req *http.Request) {
	help := `
	params:
	
	sleep        : sleep ms,eg:100
	http_code    : http status code, eg:500
	content_type : content type, eg: text/html;chatset=utf-8
	repeat       : repeat content times, eg:10
	broken       : broken this connect,eg borken=1
	type         : data output type,allow:[json,xml] 
	
	eg:
	http://{host}/?sleep=100
	http://{host}/?sleep=100&http_code=500&repeat=1
	`
	help = strings.Replace(help, "{host}", req.Host, -1)
	w.Write([]byte(help))
}
