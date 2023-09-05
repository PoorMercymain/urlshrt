package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var OsExitAnalyzer = &analysis.Analyzer{
	Name: "osexitanalyzer",
	Doc:  "check for os.Exit usage in main func of package main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	files := make([]*ast.File, 0)

	for _, file := range pass.Files {
		if file.Name.Name == "main" {
			files = append(files, file)
		}
	}

	mainFuncs := make([]*ast.FuncDecl, 0)

	for _, file := range files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
				mainFuncs = append(mainFuncs, fn)
				break
			}
		}
	}

	if len(mainFuncs) == 0 {
		return nil, nil
	}

	for _, mainFunc := range mainFuncs {
		ast.Inspect(mainFunc, func(n ast.Node) bool {
			if callExpr, ok := n.(*ast.CallExpr); ok {
				if funIdent, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					if pkgIdent, ok := funIdent.X.(*ast.Ident); ok {
						if pkgIdent.Name == "os" && funIdent.Sel.Name == "Exit" {
							pass.Reportf(callExpr.Pos(), "os.Exit is not allowed in main func of main package")
						}
					}
				}
			}
			return true
		})
	}

	return nil, nil
}
