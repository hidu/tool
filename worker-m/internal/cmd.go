// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Command struct {
	Cmd     string
	Home    string
	Args    []string
	Expire  string // 进程最大运行时长，如 1h
	Next    string // 下次运行的时间间隔,如 1h
	Actions actions
}

func (c Command) Run(ctxRoot context.Context, name string) error {
	next := parseDuration(c.Next)
	expire := parseDuration(c.Expire)

	ctxWithExpire := func() (context.Context, context.CancelFunc) {
		if expire > time.Second {
			return context.WithTimeout(ctxRoot, expire)
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

		err := cmd.Start()
		if err != nil {
			lg.Println(cmdName, "Start failed, err=", err)
			return
		}
		time.AfterFunc(100*time.Millisecond, func() {
			c.Actions.Run(ctx, "after_start", lg)
		})
		err = cmd.Wait()
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
