package main

// thanks https://github.com/John520/goroutine-with-recover

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `find goroutines not recovered`

var Analyzer = &analysis.Analyzer{
	Name: "recovered",
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
		findcall.Analyzer,
	},
	Run: run,
}

var wd string

func init() {
	c, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd = c
}

var debug bool

func run(pass *analysis.Pass) (any, error) {
	log.SetPrefix("")
	if ft := flag.Lookup("debug"); ft != nil {
		debug = ft.Value.String() != ""
	}

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
			ignore = checkIgnore(pass, nf)
			if ignore {
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

func countGoStmt(f *ast.File) int {
	var num int
	ast.Inspect(f, func(node ast.Node) bool {
		if _, ok := node.(*ast.GoStmt); ok {
			num++
		}
		return true
	})
	return num
}

func checkIgnore(pass *analysis.Pass, nf *ast.File) bool {
	tokenFile := pass.Fset.File(nf.Pos())

	if !strings.HasSuffix(tokenFile.Name(), ".go") {
		return true
	}

	// protobuf 自动生成的 go 文，其编解码器直接依赖其生成的字段顺序，不能优化
	rn, _ := filepath.Rel(wd, tokenFile.Name())
	if strings.HasSuffix(tokenFile.Name(), "_test.go") {
		// log.Println("ignored by *_test.go:", rn)
		return true
	}

	for i := 0; i < len(nf.Imports); i++ {
		ni := nf.Imports[i]
		p1, _ := strconv.Unquote(ni.Path.Value)
		if p1 == "testing" {
			if debug {
				log.Println(`ignored: has import "testing":`, rn, ", has GoStmt:", countGoStmt(nf))
			}
			return true
		}
	}

	return false
}

var successID int
var failID int

func check(pass *analysis.Pass, gs *ast.GoStmt) (ok bool) {
	code1 := nodeCode(pass, gs, 10)
	defer func() {
		if !ok {
			return
		}
		successID++
		str1 := color.CyanString("[%d] GoStmt recovered >> %s\n", successID, lineNo(pass, gs))
		str2 := color.GreenString("\nrecover() at %s\n", lineNo(pass, recoverAt))
		code2 := nodeCode(pass, recoverAt, 1)
		if debug {
			log.Println(str1 + code1 + str2 + code2)
		}
	}()
	switch vt0 := gs.Call.Fun.(type) {
	case *ast.FuncLit:
		// go func(){}
		if hasRecover(vt0.Body) {
			return true
		}
	case *ast.Ident:
		// go goFuncWithoutRecover()
		fd, ok := vt0.Obj.Decl.(*ast.FuncDecl) // fd 是 goFuncWithoutRecover 定义
		if !ok {
			return true
		}
		if hasRecover(fd.Body) {
			return true
		}
	case *ast.SelectorExpr:
		// go abc.fn1(user.fn2)
		ov := pass.TypesInfo.ObjectOf(vt0.Sel)
		astFile := findAstFileByObject(pass, ov)
		if astFile == nil {
			pass.Reportf(gs.Pos(), "cannot find *ast.File")
			return false
		}

		funcNode := findFuncDeclNode(astFile, vt0.Sel.Name)
		if funcNode == nil {
			pass.Reportf(gs.Pos(), "cannot find *ast.FuncDecl")
			return false
		}
		if hasRecover(funcNode.Body) {
			return true
		}
	default:
		pass.Reportf(gs.Pos(), "unsupported type: %T", gs.Call.Fun)
	}
	failID++
	pass.Reportf(gs.Pos(), "[%d] goroutine not recovered, func type is %T \n%s", failID, gs.Call.Fun, code1)
	return false
}

func lineNo(pass *analysis.Pass, node ast.Node) string {
	pos := pass.Fset.Position(node.Pos())
	rn, _ := filepath.Rel(wd, pos.Filename)
	return fmt.Sprintf("%s:%d", rn, pos.Line)
}

func nodeCode(pass *analysis.Pass, node ast.Node, line int) string {
	bf := &bytes.Buffer{}
	format.Node(bf, pass.Fset, node)
	lines := strings.SplitN(bf.String(), "\n", line+1)
	if len(lines) > line {
		lines = lines[:line]
	}
	lines = strings.Split(strings.TrimSpace(strings.Join(lines, "\n")), "\n")
	pos := pass.Fset.Position(node.Pos())
	for i := 0; i < len(lines); i++ {
		lines[i] = fmt.Sprintf("%-5d %s", i+pos.Line, lines[i])
	}
	return strings.Join(lines, "\n")
}

func findFuncDeclNode(f *ast.File, name string) *ast.FuncDecl {
	for _, d := range f.Decls {
		if funcDecl, ok := d.(*ast.FuncDecl); ok && funcDecl.Name.Name == name {
			return funcDecl
		}
	}
	return nil
}

func findAstFileByObject(pass *analysis.Pass, ov types.Object) *ast.File {
	tokenFile := pass.Fset.File(ov.Pos())
	for _, astFile := range pass.Files {
		tokenFile2 := pass.Fset.File(astFile.Pos())
		if tokenFile.Name() == tokenFile2.Name() {
			return astFile
		}
	}
	mod := parser.Mode(0) | parser.ParseComments
	f, err := parser.ParseFile(pass.Fset, tokenFile.Name(), nil, mod)
	if err != nil {
		pass.Reportf(ov.Pos(), "ParseFile %s failed: %v", tokenFile.Name(), err)
	} else {
		pass.Files = append(pass.Files, f)
	}
	return f
}

func hasRecover(bs *ast.BlockStmt) bool {
	for _, blockStmt := range bs.List {
		deferStmt, ok := blockStmt.(*ast.DeferStmt) // 是否包含defer 语句
		if !ok {
			continue
		}
		switch vt0 := deferStmt.Call.Fun.(type) {
		// case *ast.SelectorExpr:
		// 	// 判断是否defer中包含  helper.Recover()
		// 	if vt0.Sel.Name == "Recover" {
		// 		return true
		// 	}
		case *ast.FuncLit:
			// 判断是否有 defer func(){ }()
			for i := range vt0.Body.List {
				stmt := vt0.Body.List[i]
				if isStmtRecovered(stmt) {
					recoverAt = stmt
					return true
				}
			}
		}
	}
	return false
}

func isStmtRecovered(stmt ast.Stmt) bool {
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
			return false
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
	return false
}

var recoverAt ast.Node

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
