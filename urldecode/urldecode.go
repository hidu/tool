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
	"unicode/utf8"
)

var version = "0.2 20160902"

var encodeStr = flag.Bool("e", false, "urlencode string")
var sword = flag.Bool("w", false, "split txt by word")

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
	if *sword {
		scanner := bufio.NewScanner(rd)
		scanner.Split(ScanWords)
		for scanner.Scan() {
			fmt.Print(QueryUnescape(scanner.Bytes()))
		}
	} else {
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

/////////=============

// isSpace reports whether the character is a Unicode white space character.
// We avoid dependency on the unicode package, but check validity of the implementation
// in the tests.
func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// ScanWords is a split function for a Scanner that returns each
// space-separated word of text, with surrounding spaces deleted. It will
// never return an empty string. The definition of space is set by
// unicode.IsSpace.
func ScanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
	}
	// Scan until space, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if isSpace(r) {
			return i + width, data[start : i+width], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}
