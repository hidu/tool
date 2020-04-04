package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var fields = flag.String("fs", "", "field names,eg:logid,uid")
var notShowKeys = flag.Bool("nokeys", false, "don't show keys")
var separator = flag.String("sep", "\t", "print split str")
var asJson = flag.Bool("json", false, "output as json")
var oldFormat = flag.Bool("old", false, "is old log format")

var printHeader = flag.Bool("h", false, "print header")

var version = "0.2 20160516"

func init() {
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\ncat a.log.wf|bdlog_kv")
		fmt.Println("\n site: https://github.com/hidu/tool")
		fmt.Println(" version:", version)
	}
}

var outFields []string

func main() {
	flag.Parse()
	if *fields == "" {
		fmt.Fprint(os.Stderr, "output fields is empty\n")
		os.Exit(1)
	}
	outFields = parseFields(*fields)

	if *printHeader {
		*notShowKeys = true
		fmt.Println(strings.Join(outFields, *separator))
	}

	parselog(os.Stdin)
}

func parseFields(fs string) (result []string) {
	ns := strings.Split(fs, ",")
	for _, name := range ns {
		name = strings.TrimSpace(name)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
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
		return
	}
	var vas []string
	var vmap = make(map[string]string)
	for _, key := range outFields {
		if key == "-" { // all line
			if *asJson {
				vmap["-"] = string(line)
			} else {
				vas = append(vas, string(line))
			}
		} else {
			if *asJson {
				vmap[key] = kv.get(key)
			} else {
				if !*notShowKeys {
					vas = append(vas, key)
				}
				vas = append(vas, kv.get(key))
			}
		}
	}
	if *asJson {
		bs, _ := json.Marshal(vmap)
		fmt.Println(string(bs))
	} else {
		fmt.Println(strings.Join(vas, *separator))
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

func parseLineOldFormat(line []byte) logKv {
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
