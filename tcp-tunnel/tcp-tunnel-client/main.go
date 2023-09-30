// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/9/23

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/hidu/tool/go/tcp-tunnel/internal"
)

var remoteAddr = internal.FlagEnvString("remote", "TT_C_remove", "192.168.1.10:8090", "remote  tunnel server addr")
var localAddr = internal.FlagEnvString("local", "TT_C_local", "127.0.0.1:8080", "local server addr tunnel to")

var worker = internal.FlagEnvInt("c", "TT_C_c", 3, "connections to keep")
var timeout = internal.FlagEnvDuration("t", "TT_C_t", 300*time.Second, "connect timeout")
var useTLS = internal.FlagEnvBool("tls", "TT_C_tls", true, "enable tls")

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
	var dialer1 internal.Dialer
	if *useTLS {
		dialer1 = &tls.Dialer{
			NetDialer: &net.Dialer{},
			Config:    internal.ClientTlsConfig,
		}
	} else {
		dialer1 = &net.Dialer{}
	}
	var cf int
	for {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		conn, err := dialer1.DialContext(ctx, "tcp", *remoteAddr)
		cancel()
		cost := time.Since(start)

		if err != nil {
			cf++
			log.Printf("connect to remote %s failed: %s, cost=%s\n", *remoteAddr, err.Error(), cost.String())
			wait(cf)
			continue
		}
		log.Printf("connect to remote %s, cost=%s\n", *remoteAddr, cost.String())
		cf = 0
		tunnelConns <- conn
	}
}

func wait(n int) {
	if n < 10 {
		time.Sleep(200 * time.Millisecond)
		return
	}
	time.Sleep(time.Second)
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
			internal.SetConnFlags(conn)
			return conn
		}
		wait(i)
		log.Printf("try[%d] connect to remote %s failed: %s\n", i, *remoteAddr, err.Error())
	}
}
