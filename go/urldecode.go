package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
)

var encodeStr = flag.Bool("e", false, "urlencode string")
var version = "0.1 20151029"

func init() {
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\ncat a.log|urldecode|grep xxx")
		fmt.Println("urldecode -e \"你好\"")
		fmt.Println("\n site: https://github.com/hidu/tool")
		fmt.Println(" version:", version)
	}
}

func main() {
	flag.Parse()
	if *encodeStr {
		str := flag.Arg(0)
		if str == "" {
			bf, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprint(os.Stderr, "read from stdin err:", err)
				os.Exit(2)
			}
			str = string(bf)
		}
		fmt.Print(url.QueryEscape(str))
		os.Exit(0)
	}
	unescape(os.Stdin)
}

func unescape(rd io.Reader) {
	buf := bufio.NewReaderSize(rd, 81920)
	for {
		line, err := buf.ReadBytes('\n')
		if len(line) > 0 {
			fmt.Print(QueryUnescape(line))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "urldecode_error:", err)
		}
	}
}

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

var encodeQueryComponent = 4
var mode = 4

var bp = []byte("%")
var bplus = []byte("+")

var flagEncodeQueryComponent = mode == encodeQueryComponent

var unescapeBuf = make([]byte, 1024*1024)

func QueryUnescape(s []byte) string {
	// Count %, check that they're well-formed.

	n := bytes.Index(s, bp)
	hasPlus := flagEncodeQueryComponent && bytes.Index(s, bplus) > -1

	if n < 0 && !hasPlus {
		return string(s)
	}
	slen := len(s)
	if slen > len(unescapeBuf) {
		unescapeBuf = make([]byte, slen+1024)
	}
	j := 0
	for i := 0; i < slen; {
		switch s[i] {
		case '%':
			if i+2 >= slen || !ishex(s[i+1]) || !ishex(s[i+2]) {
				unescapeBuf[j] = s[i]
				j++
				i++
				break
			}
			unescapeBuf[j] = unhex(s[i+1])<<4 | unhex(s[i+2])
			j++
			i += 3
		case '+':
			if flagEncodeQueryComponent {
				unescapeBuf[j] = ' '
			} else {
				unescapeBuf[j] = '+'
			}
			j++
			i++
		default:
			unescapeBuf[j] = s[i]
			j++
			i++
		}
	}
	return string(unescapeBuf[:j])
}
