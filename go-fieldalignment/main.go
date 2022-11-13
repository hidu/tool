// Copyright(C) 2022 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2022/9/3

package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// fork from
// https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go

func main() {
	singlechecker.Main(Analyzer)
}

func doFix(pass *analysis.Pass, node *ast.StructType, indexes []int) ([]byte, error) {
	var buf1 bytes.Buffer
	if err := format.Node(&buf1, pass.Fset, node); err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	p := `package m 
type User `
	code := p + buf1.String()
	df, err := decorator.ParseFile(fset, "code.go", []byte(code), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	dst.Inspect(df, func(node dst.Node) bool {
		sn, ok := node.(*dst.StructType)
		if !ok {
			return true
		}
		var flat []*dst.Field
		for _, f := range sn.Fields.List {
			if len(f.Names) <= 1 {
				flat = append(flat, f)
				continue
			}
			for _, name := range f.Names {
				nf := &dst.Field{
					Names: []*dst.Ident{name},
					Type:  copyExpr(f.Type),
					Tag:   f.Tag,
					Decs:  f.Decs,
				}
				flat = append(flat, nf)
			}
		}

		// fix for
		//  name chan struct{}
		if len(indexes) != len(flat) {
			return true
		}

		// Sort fields according to the optimal order.
		var reordered []*dst.Field
		for _, index := range indexes {
			reordered = append(reordered, flat[index])
		}
		sn.Fields.List = reordered
		return true
	})

	bf := &bytes.Buffer{}
	decorator.Fprint(bf, df)

	after := bf.Bytes()
	after = bytes.TrimSpace(after[len(p):])
	return after, nil
}

func copyExpr(tp dst.Expr) dst.Expr {
	if tp == nil {
		return nil
	}
	switch val := tp.(type) {
	case *dst.Ident:
		v := &dst.Ident{}
		*v = *val
		return v
	case *dst.StarExpr:
		v := &dst.StarExpr{}
		*v = *val
		v.X = copyExpr(v.X)
		return v
	case *dst.SelectorExpr:
		v := &dst.SelectorExpr{}
		*v = *val
		v.X = copyExpr(v.X)
		v.Sel = copyIdent(v.Sel)
		return v
	}
	return tp
}

func copyIdent(id *dst.Ident) *dst.Ident {
	if id == nil {
		return nil
	}
	n := &dst.Ident{}
	*n = *id
	return n
}
