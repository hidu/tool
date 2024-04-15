// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func parseDuration(str string) time.Duration {
	if str == "" {
		return 0
	}
	n, err := time.ParseDuration(str)
	Assert(err, "parser %q failed", str)
	return n
}

func parseDurationDef(str string, def time.Duration) time.Duration {
	dur, _ := time.ParseDuration(str)
	if dur > 0 {
		return dur
	}
	return def
}

func transform(dest, source any) error {
	bf, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("marshal source failed: %w", err)
	}

	return json.Unmarshal(bf, dest)
}

func sleep(ctx context.Context, sleep time.Duration) {
	tm := time.NewTimer(sleep)
	defer tm.Stop()
	select {
	case <-tm.C:
	case <-ctx.Done():
	}
}

func Assert(err error, format string, args ...any) {
	if err != nil {
		msg := fmt.Sprintf(format, args...)
		log.Fatalln(msg, err.Error())
	}
}

func checkTimes(index int, times int) bool {
	switch times {
	case -1:
		return true
	case 0:
		return index == 0
	default:
		return index < times
	}
}
