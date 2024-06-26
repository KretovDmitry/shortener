// Package exitmain defines an Analyzer that report os.Exit call
// inside main function of the main package.
package exitinmain

import (
	"errors"
	"go/ast"
	"log"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is a go analysis package analyzer implementation.
var Analyzer = &analysis.Analyzer{
	Name:     "exitinmain",
	Doc:      "reports os.Exit call inside main function of the main package",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Get the inspector. This will not panic because inspect.Analyzer is part
	// of `Requires`. go/analysis will populate the `pass.ResultOf` map with
	// the prerequisite analyzers.
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		message := "failed to obtain the prerequisite inspector"
		log.Println(message)
		return nil, errors.New(message)
	}

	// The inspector has a `filter` feature that enables type-based filtering
	// The anonymous function will be only called for the ast nodes whose type
	// matches an element in the filter.
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.SelectorExpr)(nil),
	}

	var insideMain bool

	// This is basically the same as ast.Inspect(), only we don't return a
	// boolean anymore as it'll visit all the nodes based on the filter.
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch x := n.(type) {
		case *ast.File:
			if !isMainPkg(x) {
				return
			}
		case *ast.FuncDecl:
			main := isMainFunc(x)
			if insideMain && !main {
				insideMain = false
				return
			}
			insideMain = main
		case *ast.SelectorExpr:
			if insideMain && isOsExit(x) {
				pass.Reportf(x.Pos(), "os.Exit call inside main function")
				return
			}
		}
	})

	return nil, nil
}

func isMainPkg(x *ast.File) bool {
	return x.Name.Name == "main"
}

func isMainFunc(x *ast.FuncDecl) bool {
	return x.Name.Name == "main"
}

func isOsExit(x *ast.SelectorExpr) bool {
	if x.X == nil {
		return false
	}

	ident, ok := x.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "os" && x.Sel.Name == "Exit"
}
