// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/7/31

package main

import (
	"time"

	"github.com/go-vgo/robotgo"
)

func main() {
	xSize, ySize := robotgo.GetScreenSize()
	for i := 0; ; i++ {
		// fmt.Println("i: ", i)
		// MoveMouse(800, i)
		robotgo.Move((i*30)%xSize, (i*20)%ySize)
		time.Sleep(5*time.Second)
	}
}
