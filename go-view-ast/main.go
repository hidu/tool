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

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var fn = flag.String("fn", "", "filter with func name")
var ty = flag.String("type", "", "filter with type name")
var useDst = flag.Bool("dst", false, "use dst")

func main() {
	flag.Parse()
	fileName := flag.Arg(0)
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil {
		log.Fatalln(err)
	}

	var node ast.Node
	if len(*fn) > 0 {
		ast.Inspect(file, func(n ast.Node) bool {
			if n1, ok := n.(*ast.FuncDecl); ok && n1.Name.Name == *fn {
				node = n1
				return false
			}
			return true
		})
	} else if len(*ty) > 0 {
		ast.Inspect(file, func(n ast.Node) bool {
			if n1, ok := n.(*ast.TypeSpec); ok && n1.Name.Name == *ty {
				node = n1
				return false
			}
			return true
		})
	} else {
		node = file
	}

	if *useDst {
		dn, err := decorator.Decorate(fset, node)
		if err != nil {
			log.Fatalln(err)
		}
		dst.Print(dn)
	} else {
		ast.Print(fset, node)
	}
}
