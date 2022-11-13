// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/7/1

package main

import (
	"flag"
	"log"
	"net"
	"strconv"
)

var addr = flag.String("addr", "127.0.0.1:8001", "server address")
var num = flag.Int("n", 10, "send message total")

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalf("ResolveUDPAddr(%s) failed: %v", *addr, err)
	}
	socket, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatalf("DialUDP(%s) failed: %v", udpAddr.String(), err)
	}
	defer socket.Close()
	for i := 0; i < *num; i++ {
		msg := "hello:" + strconv.Itoa(i)

		wrote, err := socket.Write([]byte(msg))
		log.Println("socket.Write n=", wrote, " err=", err)

		data := make([]byte, 4096)
		n, remoteAddr, err := socket.ReadFromUDP(data)
		log.Println("ReadFromUDP n=", n, " remoteAddr=", remoteAddr.String(), "err=", err, "data=", string(data[:n]))
	}
}
