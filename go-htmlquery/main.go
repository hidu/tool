// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/9/1

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/antchfx/htmlquery"
)

var input = flag.String("f", "", "input file path. read from stdin when it's empty")

var nodePath = flag.String("xp", os.Getenv("Go_HTMLQuery_XP"), "node's xpath. default value from env  'Go_HTMLQuery_XP'")

func main() {
	flag.Parse()
	if *nodePath == "" {
		log.Fatalln("with empty -xp")
	}

	var code []byte
	var err error

	if *input == "" {
		code, err = io.ReadAll(os.Stdin)
	} else {
		code, err = os.ReadFile(*input)
	}
	if err != nil {
		log.Fatalln(err)
	}
	doc, err := htmlquery.Parse(bytes.NewReader(code))
	if err != nil {
		log.Fatalln(err)
	}
	node := htmlquery.FindOne(doc, *nodePath)
	fmt.Println(htmlquery.InnerText(node))
}
