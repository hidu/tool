// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/8/21

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var hp = flag.String("hp", "-c 1 -n 1", "hey params")
var in = flag.String("in", "task.txt", "task file")

func main() {
	flag.Parse()
	content, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalln(err.Error())
	}

	lines := strings.Split(string(content), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		arr := strings.Fields(line)
		if len(arr) != 2 {
			log.Println("ignore line", i+1, ":", line)
			continue
		}
		ret, err := callHey(arr[0], arr[1])
		if err != nil {
			log.Println("has error:", err)
			continue
		}
		fmt.Println(ret.String())
	}
}

func callHey(name string, url string) (*result, error) {
	var args []string
	args = append(args, strings.Fields(*hp)...)
	args = append(args, url)
	cmd := exec.Command("hey", args...)
	log.Println("exec:", cmd.String())

	out := &bytes.Buffer{}
	cmd.Stdout = out

	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(out)
	if err != nil {
		return nil, err
	}
	content = bytes.TrimSpace(content)
	matches := qpsReg.FindAllStringSubmatch(string(content), -1)
	qps, err := strconv.ParseFloat(matches[0][1], 10)
	if err != nil {
		return nil, err
	}
	ret := &result{
		Name:   name,
		URL:    url,
		QPS:    qps,
		Detail: string(content),
	}
	return ret, nil
}

type result struct {
	Name   string
	URL    string
	QPS    float64
	Detail string
}

func (r *result) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

var qpsReg = regexp.MustCompile(`Requests/sec:\s+(\d+\.\d+)`)
