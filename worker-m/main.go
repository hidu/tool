// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/10/21

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
)

var config = flag.String("c", "./task.toml", "toml config path")
var listen = flag.String("l", ":9205", "listen addr")

func main() {
	flag.Parse()
	cf := loadConfig()

	runServer()

	ctx, cancel := getContext()
	defer cancel()

	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		log.Println("call /exit", r.RemoteAddr)
		cancel()
		_, _ = w.Write([]byte("exiting ..."))
	})

	log.Fatalln("exit:", cf.Run(ctx))
}

func getContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-ch
		cancel()
		time.Sleep(time.Second)
	}()

	return ctx, cancel
}

func assert(err error, format string, args ...any) {
	if err != nil {
		msg := fmt.Sprintf(format, args...)
		log.Fatalln(msg, err.Error())
	}
}

func runServer() {
	if *listen == "" || *listen == "no" {
		return
	}
	l, err := net.Listen("tcp", *listen)
	assert(err, "listen %s failed", *listen)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	go func() {
		err = http.Serve(l, nil)
		assert(err, "http.Serve failed")
	}()
}

func loadConfig() *ConfigFile {
	cf := &ConfigFile{}
	_, err := toml.DecodeFile(*config, &cf)
	if err != nil {
		log.Fatalln(err)
	}
	if len(cf.Workers) == 0 {
		log.Fatalln("no workers")
	}
	return cf
}

type ConfigFile struct {
	Workers map[string]Command
}

func (cf *ConfigFile) Run(ctx context.Context) error {
	ec := make(chan error, len(cf.Workers))
	for name, cmd := range cf.Workers {
		go func(name string, cmd Command) {
			ec <- cmd.Run(ctx, name)
		}(name, cmd)
	}
	return <-ec
}

type Command struct {
	Cmd    string
	Home   string
	Args   []string
	Expire string // 进程最大运行时长，如 1h
	Next   string // 下次运行的时间间隔,如 1h
}

func parseDuration(str string) time.Duration {
	if str == "" {
		return 0
	}
	n, err := time.ParseDuration(str)
	assert(err, "parser %q failed", str)
	return n
}

func (c Command) Run(ctxRoot context.Context, name string) error {
	next := parseDuration(c.Next)
	expire := parseDuration(c.Expire)

	ctxWithExpire := func() (context.Context, context.CancelFunc) {
		if expire > time.Second {
			return context.WithCancel(ctxRoot)
		}
		return context.WithCancel(ctxRoot)
	}

	lg := log.New(os.Stderr, "["+name+"] ", log.LstdFlags)

	exec := func() {
		ctx, cancel := ctxWithExpire()
		defer cancel()

		start := time.Now()

		cmd := exec.CommandContext(ctx, c.Cmd, c.Args...)
		cmdName := fmt.Sprintf("[%s]", cmd.String())
		lg.Println(cmdName, "Start..., Dir=", c.Home)
		cmd.Dir = c.Home
		cmd.Stdout = os.Stdout
		cmd.Stderr = writeFn(func(b []byte) (int, error) {
			lg.Print(string(b))
			return len(b), nil
		})
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		err := cmd.Run()
		cost := time.Since(start)
		lg.Println(cmdName, "Exit, err=", err, ", Duration=", cost.String())
		if cost < time.Minute {
			time.Sleep(time.Second)
		}
		if next > 0 && next > cost {
			dur := next - cost
			log.Println("will sleep", dur.String(), "for next time")
			time.Sleep(dur)
		}
	}

	for idx := 0; ; idx++ {
		if err := ctxRoot.Err(); err != nil {
			return err
		}
		lg.SetPrefix(fmt.Sprintf("[%s %d] ", name, idx))
		exec()
	}
}

type writeFn func(b []byte) (int, error)

func (w writeFn) Write(p []byte) (n int, err error) {
	return w(p)
}
