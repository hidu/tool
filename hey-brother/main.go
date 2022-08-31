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
	"time"
)

var hp = flag.String("hp", "-c 1 -n 1", "hey params")

var in = flag.String("in", "task.txt", "task file")
var out = flag.String("out", "", "result file")

var s = flag.Int("sleep", 0, "sleep x seconds after each")
var detail = flag.Bool("detail", true, "save hey result to content")

func main() {
	flag.Parse()
	content, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalln(err.Error())
	}

	outName,outFile := getOutFile()
	defer outFile.Close()
	log.Println("out=",outName)
	_, _ = fmt.Fprintf(outFile, specLine("From "+outName)+"\n")

	lines := strings.Split(string(content), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		if line[0]=='#'{
			_, _ = fmt.Fprintf(outFile, specLine(line)+"\n")
			continue
		}
		arr := strings.Fields(line)
		log.Printf("line %d: %q\n", i+1, arr)
		if len(arr) != 2 {
			log.Println("ignored")
			continue
		}
		ret, err := callHey(arr[0], arr[1])
		if err != nil {
			log.Println("has error:", err)
			continue
		}
		_, _ = fmt.Fprintf(outFile, ret.String()+"\n")
		if i < len(lines)-1 {
			time.Sleep(time.Duration(*s) * time.Second)
		}
	}
}

func getOutFile() (string,io.WriteCloser) {
	name := *out
	if name == "stdout" {
		return name,os.Stdout
	}

	if len(name) == 0 || name == "auto" {
		name = *in + ".result." + time.Now().Format("200601021504")
	}
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln("open result file failed:", err)
	}
	return name,f
}

func callHey(name string, url string) (*result, error) {
	var args []string
	args = append(args, strings.Fields(*hp)...)
	args = append(args, url)
	cmd := exec.Command("hey", args...)
	log.Println("exec:", cmd.String())

	bf := &bytes.Buffer{}
	cmd.Stdout = bf

	start:=time.Now()
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	cost:=time.Since(start)
	
	content, err := io.ReadAll(bf)
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
		Name: name,
		URL:  url,
		QPS:  qps,
		Seconds: cost.Seconds(),
	}
	if *detail {
		ret.Detail = string(content)
	}
	return ret, nil
}


func specLine(line string)string{
	data:=map[string]interface{}{
		"Special":line,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

type result struct {
	Name   string
	URL    string
	QPS    float64
	Seconds float64
	Detail string
}

func (r *result) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

var qpsReg = regexp.MustCompile(`Requests/sec:\s+(\d+\.\d+)`)
