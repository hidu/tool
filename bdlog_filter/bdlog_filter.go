package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var conds = flag.String("cond", "", "filter condition,eg:   a>0")

var version = "0.1 20170407"

type CondItem struct {
	Key string
	Op  string
	Val float64
}

func (cond *CondItem) Match(val float64) bool {
	match := false
	switch cond.Op {
	case ">":
		match = val > cond.Val
		break
	case ">=":
		match = val >= cond.Val
		break
	case "<":
		match = val < cond.Val
		break
	case "<=":
		match = val <= cond.Val
		break
	case "=":
		match = val == cond.Val
		break
	case "<>":
		match = val != cond.Val
		break
	default:
		log.Println("not support operate:", cond.Op)
		break
	}
	return match
}

func init() {
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\ncat a.log.wf|bdlog_filter")
		fmt.Println("\n site: https://github.com/hidu/tool")
		fmt.Println(" version:", version)
	}
}

var cond *CondItem

func main() {
	flag.Parse()
	if *conds == "" {
		fmt.Fprint(os.Stderr, "filter is empty\n")
		os.Exit(1)
	}
	var err error
	cond, err = parseConds(*conds)
	if err != nil {
		fmt.Fprint(os.Stderr, "parse filter failed:"+err.Error()+"\n")
		os.Exit(1)
	}
	parselog(os.Stdin)
}

func parseConds(condStr string) (*CondItem, error) {
	r := regexp.MustCompile(`(\w+)([><=]{1,2})(\d+(.\d+)?)`)
	m := r.FindStringSubmatch(condStr)
	//["a>=4.0" "a" ">=" "4.0" ".0"]
	if len(m) < 3 {
		return nil, fmt.Errorf("parse cond failed")
	}
	result := &CondItem{
		Key: strings.TrimSpace(m[1]),
		Op:  strings.TrimSpace(m[2]),
	}
	v, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return nil, fmt.Errorf("parse cond val failed:%s", err.Error())
	}
	result.Val = v
	return result, nil
}

func parselog(rd io.Reader) {
	buf := bufio.NewReaderSize(rd, 81920)
	for {
		line, err := buf.ReadBytes('\n')
		if len(line) > 0 {
			printLine(line)
		}
		if err == io.EOF {
			break
		}
	}
}

func printLine(line []byte) {
	kv := parseLine(line)
	if kv == nil {
		log.Print("parse line failed:", string(line))
		return
	}
	v_f, _ := strconv.ParseFloat(kv.get(cond.Key), 64)

	if cond.Match(v_f) {
		fmt.Print(string(line))
	}
}

type logKv map[string]string

func (kv logKv) get(key string) string {
	if val, has := kv[key]; has {
		return val
	}
	return ""
}

var startKey = []byte("[")
var stopKey = []byte("]")
var kvSpace = []byte(" ")
var eqSign = []byte("=")

func parseLine(line []byte) logKv {
	m := bytes.LastIndex(line, startKey)
	if m < 1 {
		return nil
	}
	n := bytes.LastIndex(line, stopKey)
	if n < m {
		return nil
	}
	kv := make(logKv)

	sub := line[m+1 : n]
	items := bytes.Split(sub, kvSpace)

	for _, item := range items {
		x := bytes.Index(item, eqSign)
		if x < 1 {
			continue
		}
		key := string(item[:x])
		val := string(item[x+1:])
		kv[key] = val
	}

	return kv
}
