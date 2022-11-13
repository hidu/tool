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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var hp = flag.String("hp", "-c 1 -n 1", "hey params")

var in = flag.String("in", "task.txt", `task file name`)

var dir = flag.String("dir", "", "result file dir")

var out = flag.String("out", "", `result file name, 
when empty, will named with task file name and time, e.g: task1_2.txt.result.202209021543
when 'stdout', will print result to os.Stdout

support var:
{time} -> 202209021543
{in}   -> task1_2.txt
`)

var s = flag.Int("sleep", 10, "sleep x seconds after each")
var detail = flag.Bool("detail", true, "save hey result to content")

func main() {
	flag.Parse()
	content, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalln(err.Error())
	}

	outName, outFile := getOutFile()
	defer outFile.Close()
	log.Println("out: ", outName)
	_, _ = fmt.Fprintf(outFile, specLine("From: "+*in+", Args: "+*hp)+"\n")

	lines := strings.Split(string(content), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
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

func getOutFile() (string, io.WriteCloser) {
	name := *out
	if name == "stdout" {
		return name, os.Stdout
	}
	inName := filepath.Base(*in)

	nowStr := time.Now().Format("200601021504")

	if len(name) == 0 || name == "auto" {
		name = inName + ".result." + nowStr
	}

	name = strings.ReplaceAll(name, "{in}", inName)
	name = strings.ReplaceAll(name, "{time}", nowStr)

	if len(*dir) > 0 {
		_ = os.MkdirAll(*dir, 0777)
		name = filepath.Join(*dir, name)
	}
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln("open result file failed:", err)
	}
	return name, f
}

func callHey(name string, url string) (*result, error) {
	var args []string
	args = append(args, strings.Fields(*hp)...)
	args = append(args, url)
	cmd := exec.Command("hey", args...)
	log.Println("exec:", cmd.String())

	bf := &bytes.Buffer{}
	cmd.Stdout = bf

	start := time.Now()
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	cost := time.Since(start)

	content, err := io.ReadAll(bf)
	if err != nil {
		return nil, err
	}
	content = bytes.TrimSpace(content)
	matches := qpsReg.FindAllStringSubmatch(string(content), -1)
	qps, err := strconv.ParseFloat(matches[0][1], 64)
	if err != nil {
		return nil, err
	}
	ret := &result{
		Name:    name,
		URL:     url,
		QPS:     qps,
		Seconds: cost.Seconds(),
	}
	if *detail {
		ret.Detail = string(content)
	}
	return ret, nil
}

func specLine(line string) string {
	data := map[string]any{
		"Special": line,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

type result struct {
	Name    string
	URL     string
	Detail  string
	QPS     float64
	Seconds float64
}

func (r *result) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

var qpsReg = regexp.MustCompile(`Requests/sec:\s+(\d+\.\d+)`)
