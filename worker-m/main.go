// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/10/21

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
)

var config = flag.String("c", "./task.toml", "toml config path")

func main() {
	flag.Parse()
	cf := &ConfigFile{}
	_, err := toml.DecodeFile(*config, &cf)
	if err != nil {
		log.Fatalln(err)
	}
	if len(cf.Workers) == 0 {
		log.Fatalln("no workers")
	}
	log.Fatalln("exit:", cf.Run())
}

type ConfigFile struct {
	Workers map[string]Command
}

func (cf *ConfigFile) Run() error {
	ec := make(chan error, len(cf.Workers))
	for name, cmd := range cf.Workers {
		go func(name string, cmd Command) {
			ec <- cmd.Run(name)
		}(name, cmd)
	}
	return <-ec
}

type Command struct {
	Cmd  string
	Home string
	Args []string

	Next string // 下次运行的时间间隔
}

func (c Command) Run(name string) error {
	var next time.Duration
	if c.Next != "" {
		n, err := time.ParseDuration(c.Next)
		if err != nil {
			return err
		}
		next = n
	}

	ctxRoot, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	lg := log.New(os.Stderr, name, log.LstdFlags)

	exec := func() {
		ctx, cancel := context.WithCancel(ctxRoot)
		defer cancel()

		start := time.Now()

		cmd := exec.CommandContext(ctx, c.Cmd, c.Args...)
		lg.Println(cmd.String(), "Start..., Dir=", c.Home)
		cmd.Dir = c.Home
		cmd.Stdout = os.Stdout
		cmd.Stderr = writeFn(func(b []byte) (int, error) {
			lg.Print(string(b))
			return len(b), nil
		})
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		err := cmd.Run()
		cost := time.Since(start)
		lg.Println(cmd.String(), "Exit, err=", err, ", Duration=", cost.String())
		if cost < time.Minute {
			time.Sleep(time.Second)
		}
		if next > 0 && next > cost {
			dur := next - cost
			log.Println("Will sleep", dur.String(), "for next time")
			time.Sleep(dur)
		}
	}

	for idx := 0; ; idx++ {
		lg.SetPrefix(fmt.Sprintf("%s [%d]", name, idx))
		exec()
	}
}

type writeFn func(b []byte) (int, error)

func (w writeFn) Write(p []byte) (n int, err error) {
	return w(p)
}
