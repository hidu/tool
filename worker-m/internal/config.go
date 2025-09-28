// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"log"
	"slices"

	"github.com/BurntSushi/toml"
)

func LoadConfig(fp string) *ConfigFile {
	cf := &ConfigFile{}
	_, err := toml.DecodeFile(fp, &cf)
	if err != nil {
		log.Fatalln(err)
	}
	if len(cf.Workers) == 0 {
		log.Fatalln("no workers")
	}
	if len(cf.Env) > 0 {
		for name, cmd := range cf.Workers {
			cmd.Env = slices.Insert(cmd.Env, 0, cf.Env...)
			cf.Workers[name] = cmd
		}
	}
	return cf
}

type ConfigFile struct {
	Workers map[string]Command
	Env     []string // 环境变量，会设置给所有的 Workers
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
