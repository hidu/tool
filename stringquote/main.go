// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/7/11

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln("read toml from stdin failed:", err)
	}
	str := strconv.Quote(string(input))
	fmt.Println(str)
}
