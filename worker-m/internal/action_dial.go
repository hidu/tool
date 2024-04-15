// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"log"
	"net"
	"time"
)

type dialParam map[string]any

func (hc dialParam) run(ctx context.Context, lg *log.Logger) {
	type params struct {
		Addr     string
		Times    int    // 访问次数
		Interval string // 间隔时间，如 1s
		Timeout  string // 超时时间，如 1s
	}
	p := &params{}
	if err := transform(p, hc); err != nil {
		lg.Printf("transform %#v failed: %v\n", hc, err)
		return
	}
	if p.Addr == "" {
		return
	}

	interval := parseDurationDef(p.Interval, time.Second)
	timeout := parseDurationDef(p.Timeout, time.Second)

	for i := 0; checkTimes(i, p.Times); i++ {
		if ctx.Err() != nil {
			return
		}
		conn, err := hc.dial(ctx, p.Addr, timeout)
		if err != nil {
			lg.Printf("Dial %d/%d %s, err: %v\n", i+1, p.Times, p.Addr, err)
		} else {
			_ = conn.Close()
			lg.Printf("Dial %d/%d %s, remote %s\n", i+1, p.Times, p.Addr, conn.RemoteAddr().String())
		}
		sleep(ctx, interval)
	}
}

func (hc dialParam) dial(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
}
