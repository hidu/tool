// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/15

package internal

import (
	"context"
	"log"

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
