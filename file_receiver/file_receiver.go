package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var port = flag.Int("port", 8111, "http server port")

func main() {
	flag.Parse()
	addr := fmt.Sprintf(":%d", *port)
	log.Println("Listen at:", addr)

	http.HandleFunc("/", fileReceiver)
	err := http.ListenAndServe(addr, nil)
	log.Println("exit,err=", err)
}

func fileReceiver(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		w.Write([]byte("<h1>Hello World!</h1>"))
		return
	}

	upFile, _, err := req.FormFile("file")
	if err != nil {
		w.WriteHeader(500)
		return
	}
	to := req.PostFormValue("to")
	info, err := os.Stat(to)
	if err == nil && info.IsDir() {
		w.WriteHeader(500)
		return
	}
	if os.IsNotExist(err) {
		parentDir := filepath.Dir(to)
		parentDirInfo, err := os.Stat(parentDir)
		if err == nil {
			if !parentDirInfo.IsDir() {
				log.Println("parent dir is not dir:", parentDir)
				w.WriteHeader(500)
				return
			}
		} else {
			if os.IsNotExist(err) {
				err := os.MkdirAll(parentDir, 0755)
				if err != nil {
					log.Printf("create parentDir[%s] failed,err:%v", parentDir, err)
					w.WriteHeader(500)
					return
				}
			} else {
				log.Printf("read parent dir [%s] stat failed,err:=%v", parentDir, err)
				w.WriteHeader(500)
				return
			}
		}
	} else {
		os.Remove(to)
	}
	toFile, err := os.OpenFile(to, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.Printf("create file [%s],failed,err=:%v", to, err)
		w.WriteHeader(500)
		return
	}
	defer toFile.Close()
	n, err := io.Copy(toFile, upFile)
	if err != nil {
		log.Println("upload file ", to, "failed,err=", err)
		w.WriteHeader(500)
		return
	}
	log.Println("upload file ", to, "suc,size=", n)
}
