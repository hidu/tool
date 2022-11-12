package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hidu/tool/go-qqwry/qqwry"
)

var data = flag.String("db", "../data/qqwry.dat", "the qqwry.data path")
var port = flag.Int("port", 8510, "http port")

var qw *qqwry.QQwry

func main() {
	flag.Parse()
	qw = qqwry.NewQQwry(*data)
	if qw == nil {
		log.Println("init qqwry failed")
		return
	}
	defer qw.Close()
	http.HandleFunc("/", hander_index)
	http.HandleFunc("/ip", hander_ip)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	log.Println(err)
}

func hander_index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(indexHtml))
}

func hander_ip(w http.ResponseWriter, r *http.Request) {
	ip := strings.TrimSpace(r.FormValue("ip"))
	ips := strings.TrimSpace(r.FormValue("ips"))
	if len(ip) == 0 && len(ips) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("wrong params"))
		return
	}
	isShort := r.FormValue("short") == "1"

	start := time.Now()
	if len(ip) != 0 {
		res := qw.Search(ip)
		used := time.Since(start)
		log.Printf("remote:%s ip=%s country:%s used:%s\n", r.RemoteAddr, ip, res.Country, used.String())

		if isShort {
			w.Write([]byte(res.Country))
		} else {
			w.Write([]byte(res.String()))
		}
	} else if len(ips) != 0 {
		lines := strings.Split(ips, "\n")
		results := make([]string, 0)
		var area string
		for _, line := range lines {
			line := strings.TrimSpace(line)
			if len(line) != 0 {
				res := qw.Search(line)
				if isShort {
					area = res.Country
				} else {
					area = res.String()
				}
				results = append(results, fmt.Sprintf("%s\t%s", line, area))
			}
		}
		w.Write([]byte(strings.Join(results, "\n")))
	}
}

var indexHtml string = `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="author" content="duwei">
<title>ip search with qqwry</title>
<style>
body{margin:0 auto;width:90%}
</style>
</head>
<body>
<form action="/ip" method="get" target="ifr_tar">
ip:<input type="text" required name="ip">
short:<input type="checkbox" name="short" value="1">
<input type="submit" value="search">
<span style="color:gray;margin-left:100px">api: /ip?ip=127.0.0.1 or POST /ip ips=ip list</span>
</form>
<iframe src="about:blank" name="ifr_tar" style="width:100%;height:100px;border:none"></iframe>

<form action="/ip" method="post" target="ifr_tar1">
ip list:<br/>
<textarea name="ips" style="width:40%;height:200px"></textarea><br/>
short:<input type="checkbox" name="short" value="1"><br/>
<input type="submit" value="search">
</form>
<iframe src="about:blank" name="ifr_tar1" style="width:100%;height:100px;border:none" onload="this.style.height=(this.contentDocument.body.scrollHeight+120)+'px';"></iframe>
</body>
</html>
`
