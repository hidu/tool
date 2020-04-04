package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hidu/go-speed"
)

var version = "0.2 20190217"

var k = flag.Int("k", 0, "count QPS via a key")
var v = flag.Int("v", 0, "count QPS via a value,the value must be a int")
var sep = flag.String("t", " ", "use SEP instead of non-blank to blank transition")
var s = flag.Int("s", 2, "count qps each 's' second")

func init() {
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\ncat a.log|qps_count")
		fmt.Println("\n site: https://github.com/hidu/tool")
		fmt.Println(" version:", version)
	}
}

func main() {
	flag.Parse()

	q := NewQPSCount(*k, *v, *sep, *s)
	defer q.Close()

	q.Start(os.Stdin)
}

func NewQPSCount(k int, v int, sep string, sec int) *QPSCount {
	q := &QPSCount{
		k:   k,
		v:   v,
		sep: sep,
	}
	q.sp = speed.NewSpeed("qps", sec, q.Print)
	return q
}

type QPSCount struct {
	sp  *speed.Speed
	k   int
	v   int
	sep string
	n   int
}

func (q *QPSCount) Start(rd io.Reader) {
	re := regexp.MustCompile(`\s+`)

	buf := bufio.NewReaderSize(rd, 81920)
	for {
		line, err := buf.ReadBytes('\n')
		lineStr := strings.TrimSpace(string(line))
		if len(lineStr) > 0 {
			q.sp.Success("LINE", 1)

			if q.k > 0 {
				var arr []string
				if q.sep == " " {
					arr = re.Split(lineStr, -1)
				} else {
					arr = strings.Split(lineStr, q.sep)
				}

				if len(arr) < q.k {
					q.sp.Success("MISS", 1)
				} else {
					k := strings.TrimSpace(arr[q.k-1])

					if q.v > 0 {
						if len(arr) < q.v {
							q.sp.Success("ERROR", 1)
						} else {
							if val, err := strconv.ParseInt(arr[q.v-1], 10, 32); err != nil {
								q.sp.Success("ERROR", 1)
							} else {
								q.sp.Success(k, int(val))
							}
						}
					} else {
						q.sp.Success(k, 1)
					}
				}
			}
		}
		if err == io.EOF {
			break
		}
	}
}

func (q *QPSCount) Print(str string) {
	r := regexp.MustCompile(`,\(suc:[^\)]+\),\(fail:[^\)]+\)`)
	str = r.ReplaceAllString(str, "")
	p := strings.Index(str, "LINE_all")
	if p > -1 {
		str = str[p:]
	}
	arr := strings.Split(str, ";")
	h_m_l := 0
	var infos [][]string
	for _, l := range arr {
		l = strings.Replace(l, "]", "", -1)
		l = strings.Replace(l, "[", "", -1)
		l = strings.Replace(l, "(", "\t", -1)
		l = strings.Replace(l, ")", "", -1)
		l = strings.Replace(l, ",", "\t", -1)
		l = strings.Replace(l, "/s", "", -1)
		l = strings.Replace(l, ":", "", -1)

		tmp := strings.Split(l, "\t")
		h_l := len(tmp[0])
		if h_l > h_m_l {
			h_m_l = h_l
		}
		if len(tmp) > 1 {
			tmp[1] = strings.Replace(tmp[1], "total", "", -1)
		}

		infos = append(infos, tmp)
	}

	if q.n > 0 {
		fmt.Fprint(os.Stdout, fmt.Sprintf("\033[%dA", q.n))
		fmt.Fprint(os.Stdout, "\033[K\033[D")
	}

	ft := fmt.Sprintf("%%%ds\t%%10s\t%%10s\n", h_m_l+1)
	fmt.Fprintf(os.Stdout, ft, "KEY", "TOTAL", "QPS")
	fmt.Fprintln(os.Stdout, strings.Repeat("-", h_m_l+35))

	for _, a := range infos {
		fmt.Fprintf(os.Stdout, ft, a[0], a[1], a[2])
	}
	q.n = len(infos) + 2
}

func (q *QPSCount) Close() {
	q.sp.Stop()
}
