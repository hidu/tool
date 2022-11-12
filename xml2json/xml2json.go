package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/clbanning/x2j"
	"github.com/hidu/goutils/json_util"
)

const VERSION = "20150712"

var xmlName = flag.String("xml", "", "xml file path")
var outJson = flag.String("out", "", "json output file path")
var schemaPath = flag.String("jsonschema", "", "json schema file path")

func init() {
	d := flag.Usage
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Convert XML to JSON")
		fmt.Fprintln(os.Stderr, "version:", VERSION)
		fmt.Fprintln(os.Stderr, "site:", "https://github.com/hidu/tool/")
		d()
	}
}

func main() {
	flag.Parse()

	jsonFile := os.Stdin
	var err error

	if len(*xmlName) != 0 {
		jsonFile, err = os.Open(*xmlName)
		checkErr(err)
	}

	if len(*outJson) != 0 {
		err := checkOutFilePath(*outJson)
		checkErr(err)
	}

	jsonStr, err := x2j.ToJson(jsonFile)
	checkErr(err)

	var jsonData any
	json.Unmarshal([]byte(jsonStr), &jsonData)

	if len(*schemaPath) != 0 {
		schema, err := loadJsonFile(*schemaPath)
		checkErr(err)

		jsonData, err = json_util.FixDataWithSchema(jsonData, schema)
		checkErr(err)
	}

	jsonBs, err := json.MarshalIndent(jsonData, "", "  ")
	checkErr(err)
	if len(*outJson) != 0 {
		os.WriteFile(*outJson, jsonBs, 0664)
	} else {
		fmt.Println(string(jsonBs))
	}
}

func checkErr(err error) {
	if err == nil {
		return
	}
	log.Fatalln(err)
}

func loadJsonFile(jsonPath string) (data any, err error) {
	jsonBs, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBs, &data)
	return
}

func checkOutFilePath(outPath string) error {
	info, err := os.Stat(outPath)
	if os.IsExist(err) {
		if info.IsDir() {
			return errors.New("outpath exist and is dir")
		}
	}
	dirPath := filepath.Dir(outPath)
	_, dirErr := os.Stat(dirPath)
	if os.IsNotExist(dirErr) {
		return os.MkdirAll(dirPath, os.ModePerm)
	}
	return nil
}
