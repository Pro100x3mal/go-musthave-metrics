package main

import (
	"github.com/Pro100x3mal/go-musthave-metrics/cmd/linter/cleanexitanalyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(cleanexitanalyzer.NewAnalyzer())
}
