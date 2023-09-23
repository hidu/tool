// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/9/23

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/hidu/tool/go/tcp-tunnel/internal"
)

// var adminAddr = flag.String("admin", "192.168.1.10:8080", "remote admin server addr")
var remoteAddr = flag.String("remote", "192.168.1.10:8081", "remote  tunnel server addr")
var localAddr = flag.String("local", "127.0.0.1:8080", "local server addr tunnel to")

// var token = flag.String("token", "", "token")
var worker = flag.Int("c", 3, "connections to keep")
var timeout = flag.Duration("t", 10*time.Second, "connect timeout")

func main() {
	flag.Parse()
	log.Println("tunnel client start:", os.Args)
	go startTunnelConn()
	for i := 0; i < *worker; i++ {
		go doTunnelConn()
	}
	select {}
}

var tunnelConns = make(chan net.Conn, 3)

func startTunnelConn() {
	var cf int
	for {
		conn, err := net.DialTimeout("tcp", *remoteAddr, *timeout)
		if err != nil {
			cf++
			log.Printf("connect to remote %s failed: %s\n", *remoteAddr, err.Error())
			wait(cf)
			continue
		}
		cf = 0
		tunnelConns <- conn
	}
}

func wait(n int) {
	if n < 10 {
		time.Sleep(200 * time.Millisecond)
		return
	}
	time.Sleep(500 * time.Millisecond)
}

var tunnelID atomic.Int64

func doTunnelConn() {
	for conn := range tunnelConns {
		func() {
			id := tunnelID.Add(1)
			start := time.Now()
			lc := getLocalConn()
			defer conn.Close()
			defer lc.Close()
			err := internal.NetCopy(conn, lc)
			cost := time.Since(start)
			log.Printf("tunnel id[%d] from[%s] to[%s] cost[%s], err: %v",
				id,
				conn.RemoteAddr().String(),
				lc.RemoteAddr().String(),
				cost.String(),
				err,
			)
		}()
	}
}

func getLocalConn() net.Conn {
	for i := 0; ; i++ {
		conn, err := net.DialTimeout("tcp", *localAddr, *timeout)
		if err == nil {
			return conn
		}
		wait(i)
		log.Printf("try[%d] connect to remote %s failed: %s\n", i, *remoteAddr, err.Error())
	}
}
