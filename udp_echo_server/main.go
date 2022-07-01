// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/7/1

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

var addr = flag.String("addr", ":8001", "server listen address")
var limit = flag.Int("l", 3, "limit")

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalf("ResolveUDPAddr(%s) failed: %v", *addr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("ListenUDP(%s) failed: %v", udpAddr.String(), err)
	}
	defer conn.Close()

	log.Println("Start UDPServer at", *addr)

	limiter := make(chan bool, *limit)
	for {
		limiter <- true
		go func() {
			handler(conn)
			<-limiter
		}()
	}
}

func handler(conn *net.UDPConn) {
	data := make([]byte, 1024)
	n, remote, err := conn.ReadFromUDP(data)
	if err != nil {
		fmt.Println("Failed To Read UDP Message, error: ", err)
		return
	}
	log.Println("Receive From ", remote.String(), "n=", n, "msg=", string(data[:n]))
	n, err = conn.WriteToUDP(data[:n], remote)
	log.Println("WriteToUDP", remote.String(), "n=", n, "err=", err)
}
