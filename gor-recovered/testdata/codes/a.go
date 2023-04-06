// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/6

package codes

import "fmt"

func a0(){
	go func() {
		fmt.Println("not recover")
	}()
}

func a1(){
	go func() {
		defer func() {
			recover()
		}()
	}()
}


func a2(){
	go func() {
		defer func() {
			_=recover()
		}()
	}()
}

func a3(){
	go func() {
		defer func() {
			if r := recover(); r != nil {
			}
		}()
	}()
}