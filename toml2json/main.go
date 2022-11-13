// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/5/26

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

func main() {
	var m any
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln("read toml from stdin failed:", err)
	}
	if _, err = toml.Decode(string(input), &m); err != nil {
		log.Fatalln("decode toml  failed:", err)
	}
	b, err := json.MarshalIndent(m, " ", " ")
	if err != nil {
		log.Fatalln("encode to json failed:", err)
	}
	fmt.Println(string(b))
}
