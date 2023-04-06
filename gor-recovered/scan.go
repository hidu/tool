package main

// thanks https://github.com/John520/goroutine-with-recover

import (
	"go/ast"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `find goroutines not recovered`

var Analyzer = &analysis.Analyzer{
	Name:     "gor-recovered",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var wd string

func init() {
	c, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd = c
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.GoStmt)(nil),
	}

	var ignore bool
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		if ignore {
			return
		}

		if nf, ok := node.(*ast.File); ok {
			f := pass.Fset.File(nf.Pos())
			// protobuf 自动生成的 go 文，其编解码器直接依赖其生成的字段顺序，不能优化
			if strings.HasSuffix(f.Name(), "_test.go") {
				ignore = true
				rn, _ := filepath.Rel(wd, f.Name())
				log.Println("ignored:", rn)
				return
			}
		}
		gs, ok := node.(*ast.GoStmt)
		if !ok {
			return
		}
		check(pass, gs)
	})
	return nil, nil
}

func check(pass *analysis.Pass, gs *ast.GoStmt) {
	var r bool
	switch gs.Call.Fun.(type) {
	case *ast.FuncLit: // go func(){}
		funcLit := gs.Call.Fun.(*ast.FuncLit)
		r = hasRecover(funcLit.Body)
	case *ast.Ident: // go goFuncWithoutRecover()
		id := gs.Call.Fun.(*ast.Ident)
		fd, ok := id.Obj.Decl.(*ast.FuncDecl) // fd 是 goFuncWithoutRecover 定义
		if !ok {
			return
		}
		r = hasRecover(fd.Body)
	default:
	}
	if !r {
		pass.Reportf(gs.Pos(), "goroutine not recovered")
	}
}

func hasRecover(bs *ast.BlockStmt) bool {
	for _, blockStmt := range bs.List {
		deferStmt, ok := blockStmt.(*ast.DeferStmt) // 是否包含defer 语句
		if !ok {
			return false
		}
		switch vt0 := deferStmt.Call.Fun.(type) {
		case *ast.SelectorExpr:
			// 判断是否defer中包含  helper.Recover()
			if vt0.Sel.Name == "Recover" {
				return true
			}
		case *ast.FuncLit:
			// 判断是否有 defer func(){ }()
			for i := range vt0.Body.List {
				stmt := vt0.Body.List[i]
				switch vt1 := stmt.(type) {
				case *ast.ExprStmt:
					// recover()
					if isRecoverExpr(vt1.X) {
						return true
					}
				case *ast.IfStmt:
					// if r:=recover();r!=nil{}
					as, ok := vt1.Init.(*ast.AssignStmt)
					if !ok {
						continue
					}
					if isRecoverExpr(as.Rhs[0]) {
						return true
					}
				case *ast.AssignStmt:
					// r=:recover
					if isRecoverExpr(vt1.Rhs[0]) {
						return true
					}
				}
			}
		}
	}
	return false
}

func isRecoverExpr(expr ast.Expr) bool {
	ac, ok := expr.(*ast.CallExpr) // r:=recover()
	if !ok {
		return false
	}
	id, ok := ac.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	if id.Name == "recover" {
		return true
	}
	return false
}
