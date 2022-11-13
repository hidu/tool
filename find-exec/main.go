// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/11/13

package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/fatih/color"
)

var name = flag.String("name", "go.mod", "find file name")
var useReg = flag.Bool("e", false, "name as regular expression")

func main() {
	flag.Parse()
	if len(*name) == 0 {
		log.Fatalln(color.RedString("-name is required"))
	}
	cmdName := flag.Arg(0)
	if len(cmdName) == 0 {
		log.Fatalln(color.RedString("cmd is empty"))
	}

	var reg *regexp.Regexp
	if *useReg {
		r, err := regexp.Compile(*name)
		if err != nil {
			log.Fatalln(color.RedString("regexp.Compile(%q): %v", *name, err))
		}
		reg = r
	}

	match := func(fileName string) bool {
		if *useReg {
			return reg.MatchString(fileName)
		}
		return fileName == *name
	}
	var index int
	var fail int
	err := filepath.Walk("./", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileName := filepath.Base(path)

		if !match(fileName) {
			return nil
		}
		index++

		dir := filepath.Dir(path)
		cmd := exec.Command(cmdName, flag.Args()[1:]...)

		s0 := color.GreenString("%3d.", index)
		s1 := color.CyanString("Dir: %s, MatchFile: %s", dir, fileName)
		s2 := color.YellowString("Exec: %s", cmd.String())
		fmt.Println(s0, s1, s2)

		cmd.Dir = dir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e1 := cmd.Run(); e1 != nil {
			fail++
			color.Red(e1.Error())
		}
		return fs.SkipDir
	})
	if err != nil {
		log.Fatalln(color.RedString(err.Error()))
	}
	if fail > 0 {
		log.Fatalln(color.RedString("total %d tasks failed", fail))
	}
}

func init() {
	color.Output = os.Stderr
}
