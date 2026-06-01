//go:build !integration

package seenmapbool

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestSeenMapBool(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "seenmapbool")
}

func TestAnalyzerFields(t *testing.T) {
	if Analyzer.Name != "seenmapbool" {
		t.Errorf("expected Name %q, got %q", "seenmapbool", Analyzer.Name)
	}
	if Analyzer.Doc == "" {
		t.Error("Doc must not be empty")
	}
	if Analyzer.URL == "" {
		t.Error("URL must not be empty")
	}
}
