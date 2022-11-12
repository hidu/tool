// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/10/13

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

var ts = token.NewFileSet()

var needCall = flag.Bool("call", false, "func call")
var needDefine = flag.Bool("d", true, "func define")
var needResult = flag.Bool("r", true, "need result")

func main() {
	flag.Parse()
	filepath.Walk("./", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		do(path)
		return nil
	})

	printResult()
}

func do(f string) {
	af, err := parser.ParseFile(ts, f, nil, parser.ParseComments)
	if err != nil {
		log.Printf("parser %s failed", f)
		return
	}
	fname := filepath.Base(f)
	ast.Inspect(af, func(node ast.Node) bool {
		switch vt := node.(type) {
		case *ast.FuncDecl:
			if *needDefine {
				whenFuncDecl(fname, vt)
			}
		case *ast.CallExpr:
			if *needCall {
				whenCallExpr(fname, vt)
			}
		}
		return true
	})
}

var (
	fnTotal   int
	argsTotal int
	argsMax   int
	fnDetail  = map[int]int{}
)

func whenFuncDecl(fname string, node *ast.FuncDecl) {
	num := len(node.Type.Params.List)

	fnTotal++
	argsTotal += num
	if num > argsMax {
		argsMax = num
	}
	fnDetail[num]++

	fmt.Printf(" def_fn %20s %20s %d\n", fname, node.Name, num)
}

func printResult() {
	var avg float64
	if fnTotal > 0 {
		avg = float64(argsTotal) / float64(fnTotal)
	}
	fmt.Printf("\ndefine_fnc_total: %d, args_num_argv: %.1f, args_num_argv: %d\n", fnTotal, avg, argsMax)
	fmt.Printf("detail: %v\n", fnDetail)
	keys := make([]int, 0, len(fnDetail))
	for k := range fnDetail {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for k := range keys {
		v := fnDetail[k]
		fmt.Printf("args_num: %-6d fn_total: %-7d rate: %.2f%%\n", k, v, float64(v)/float64(fnTotal)*100)
	}
}

func whenCallExpr(fname string, node *ast.CallExpr) {
	fn, ok := node.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	fnX, ok2 := fn.X.(*ast.Ident)
	if !ok2 {
		return
	}
	fmt.Printf("call_fn %20s %20s %d\n", fname, fnX.Name+"."+fn.Sel.Name, len(node.Args))
}
