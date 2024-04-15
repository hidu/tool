// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"log"
	"net/http"
	"time"
)

type httpCallParam map[string]any

func (hc httpCallParam) run(ctx context.Context, lg *log.Logger) {
	type params struct {
		URL      string
		Times    int    // 访问次数
		Interval string // 间隔时间，如 1s
		Timeout  string // 超时时间，如 1s
	}
	p := &params{}
	if err := transform(p, hc); err != nil {
		lg.Printf("transform %#v failed: %v\n", hc, err)
		return
	}
	if p.URL == "" {
		return
	}
	interval := parseDurationDef(p.Interval, time.Second)
	timeout := parseDurationDef(p.Timeout, time.Second)

	httpClient := &http.Client{
		Timeout: timeout,
	}
	defer httpClient.CloseIdleConnections()

	for i := 0; checkTimes(i, p.Times); i++ {
		if ctx.Err() != nil {
			return
		}
		resp, err := httpClient.Get(p.URL)
		if err != nil {
			lg.Printf("HTTPCall %d/%d %s, err: %v\n", i+1, p.Times, p.URL, err)
		} else {
			_ = resp.Body.Close()
			lg.Printf("HTTPCall %d/%d %s, status %d\n", i+1, p.Times, p.URL, resp.StatusCode)
		}
		sleep(ctx, interval)
	}
}
