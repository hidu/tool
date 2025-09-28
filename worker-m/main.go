// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/10/21

package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hidu/tool/worker-m/internal"
)

var config = flag.String("c", "./task.toml", "toml config path")
var listen = flag.String("l", ":9205", "listen addr")

var createAt = time.Now()

func main() {
	flag.Parse()
	cf := internal.LoadConfig(*config)

	runServer()

	ctx, cancel := getContext()
	defer cancel()

	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		log.Println("call /exit", r.RemoteAddr)
		cancel()
		tpl := template.Must(template.New("exit").Parse(exitHTML))
		values := map[string]any{
			"PID":      os.Getpid(),
			"CreateAt": createAt.Format(time.DateTime),
		}
		_ = tpl.Execute(w, values)
	})

	log.Fatalln("exit:", cf.Run(ctx))
}

const exitHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>Exit</title>
  <meta http-equiv="refresh" content="3;url=/">
</head>
<body>
  <p>PID={{.PID}}</p>
  <p>CreateAt={{.CreateAt}}</p>
  <p>exiting ...</p>
</body>
</html>
`

func getContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-ch
		cancel()
		time.Sleep(time.Second)
	}()

	return ctx, cancel
}

func runServer() {
	if *listen == "" || *listen == "no" {
		return
	}
	l, err := net.Listen("tcp", *listen)
	internal.Assert(err, "listen %s failed", *listen)
	log.Println("server listening on", l.Addr().String())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		txt := fmt.Sprintf("PID=%d\nCreateAt=%s", os.Getpid(), createAt.Format(time.DateTime))
		_, _ = w.Write([]byte(txt))
	})
	go func() {
		err = http.Serve(l, nil)
		internal.Assert(err, "http.Serve failed")
	}()
}
