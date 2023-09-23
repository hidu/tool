// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/9/23

package main

import (
	"context"
	"flag"
	"log"
	"net"

	"github.com/fsgo/fsgo/fsserver"

	"github.com/hidu/tool/go/tcp-tunnel/internal"
)

// var adminAddr=flag.String("admin",":8080","admin server addr to listen")
var tunnelOutAddr = flag.String("out", ":8081", "tunnel server outer addr to listen")
var tunnelInAddr = flag.String("in", ":8082", "tunnel server in addr to listen")

// var token=flag.String("token","","token")

func main() {
	flag.Parse()
	ec := make(chan error, 1)
	// go func() {
	// 	ec<-adminServer()
	// }()
	go func() {
		ec <- tunnelInServer()
	}()
	go func() {
		ec <- tunnelOutServer()
	}()
	err := <-ec
	log.Fatalln("exit with error:", err)
}

// func adminServer()error{
// 	mux:=&http.ServeMux{}
// 	mux.Handle("/auth", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		tk:=r.URL.Query().Get("token")
// 		if tk!=*token{
// 			w.WriteHeader(http.StatusBadRequest)
// 			_,_=w.Write([]byte("invalid token"))
// 			return
// 		}
// 	}))
// 	ser:=&http.Server{
// 		Handler: mux,
// 		Addr: *adminAddr,
// 	}
// 	log.Println("Listen adminServer at:",*adminAddr)
// 	return ser.ListenAndServe()
// }

var tunnelConns = make(chan net.Conn, 1000)

func tunnelOutServer() error {
	log.Println("Listen tunnelOutServer at:", *tunnelOutAddr)
	l, err := net.Listen("tcp", *tunnelOutAddr)
	if err != nil {
		return err
	}
	fs := &fsserver.AnyServer{
		Handler: fsserver.HandleFunc(func(ctx context.Context, conn net.Conn) {
			log.Println("new conn, remote=", conn.RemoteAddr().String())
			defer conn.Close()
			in := <-tunnelConns
			defer in.Close()
			_ = internal.NetCopy(in, conn)
		}),
	}
	return fs.Serve(l)
}

func tunnelInServer() error {
	log.Println("Listen tunnelInServer at:", *tunnelInAddr)
	l, err := net.Listen("tcp", *tunnelInAddr)
	if err != nil {
		return err
	}
	fs := &fsserver.AnyServer{
		Handler: fsserver.HandleFunc(func(ctx context.Context, conn net.Conn) {
			log.Println("new tunnel in conn,",
				"local=", conn.LocalAddr().String(),
				"remote=", conn.RemoteAddr().String(),
			)
			tunnelConns <- conn
		}),
	}
	return fs.Serve(l)
}
