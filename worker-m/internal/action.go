// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"log"
)

type actions []action

func (as actions) Run(ctx context.Context, when string, lg *log.Logger) {
	for _, a := range as {
		if a.When != when {
			continue
		}
		a.run(ctx, lg)
	}
}

type action struct {
	When   string
	Do     string
	Params map[string]any
}

func (a action) run(ctx context.Context, lg *log.Logger) {
	if a.Do == "" {
		return
	}
	if ctx.Err() != nil {
		return
	}
	switch a.Do {
	case "HTTPCall":
		httpCallParam(a.Params).run(ctx, lg)
	case "Dial":
		dialParam(a.Params).run(ctx, lg)
	default:
		lg.Printf("not supported action: %q", a.Do)
	}
}
