package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var leftFieldId = flag.Uint("fac", 0, "file_a's column id to compare. 0:all line(default),1:1st,2:2nd ...")
var leftFieldSep = flag.String("fas", "\t", "file_a field separator")

var rightFieldId = flag.Uint("fbc", 0, "file_b's column id to compare. 0:all line(default),1:1st,2:2nd ...")
var rightFieldSep = flag.String("fbs", "\t", "file_b field separator")

var fileAInFileB = flag.Bool("a_in_b", true, "file_a must in file_b")
var concatBLine = flag.Bool("concat_b", false, "when a_in_b=ture,concat file b's line")

var debug = flag.Bool("debug", false, "debug")
var reverse = flag.Bool("r", false, "file is reverse sort")
var number = flag.Bool("number", false, "compare as numbers")
var version = "20170528 0.1.1"

func init() {
	usage := flag.Usage
	flag.Usage = func() {
		usage()
		fmt.Println("\nUsage: fcomm [OPTION]... file_a file_b")
		fmt.Println("Compare sorted files file_a and file_b line by line.")
		fmt.Println("\nsite: https://github.com/hidu/tool")
		fmt.Println("Version:", version)
	}
}

type CommFile struct {
	Name string
	File *os.File
	//use which field to comp,-1:all line
	BufLine        chan *FileLine
	FieldID        int //-1:all line,
	FieldSeparator string
	LastLine       *FileLine
	LastLines      map[string]*FileLine
}

func NewCommFile(name string, fieldId uint, fieldSep string) (*CommFile, error) {
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	cf := &CommFile{
		Name:           name,
		File:           f,
		FieldID:        int(fieldId) - 1,
		FieldSeparator: fieldSep,
		BufLine:        make(chan *FileLine, 100),
	}
	go cf.Start()
	return cf, nil
}

func (f *CommFile) Start() {
	scaner := bufio.NewScanner(f.File)
	var lineNo int64
	for scaner.Scan() {
		lineNo++
		line := scaner.Text()
		fl, err := NewFileLine(line, lineNo, f.FieldID, f.FieldSeparator)

		if err != nil {
			log.Fatalln("parse file:", f.Name, ",failed,", err)
		}

		f.BufLine <- fl
	}
	close(f.BufLine)
}

func (f *CommFile) Close() error {
	var err error
	if f.File != nil {
		err = f.File.Close()
	}
	return err
}

func (f *CommFile) Next() (*FileLine, bool) {
	line, ok := <-f.BufLine
	f.LastLine = line
	return line, ok
}

type FileLine struct {
	Line     string
	NumValue float64
	LineNo   int64
	Raw      string
	IsRaw    bool
}

func NewFileLine(raw string, lineNo int64, fieldId int, fieldSep string) (*FileLine, error) {
	fl := &FileLine{
		IsRaw:  fieldId < 0,
		LineNo: lineNo,
	}

	if fl.IsRaw {
		fl.Line = raw
	} else {
		arr := strings.Split(raw, fieldSep)
		if len(arr) > fieldId {
			fl.Line = arr[fieldId]
		} else {
			return nil, fmt.Errorf("lineNo=%d,line=[%s] error", lineNo, raw)
		}
		if *number {
			v, err := strconv.ParseFloat(fl.Line, 64)
			if err != nil {
				return nil, err
			}
			fl.NumValue = v
		}
		fl.Raw = raw
	}
	return fl, nil
}

func (l *FileLine) Compare(other *FileLine) (v int) {
	if *number {
		if l.NumValue == other.NumValue {
			v = 0
		} else if l.NumValue > other.NumValue {
			v = 1
		} else {
			v = -1
		}
	} else {
		v = strings.Compare(l.Line, other.Line)
	}
	if *reverse {
		v *= -1
	}
	return v
}

func (l *FileLine) String() string {
	raw := l.Raw
	if l.IsRaw {
		raw = l.Line
	}
	if !*debug {
		return raw
	}
	return fmt.Sprintf("no = %.3d, line = [%s]", l.LineNo, raw)
}

func (l *FileLine) Empty() bool {
	if l.IsRaw {
		return l.Line == ""
	}
	return l.Raw == ""
}

func main() {
	flag.Parse()

	left, err := NewCommFile(flag.Arg(0), *leftFieldId, *leftFieldSep)
	checkError("left file", err)
	defer left.Close()

	right, err := NewCommFile(flag.Arg(1), *rightFieldId, *rightFieldSep)
	checkError("right file", err)
	defer right.Close()

	var aLast *FileLine
	var bLast *FileLine

	var bLastEqA *FileLine
	var bLastGtA *FileLine

	var lastC int

	debugFormat := "COMPARE: %s | <%10s> | c= % d %s\n"

	isBFinish := false

	//假设数据全部递增
	for {
		if a, aok := left.Next(); aok {
			if a.Empty() {
				continue
			}
			if bLastEqA != nil {
				c := a.Compare(bLastEqA)
				if *debug {
					fmt.Printf(debugFormat, a, "bLastEqA", c, bLastEqA)
				}
				lastC = c
				if c == 0 {
					compareAndPrint(a, true, bLastEqA)
					continue
				} else if c > 0 {
					bLastEqA = nil
				}
			}

			if bLastGtA != nil {
				c := a.Compare(bLastGtA)
				if *debug {
					fmt.Printf(debugFormat, a, "bLastGtA", c, bLastGtA)
				}
				lastC = c
				if c == 0 {
					compareAndPrint(a, true, bLastGtA)
					bLastEqA = bLastGtA
					bLastGtA = nil
					continue
				} else if c == -1 { //a < bLastGtA
					compareAndPrint(a, false, nil)
					continue
				} else {
					bLastGtA = nil
				}
			}

			if !isBFinish {
				for {
					if b, bok := right.Next(); bok {
						if b.Empty() {
							continue
						}

						if bLast != nil && b.Compare(bLast) == 0 {
							continue
						}

						c := a.Compare(b)

						if *debug {
							fmt.Printf(debugFormat, a, "b", c, b)
						}

						lastC = c
						if c == 0 {
							compareAndPrint(a, true, b)
							bLastEqA = b
							bLastGtA = nil
						} else if c == -1 { //a < b
							if bLastGtA == nil && bLastEqA == nil {
								compareAndPrint(a, false, nil)
							}
							bLastGtA = b
							//							bLastEqA = nil
							break
						} else { // now c == 1 a > b
							if lastC < 0 && aLast != nil && bLastGtA == nil {
								compareAndPrint(aLast, false, nil)
							}
							bLastEqA = nil
							bLastGtA = nil
						}
						bLast = b
					} else {
						isBFinish = true
						compareAndPrint(a, false, nil)
						break
					}
				}
			} else {
				compareAndPrint(a, false, nil)
			}

			aLast = a

			if *debug {
				fmt.Println("<" + strings.Repeat("-", 90))
			}
		} else {
			break
		}
	}
}

func compareAndPrint(a *FileLine, inBFile bool, b *FileLine) {
	if *fileAInFileB {
		if inBFile {
			if *concatBLine {
				fmt.Println(a, *leftFieldSep, b)
			} else {
				fmt.Println(a)
			}
		}
	} else {
		if !inBFile {
			fmt.Println(a)
		}
	}
}

func checkError(msg string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Error:\n\t%s:%s\n", msg, err))
		flag.Usage()
		os.Exit(1)
	}
}
