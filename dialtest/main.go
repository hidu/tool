// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/6/13

package main

import (
	"flag"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"log"
)

var addr = flag.String("addr", "127.0.0.1:8080", "address, e.g. 127.0.0.1:8080")
var conc = flag.Int("c", 10, "Number of multiple requests to make at a time")
var timeout = flag.Int("t", 10, "Millisecond to max")
var num = flag.Int("n", 100000, "Number of requests to perform")

func main() {
	flag.Parse()

	tasks := make(chan struct{}, *num)
	go func() {
		for i := 0; i < *num; i++ {
			tasks <- struct{}{}
		}
		close(tasks)
	}()

	dt := time.Duration(*timeout) * time.Millisecond

	var wg sync.WaitGroup

	var fail atomic.Int64
	var success atomic.Int64
	start:=time.Now()

	for i := 0; i < *conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range tasks {
				conn, err := net.DialTimeout("tcp", *addr, dt)
				if err != nil {
					fail.Add(1)
				} else {
					_ = conn.Close()
					success.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	cost:=time.Since(start)

	log.Println("success:",success.Load(),"fail:",fail.Load(),"cost:",cost.String())
}
