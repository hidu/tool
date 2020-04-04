package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var dsn_conf_path = flag.String("dsn_conf", "./dsn.conf", "dsn conf path")
var dsn_index = flag.Int("dsn_index", 0, "use which dsn in dsn_conf")

var sql_file_path = flag.String("sql_file", "./sql.txt", "sql file path")

func readFile(file_path string) (lines []string) {
	data, err := ioutil.ReadFile(file_path)
	if err != nil {
		log.Fatalln(err)
	}
	tmp := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range tmp {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

type Timer struct {
	name  string
	start time.Time
	end   time.Time
	msg   string
}

var timers []*Timer

func NewTimer(name string, msg string) *Timer {
	timer := &Timer{name: name, start: time.Now(), msg: msg}
	timers = append(timers, timer)
	return timer
}

func (t *Timer) UsedStr() string {
	used := t.end.Sub(t.start)
	return fmt.Sprintf("%.4f", float64(used.Nanoseconds())/1000000)
}

func (t *Timer) Stop() {
	t.end = time.Now()
}

func (t *Timer) String() string {
	used := t.end.Sub(t.start)
	return fmt.Sprintf("%10s used: %10s ms %s", t.name, fmt.Sprintf("%.4f", float64(used.Nanoseconds())/1000000), t.msg)
}

func main() {
	flag.Usage = func() {
		fmt.Println("mysql db test toolkit\nUsage")
		flag.PrintDefaults()
		fmt.Println("\ndsn format -> username:password@protocol(address)/dbname?param=value")
		fmt.Println("dsn eg: test:test123@tcp(127.0.0.1:3306)/shop?charset=utf8")
	}
	flag.Parse()
	dsns := readFile(*dsn_conf_path)
	if len(dsns) == 0 {
		log.Fatalln("dsn conf is empty")
	}
	if *dsn_index < 0 || *dsn_index >= len(dsns) {
		log.Fatalln("dsn should range (0,", len(dsns)-1, ")")
	}

	sqls := readFile(*sql_file_path)
	if len(sqls) == 0 {
		log.Fatalln("sql file is empty")
	}
	dsn := dsns[*dsn_index]

	strLine := strings.Repeat("-", 80)
	fmt.Println(strLine)
	fmt.Println("\tdsn:", dsn)
	fmt.Println(strLine)
	timer := NewTimer("db_conn", "")
	time.Sleep(1 * time.Second)
	db, err := sql.Open("mysql", dsn)
	timer.Stop()
	if err != nil {
		log.Fatalln("connect db failed", err)
	}
	defer db.Close()

	for i, sql := range sqls {
		log.Println("run sql:", sql)
		timer := NewTimer(fmt.Sprintf("sql_%d", i), sql)
		_, err := db.Query(sql)
		timer.Stop()
		log.Println("last used:", timer.UsedStr(), "ms")
		if err != nil {
			log.Fatalln("dbError:", err)
		}
	}
	log.Println("run all sql done\n")
	fmt.Println(strLine)
	for _, t := range timers {
		fmt.Println(t.String())
	}
	fmt.Println(strLine)
	fmt.Println("\n")
}
