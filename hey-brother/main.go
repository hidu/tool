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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fsgo/fsconf"
)

var hey = flag.String("hey", "hey", "hey command")
var total = flag.Int("n", 200, "Number of requests to run")
var workers = flag.String("c", "1", "Number of workers to run concurrently.")
var hp = flag.String("hp", "", "hey params")
var in = flag.String("conf", "./task.toml", `task file name`)
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

var cfg *Config

func loadConfig() {
	if err := fsconf.Parse(*in, &cfg); err != nil {
		log.Fatalln(err)
	}

	if len(cfg.Vars) > 0 {
		for k, v := range cfg.Vars {
			os.Setenv("var_"+k, v)
		}
		if err := fsconf.Parse(*in, &cfg); err != nil {
			log.Fatalln(err)
		}
	}
}

func getWorkers() []int {
	line := strings.Split(*workers, ",")
	var nums []int
	for _, str := range line {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		c, err := strconv.Atoi(str)
		if err != nil {
			log.Fatalf("parser %q failed: %v\n", str, err)
		}
		if c > 0 {
			nums = append(nums, c)
		}
	}
	if len(nums) == 0 {
		log.Fatalln("param -c is required")
	}
	return nums
}

func main() {
	flag.Parse()

	loadConfig()

	outName, outFile := getOutFile()
	defer outFile.Close()
	log.Println("out: ", outName)
	_, _ = fmt.Fprintf(outFile, specLine("From: "+*in+", Args: "+*hp)+"\n")

	conc := getWorkers()

	for i, task := range cfg.Tasks {
		for _, num := range conc {
			ps := []string{"-c", strconv.Itoa(num)}
			ret, err := task.Execute(ps)
			if err != nil {
				log.Println("has error:", err)
				continue
			}
			_, _ = fmt.Fprintf(outFile, ret.String()+"\n")

			if i < len(cfg.Tasks)-1 {
				time.Sleep(time.Duration(*s) * time.Second)
			}
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

func specLine(line string) string {
	data := map[string]any{
		"Special": line,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

type result struct {
	Name   string
	Args   []string
	URL    string
	Detail string
	QPS    float64
	Cost   float64
}

func (r *result) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

var qpsReg = regexp.MustCompile(`Requests/sec:\s+(\d+\.\d+)`)

type Config struct {
	Vars  map[string]string
	Tasks []*Task
}

type Task struct {
	Name   string
	Wait   string
	Before []string
	URL    string
}

func (ts *Task) Execute(args []string) (*result, error) {
	ts.executeBefore()
	return callHey(ts, args)
}

func callHey(task *Task, params []string) (*result, error) {
	var args []string
	if *hp != "" {
		args = append(args, strings.Fields(*hp)...)
	}
	if *total > 0 {
		args = append(args, "-n", strconv.Itoa(*total))
	}
	args = append(args, params...)

	args = append(args, task.URL)
	cmd := exec.Command(*hey, args...)
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
		Name: task.Name,
		URL:  task.URL,
		Args: args[:len(args)-1],
		QPS:  qps,
		Cost: cost.Seconds(),
	}
	if *detail {
		ret.Detail = string(content)
	}
	return ret, nil
}

var client = &http.Client{
	Timeout: 3 * time.Second,
}

func (ts *Task) executeBefore() {
	wait, _ := time.ParseDuration(ts.Wait)
	if wait <= 0 {
		wait = 3 * time.Second
	}
	for _, u := range ts.Before {
		resp, err := client.Get(u)
		if resp != nil {
			_ = resp.Body.Close()
		}
		log.Println("call before: ", u, "err:", err)
		time.Sleep(wait)
	}
}
