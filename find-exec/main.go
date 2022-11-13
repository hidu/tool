// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/11/13

package main

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
)

var name=flag.String("name","","find file name")


func main(){
	flag.Parse()
	if *name==""{
		log.Fatalln(color.RedString("-name is required"))
	}
	cmdName:=flag.Arg(0)
	if cmdName==""{
		log.Fatalln(color.RedString("cmd is empty"))
	}
	var fail int
	err:=filepath.Walk("./", func(path string, info fs.FileInfo, err error) error {
		if err!=nil{
			return err
		}
		n:=filepath.Base(path)
		if n!=*name{
			return nil
		}
		dir:=filepath.Dir(path)
		cmd:=exec.Command(cmdName,flag.Args()[1:]...)
		color.Cyan("Dir: %s Exec: %s",dir,cmd.String())
		cmd.Dir=dir
		cmd.Stdin=os.Stdin
		cmd.Stdout=os.Stdout
		cmd.Stderr=os.Stderr
		if e1:=cmd.Run();e1!=nil{
			fail++
			color.Red(e1.Error())
		}
		return nil
	})
	if err!=nil{
		log.Fatalln(color.RedString(err.Error()))
	}
	if fail>0{
		log.Fatalln(color.RedString("total %d tasks failed",fail))
	}
}


func init(){
	color.Output=os.Stderr
}
