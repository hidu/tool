// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/5/17

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
)

func main() {
	flag.Parse()
	f := flag.Arg(0)
	if f == "" {
		log.Fatalf("filename required")
	}
	bf, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatalf("ReadFile(%q) %v", f, err)
	}
	fmt.Println(bf)
}
