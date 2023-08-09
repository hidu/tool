package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"gopkg.in/yaml.v3"
)

func main() {
	var m any
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln("read toml from stdin failed:", err)
	}
	if err = json.Unmarshal(input, &m); err != nil {
		log.Fatalln("decode toml  failed:", err)
	}
	b, err := yaml.Marshal(m)
	if err != nil {
		log.Fatalln("encode to yaml failed:", err)
	}
	fmt.Println(string(b))
}
