package exitinmain_test

import (
	"testing"

	"github.com/KretovDmitry/shortener/pkg/exitinmain"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), exitinmain.Analyzer, "./...")
}
