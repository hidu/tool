// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/11/13

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/fsgo/cmdutil"
)

var name = flag.String("name", "go.mod", "find file name")
var useReg = flag.Bool("e", false, "name as regular expression")
var fcc = flag.String("fcc", "", "file content Contains")
var conc = flag.Int("c", 1, "Number of multiple task to make at a time")

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

	nameMatch := func(filePath string) bool {
		fileName := filepath.Base(filePath)
		if *useReg {
			return reg.MatchString(fileName)
		}
		return fileName == *name
	}

	match := func(filePath string) bool {
		if !nameMatch(filePath) {
			return false
		}
		if *fcc == "" {
			return true
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("os.ReadFile(%q) failed: %v\n", filePath, err)
			return false
		}
		return bytes.Contains(content, []byte(*fcc))
	}

	wg := &cmdutil.WorkerGroup{
		Max: *conc,
	}

	var index int
	var fail atomic.Int64
	err := filepath.Walk("./", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileName := filepath.Base(path)

		if !match(path) {
			return nil
		}
		index++

		dir := filepath.Dir(path)
		cmd := exec.Command(cmdName, flag.Args()[1:]...)

		s0 := color.GreenString("%3d.", index)
		s1 := fmt.Sprintf("Dir: %s, MatchFile: %s", color.CyanString(dir), color.CyanString(fileName))
		s2 := "Exec: " + color.YellowString("%s", cmd.String())

		s4 := s0 + " " + s1 + " " + s2
		os.Stderr.WriteString(s4 + "\n")

		cmd.Dir = dir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		wg.Run(func() {
			if e1 := cmd.Run(); e1 != nil {
				fail.Add(1)
				color.Red("[Fail] "+cmdutil.CleanColor(s4)+" Err: %s\n", e1.Error())
			}
		})

		return fs.SkipDir
	})
	wg.Wait()
	if err != nil {
		log.Fatalln(color.RedString(err.Error()))
	}
	if failNum := fail.Load(); failNum > 0 {
		log.Fatalln(color.RedString("total %d tasks failed", failNum))
	}
}

func init() {
	color.Output = os.Stderr
}
