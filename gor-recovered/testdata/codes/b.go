// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/6

package codes

func b0(){
	go hello()
}

func hello(){
}

func b1(){
	go helloRecover()
}

func helloRecover(){
	defer func() {
		recover()
	}()
}
