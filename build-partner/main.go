// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/12/21

package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		return
	}
	cfg.Exec()
}

const cfgName = ".build-partner.toml"

func loadConfig() (*config, error) {
	var cfg *config
	_, err := toml.DecodeFile(cfgName, &cfg)
	return cfg, err
}

type config struct {
	GitFetch []string
}

func (c *config) Exec() {
	if len(c.GitFetch) == 0 {
		return
	}
	for _, gf := range c.GitFetch {
		c.execGitFetch(gf)
	}
}

func (c *config) execGitFetch(gf string) {
	info := strings.Fields(gf)
	if len(info) != 8 {
		log.Fatalln("invalid GitFetch", gf, ", expect has 8 fields")
	}

	repo := info[2]
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", repo)
	cmd.Dir = "../"
	runCmd(cmd)

	pos := strings.LastIndex(repo, "/")
	name := repo[pos+1:]
	cmd2 := exec.CommandContext(ctx, "git", info[1:4]...)
	cmd2.Dir = "../" + name
	runCmd(cmd2)

	cmd3 := exec.CommandContext(ctx, "git", info[6:8]...)
	cmd3.Dir = "../" + name
	runCmd(cmd3)
}

func runCmd(cmd *exec.Cmd) {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Println("exec:", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Fatalln(err.Error())
	}
}
