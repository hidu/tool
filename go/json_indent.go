package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hidu/goutils/object"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

var usenum = flag.Bool("n", true, "use number")

var need_indent = flag.Bool("i", true, "indent")

var fields = flag.String("fields", "", "out put only fields")

var allFile = flag.Bool("full", false, "read full file")

var reges []*regexp.Regexp
var fieldNames []string

func main() {
	flag.Parse()

	if *fields != "" {
		fieldsArr := strings.Split(*fields, ",")
		for _, fieldName := range fieldsArr {
			fieldName = strings.TrimSpace(fieldName)
			if fieldName != "" {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}
	jsonFile := os.Stdin

	if *allFile {
		buf, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			log.Println(err)
		} else {
			indent(buf)
		}
	} else {
		buf := bufio.NewReaderSize(jsonFile, 81920000)
		for {
			line, err := buf.ReadBytes('\n')
			indent(line)
			if err == io.EOF {
				break
			}
		}
	}
}

func init() {
	regs := []string{`\x1b\[\d+;\d+m`, `\x1b\[\d+m`}
	for _, regStr := range regs {
		reg := regexp.MustCompile(regStr)
		reges = append(reges, reg)
	}
}

func indent(json_byte []byte) {
	if len(json_byte) == 0 {
		return
	}
	for _, reg := range reges {
		json_byte = reg.ReplaceAll(json_byte, []byte{})
	}
	var obj interface{}
	dec := json.NewDecoder(bytes.NewReader(json_byte))

	if *usenum {
		dec.UseNumber()
	}
	err := dec.Decode(&obj)
	if err != nil {
		log.Println("json decode failed:", err, "\ndata-->len=", len(json_byte), "\n", string(json_byte), "\n<--")
		return
	}

	if len(fieldNames) != 0 {
		walker := object.NewInterfaceWalker(obj)
		for _, fieldName := range fieldNames {
			v, _ := walker.GetString(fieldName)
			fmt.Printf("%s=%v\t", fieldName, v)

		}
		fmt.Println()
		return
	}

	data_byte, err := json.Marshal(obj)
	if err != nil {
		log.Println("json encode failed:", err, "\ndata-->\n:", string(json_byte), "\n<--")
		return
	}

	if !*need_indent {
		fmt.Println(string(data_byte))
		return
	}

	var out bytes.Buffer
	json.Indent(&out, data_byte, "", "    ")
	out.WriteString("\n")
	out.WriteTo(os.Stdout)
}
