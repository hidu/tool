// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/9/10

package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
)

var fn=flag.String("fn","","filter with func name")
func main(){
	flag.Parse()
	
	fileName:=flag.Arg(0)
	fset:=token.NewFileSet()
	file,err:=parser.ParseFile(fset,fileName,nil,parser.ParseComments)
	if err!=nil{
		log.Fatalln(err)
	}
	var node ast.Node
	if len(*fn)>0 {
		ast.Inspect(file, func(n ast.Node) bool {
			if n1, ok := n.(*ast.FuncDecl); ok && n1.Name.Name==*fn {
				node=n1
				return false
			}
			return true
		})
	}else{
		node=file
	}
	ast.Print(fset,node)
}
