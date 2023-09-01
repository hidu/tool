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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

var input = flag.String("f", "", `Input HTML file path or url.
read from stdin when it's empty`)

var nodePath = flag.String("xp", os.Getenv("Go_HTMLQuery_XP"), `HTML node's xpath.
default value from env  'Go_HTMLQuery_XP'`)

var timeout = flag.String("t", "10s", "Timeout for HTTP Requests")

func main() {
	flag.Parse()

	if *nodePath == "" {
		log.Fatalln("node's xpath ( -xp ) is required")
	}

	defer func() {
		if re := recover(); re != nil {
			log.Fatalln(re)
		}
	}()

	code, err := fetchContent()
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

func fetchContent() ([]byte, error) {
	if *input == "" {
		return io.ReadAll(os.Stdin)
	}

	if strings.HasPrefix(*input, "http://") || strings.HasPrefix(*input, "https://") {
		tm, err := time.ParseDuration(*timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", *timeout, err)
		}
		c := &http.Client{
			Timeout: tm,
		}
		resp, err := c.Get(*input)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(*input)
}
