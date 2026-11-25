package main

import (
	"go/ast"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	// 1. Стандартные анализаторы (отсутствующие в go vet)
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/gofix"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/scannererr"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/sqlrowserr"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"golang.org/x/tools/go/analysis/passes/waitgroup"

	// 2. Анализаторы staticcheck (классы SA, S, ST)
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	// 3. Стабильные публичные анализаторы (ПРОВЕРЕННЫЕ пути)
	"github.com/gostaticanalysis/nilerr"
	"github.com/nishanths/predeclared/passes/predeclared"
)

func main() {
	var analyzers []*analysis.Analyzer

	// 1. Стандартные
	analyzers = append(analyzers,
		appends.Analyzer,
		atomicalign.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		fieldalignment.Analyzer,
		framepointer.Analyzer,
		gofix.Analyzer,
		hostport.Analyzer,
		httpmux.Analyzer,
		loopclosure.Analyzer,
		nilness.Analyzer,
		reflectvaluecompare.Analyzer,
		scannererr.Analyzer,
		shadow.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		sqlrowserr.Analyzer,
		stdversion.Analyzer,
		testinggoroutine.Analyzer,
		timeformat.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
		waitgroup.Analyzer,
	)

	// Staticcheck
	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}
	for _, a := range simple.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}
	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// Публичные анализаторы
	analyzers = append(analyzers,
		nilerr.Analyzer,      // Проверяет return err после if err != nil
		predeclared.Analyzer, // Проверяет затенение встроенных имен (например, var copy = 1)
	)

	var osExitAnalyzer = &analysis.Analyzer{
		Name: "osExit",
		Doc:  "check for call os.Exit() in main",
		Run:  run,
	}

	analyzers = append(analyzers, osExitAnalyzer)
	multichecker.Main(analyzers...)
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		filename := pass.Fset.File(f.Pos()).Name()
		if filepath.Base(filename) != "main.go" {
			continue
		}
		if pass.Pkg.Name() != "main" {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}

			if sel.Sel.Name == "Exit" {
				pass.Reportf(call.Pos(), "вызов os.Exit() запрещён")
			}

			return true
		})
	}

	return nil, nil
}
