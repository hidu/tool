// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/5/17

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	flag.Parse()
	f := flag.Arg(0)
	if len(f) == 0 {
		log.Fatal("filename required")
	}
	bf, err := os.ReadFile(f)
	if err != nil {
		log.Fatalf("ReadFile(%q) %v", f, err)
	}
	fmt.Println(bf)
}
