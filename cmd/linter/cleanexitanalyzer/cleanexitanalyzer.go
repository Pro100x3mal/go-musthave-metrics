package cleanexitanalyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "cleanexitanalyzer",
		Doc:  "reports usage of panic and forbidden log.Fatal/os.Exit outside main.main",
		Run:  run,
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		inMainPackage := pass.Pkg.Name() == "main"

		for _, cg := range file.Comments {
			for _, c := range cg.List {
				if strings.Contains(c.Text, "nolint:cleanexitanalyzer") {
					return nil, nil
				}
			}
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				if !(inMainPackage && isInsideMainFunc(pass, call)) {
					pass.Reportf(call.Pos(), "use of panic detected")
				}
				return true
			}

			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil {
					pkg := obj.Pkg()
					if pkg != nil {
						switch pkg.Path() {
						case "log":
							if sel.Sel.Name == "Fatal" && !(inMainPackage && isInsideMainFunc(pass, call)) {
								pass.Reportf(call.Pos(), "log.Fatal used outside main.main")
							}
						case "os":
							if sel.Sel.Name == "Exit" && !(inMainPackage && isInsideMainFunc(pass, call)) {
								pass.Reportf(call.Pos(), "os.Exit used outside main.main")
							}
						}
					}
				}
			}

			return true
		})
	}
	return nil, nil
}

func isInsideMainFunc(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				continue
			}
			if call.Pos() >= fn.Pos() && call.Pos() <= fn.End() {
				return true
			}
		}
	}
	return false
}
