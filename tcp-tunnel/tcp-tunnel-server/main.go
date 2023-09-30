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
	"sync/atomic"
	"time"

	"github.com/fsgo/fsgo/fsserver"

	"github.com/hidu/tool/go/tcp-tunnel/internal"
)

var tunnelOutAddr = internal.FlagEnvString("out", "TT_S_out", "127.0.0.1:8100", "addr export")
var tunnelInAddr = internal.FlagEnvString("in", "TT_S_in", ":8090", "addr for tunnel client")
var useTLS = internal.FlagEnvBool("tls", "TT_S_tls", true, "enable tls")
var drop = internal.FlagEnvDuration("idle", "TT_S_idle", 3*time.Minute, "drop idle connections")

func main() {
	flag.Parse()
	ec := make(chan error, 1)
	go func() {
		ec <- tunnelInServer()
	}()
	go func() {
		ec <- tunnelOutServer()
	}()
	err := <-ec
	log.Fatalln("exit with error:", err)
}

var tunnelConns = make(chan net.Conn, 100)

var dropID atomic.Int64

func tunnelOutServer() error {
	log.Println("Listen tunnelOutServer at:", *tunnelOutAddr)
	l, err := net.Listen("tcp", *tunnelOutAddr)
	if err != nil {
		return err
	}
	var lastUse atomic.Int64
	fs := &fsserver.AnyServer{
		Handler: fsserver.HandleFunc(func(ctx context.Context, conn net.Conn) {
			log.Println("new conn, remote=", conn.RemoteAddr().String())
			defer conn.Close()
			in := <-tunnelConns
			lastUse.Store(time.Now().Unix())
			defer in.Close()
			_ = internal.NetCopy(in, conn)
		}),
	}

	tm := time.NewTimer(time.Second)
	checkDrop := func() {
		defer tm.Reset(time.Second)
		dur := time.Duration(time.Now().Unix()-lastUse.Load()) * time.Second
		if dur < *drop {
			return
		}

		defer lastUse.Store(time.Now().Unix())
		for i := 0; i < len(tunnelConns); i++ {
			select {
			case in := <-tunnelConns:
				id := dropID.Add(1)
				e2 := in.Close()
				log.Println("drop idle connection, ",
					"drop_total=", id,
					"idle_duration", dur.String(),
					"local=", in.LocalAddr().String(),
					"remote=", in.RemoteAddr().String(),
					"close=", e2,
				)

			default:
				log.Println("no connections when check idle")
				return
			}
		}
	}

	go func() {
		for {
			<-tm.C
			checkDrop()
		}
	}()
	return fs.Serve(l)
}

// tunnelInServer 接收转发的流量
func tunnelInServer() error {
	log.Println("Listen tunnelInServer at:", *tunnelInAddr)
	var l net.Listener
	var err error
	if *useTLS {
		l, err = tls.Listen("tcp", *tunnelInAddr, internal.ServerTlsConfig)
	} else {
		l, err = net.Listen("tcp", *tunnelInAddr)
	}
	if err != nil {
		return err
	}
	var connID atomic.Int64
	fs := &fsserver.AnyServer{
		Handler: fsserver.HandleFunc(func(ctx context.Context, conn net.Conn) {
			id := connID.Add(1)
			log.Println("new tunnel in conn,",
				"id=", id,
				"local=", conn.LocalAddr().String(),
				"remote=", conn.RemoteAddr().String(),
			)
			tunnelConns <- conn
		}),
	}
	return fs.Serve(l)
}
