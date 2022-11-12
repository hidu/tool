// from https://github.com/kayon/qqwry

package qqwry

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"

	zhcn "golang.org/x/text/encoding/simplifiedchinese"
)

const (
	INDEX_LENGTH    = 7
	REDIRECT_MODE_1 = 0x01
	REDIRECT_MODE_2 = 0x02
)

type QQwry struct {
	file    *os.File
	datfile string
}

type QQwrySearch struct {
	file *os.File
}

type Result struct {
	IP      string `json:"ip"`
	Begin   string `json:"begin"`
	End     string `json:"end"`
	Country string `json:"country"`
	Area    string `json:"area"`
}

func (res *Result) String() string {
	b, _ := json.Marshal(res)
	return string(b)
}

func NewQQwry(datfile string) *QQwry {
	wry := &QQwry{datfile: datfile}
	var err error
	wry.file, err = os.OpenFile(datfile, os.O_RDONLY, 0400)
	if err != nil {
		log.Println("init qqwry failed", err)
		return nil
	}
	return wry
}

func (wry *QQwry) Close() {
	if wry.file != nil {
		wry.file.Close()
	}
}

func (wry *QQwry) Search(ipstr string) (res Result) {
	search := &QQwrySearch{file: wry.file}
	res.IP = ipstr
	offset := search.indexOf(search.packing(ipstr))
	if offset == 0 {
		return
	}
	search.file.Seek(offset, os.SEEK_SET)
	res.Begin = search.long2ip(search.readLong())
	startRedirect := search.redirectOffset()

	search.file.Seek(startRedirect, os.SEEK_SET)
	res.End = search.long2ip(search.readLong())
	switch search.mode() {
	// 1、2都重定向
	case REDIRECT_MODE_1:
		startRedirect = search.redirectOffset()
		search.file.Seek(startRedirect, os.SEEK_SET)
		switch search.mode() {
		case REDIRECT_MODE_2:
			search.file.Seek(search.redirectOffset(), os.SEEK_SET)
			res.Country = search.readString()
			search.file.Seek(startRedirect+4, os.SEEK_SET)
			res.Area = search.readArea()
		default:
			search.file.Seek(startRedirect, os.SEEK_SET)
			res.Country = search.readString()
			res.Area = search.readArea()
		} // End switch
	// 1重定向, 2没有重定向
	case REDIRECT_MODE_2:
		search.file.Seek(search.redirectOffset(), os.SEEK_SET)
		res.Country = search.readString()
		search.file.Seek(startRedirect+8, os.SEEK_SET)
		res.Area = search.readArea()
	default:
		search.file.Seek(startRedirect+4, os.SEEK_SET)
		res.Country = search.readString()
		res.Area = search.readArea()
	} // End switch
	res.Country = strings.Trim(gbk2utf8(res.Country), "\u0000")
	res.Area = strings.Trim(gbk2utf8(res.Area), "\u0000")
	return
}

func (wry *QQwrySearch) indexOf(ip []byte) (index int64) {
	wry.file.Seek(0, os.SEEK_SET)
	first := wry.readLong()
	last := wry.readLong()
	var low, mid, high uint32
	var begin, end []byte
	high = (last - first) / INDEX_LENGTH
	for low <= high {
		mid = (low + high) >> 1
		wry.file.Seek(int64(first+mid*INDEX_LENGTH), os.SEEK_SET)
		begin = wry.readBytes()
		if bytes.Compare(ip, begin) < 0 {
			high = mid - 1
		} else {
			wry.file.Seek(wry.redirectOffset(), os.SEEK_SET)
			end = wry.readBytes()
			if bytes.Compare(ip, end) > 0 {
				low = mid + 1
			} else {
				index = int64(first + mid*INDEX_LENGTH)
				break
			}
		}
	} // End for
	return
}

func (wry *QQwrySearch) readString() string {
	bs := make([]byte, 1)
	res := []byte{}
	wry.file.Read(bs)
	for bs[0] != 0 {
		res = append(res, bs[0])
		wry.file.Read(bs)
	}
	return string(res)
}

func (wry *QQwrySearch) readArea() (area string) {
	origin, _ := wry.file.Seek(0, os.SEEK_CUR)
	switch wry.mode() {
	case 0:
	case REDIRECT_MODE_1:
		fallthrough
	case REDIRECT_MODE_2:
		wry.file.Seek(wry.redirectOffset(), os.SEEK_SET)
		area = wry.readString()
	default:
		wry.file.Seek(origin, os.SEEK_SET)
		area = wry.readString()
	} // End switch
	return
}

func (wry *QQwrySearch) redirectOffset() int64 {
	bs := make([]byte, 3)
	wry.file.Read(bs)
	bs = append(bs, 0)
	return int64(binary.LittleEndian.Uint32(bs))
}

func (wry *QQwrySearch) readLong() uint32 {
	bs := make([]byte, 4)
	wry.file.Read(bs)
	return binary.LittleEndian.Uint32(bs)
}

func (wry *QQwrySearch) readBytes() (res []byte) {
	bs := make([]byte, 4)
	wry.file.Read(bs)
	for i := len(bs) - 1; i > -1; i-- {
		res = append(res, bs[i])
	}
	return
}

func (wry *QQwrySearch) mode() byte {
	bs := make([]byte, 1)
	wry.file.Read(bs)
	return bs[0]
} // End Func:flag

func (wry QQwrySearch) packing(ipstr string) (ip []byte) {
	ip = make([]byte, 4)
	binary.BigEndian.PutUint32(ip, wry.ip2long(ipstr))
	return
}

func (wry QQwrySearch) ip2long(ipstr string) uint32 {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip.To4())
}

func (wry QQwrySearch) long2ip(iplong uint32) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, iplong)
	ip := net.IP(ipByte)
	return ip.String()
}

func gbk2utf8(gbk string) string {
	gbkByte := []byte(gbk)
	dst := make([]byte, len(gbkByte)*2)
	_, _, err := zhcn.GBK.NewDecoder().Transform(dst, gbkByte, true)
	if err != nil {
		log.Println("gbk2utf8 failed", err)
		return gbk
	}
	return string(dst)
}
