// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/6

package codes

// RunVoidNoPanic 执行函数，并捕捉 panic
func RunVoidNoPanic(fn func()) {
	defer func() {
		_ = recover()
	}()
	fn()
}


func c0(){
	go RunVoidNoPanic(func() {

	})
}

func c1(){
	go RunVoidNoPanic(hello)
}

func call(fn func()){
}

func c2(){
	call(r1)
}

func r1(){
	_=recover()
}