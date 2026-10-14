package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func writeSource(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertBounds(t *testing.T, p packet) {
	t.Helper()
	ok, err := fits(p, false)
	if err != nil || !ok {
		t.Fatalf("packet outside bounds: files=%d functions=%d lines=%d error=%v", p.Files, len(p.Functions), p.SourceLines, err)
	}
	encoded, err := json.MarshalIndent(p, "", "  ")
	if err != nil || len(encoded)+1 > maxPacketBytes {
		t.Fatalf("serialized packet exceeds bound: bytes=%d error=%v", len(encoded)+1, err)
	}
	for _, fn := range p.Functions {
		if len(fn.Source) > maxExcerptBytes || strings.Count(fn.Source, "\n")+1 > maxExcerptLines || !utf8.ValidString(fn.Source) {
			t.Fatalf("excerpt outside bounds: %s:%d", fn.File, fn.StartLine)
		}
	}
}

func TestInventoryUsesASTAndExcludesTestsAndSymlinks(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "pkg/example/a.go", "//go:build windows\n\npackage example\n// func NotAFunction() {}\ntype Thing struct{}\nfunc (t *Thing) Method() {}\nfunc Generic[T any](v T) T { return v }\n")
	writeSource(t, root, "pkg/example/a_test.go", "package example\nfunc TestIgnored() {}")
	writeSource(t, root, "pkg/example/testdata/broken.go", "deliberately invalid fixture")
	writeSource(t, root, "pkg/example/no-functions.go", "package example\nvar Text = `func NotReal() {}`")
	if err := os.Symlink(filepath.Join(root, "pkg/example/a.go"), filepath.Join(root, "pkg/example/link.go")); err != nil {
		t.Fatal(err)
	}
	functions, err := inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 2 || functions[0].Name != "Method" || functions[1].Name != "Generic" {
		t.Fatalf("incorrect AST inventory: %+v", functions)
	}
	if !strings.Contains(functions[0].Source, "(t *Thing)") || functions[0].StartLine != 6 {
		t.Fatalf("receiver/source positions missing: %+v", functions[0])
	}
}

func TestSkewedPackagesBoundedFairRotationAndCoverage(t *testing.T) {
	root := t.TempDir()
	for i := range 500 {
		writeSource(t, root, fmt.Sprintf("pkg/large/f%03d.go", i), fmt.Sprintf("package large\nfunc Function%d() { println(%d) }\n", i, i))
	}
	writeSource(t, root, "pkg/small/a.go", "package small\nfunc Small() { println(\"small\") }\n")
	functions, err := inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	batches, err := rotation(functions)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 168 || !strings.HasPrefix(batches[1].Target, "pkg/small:") {
		t.Fatal("small package must get its turn before the large package's second slice")
	}
	smallVisits := 0
	for day := range 84 {
		p, err := selectPacket(functions, start.AddDate(0, 0, day))
		if err != nil {
			t.Fatal(err)
		}
		assertBounds(t, p)
		if p.TotalFunctions != 501 || len(p.Targets) != 2 || p.CoverageDays != 84 {
			t.Fatalf("unexpected inventory/targets: %+v", p)
		}
		for _, fn := range p.Functions {
			seen[fn.File+":"+fn.Name] = true
			if fn.Package == "small" {
				smallVisits++
			}
		}
		again, err := selectPacket(functions, start.AddDate(0, 0, day))
		if err != nil || !reflect.DeepEqual(p, again) {
			t.Fatal("rotation is not deterministic")
		}
	}
	if len(seen) != len(functions) {
		t.Fatalf("incomplete rotation coverage: %d/%d", len(seen), len(functions))
	}
	if smallVisits != 1 {
		t.Fatal("do not repeatedly analyze an exhausted tiny package")
	}
}

func TestOddPackageRotationAndFunctionDenseFiles(t *testing.T) {
	functions := []function{}
	for pkg := range 3 {
		for i := range 33 {
			functions = append(functions, function{File: fmt.Sprintf("pkg/p%d/all.go", pkg), Package: fmt.Sprintf("p%d", pkg), Name: fmt.Sprintf("F%d", i), Source: "func F() {}"})
		}
	}
	seen := map[string]bool{}
	packages := map[string]bool{}
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for day := range 15 {
		p, err := selectPacket(functions, date.AddDate(0, 0, day))
		if err != nil {
			t.Fatal(err)
		}
		assertBounds(t, p)
		if len(p.Targets) != 2 {
			t.Fatal("expected two bounded slices")
		}
		for _, fn := range p.Functions {
			seen[fn.File+":"+fn.Name] = true
			packages[fn.Package] = true
		}
	}
	if len(seen) != 99 || len(packages) != 3 {
		t.Fatalf("rotation failed to cover dense files/odd package count: %d functions, %d packages", len(seen), len(packages))
	}
}

func TestOversizedFunctionsLinesBytesAndEmptyInventory(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "pkg/large/a.go", "package large\nfunc LongLines() {\n"+strings.Repeat("println(\"line\")\n", 2000)+"}\nfunc LongBytes() { println(`"+strings.Repeat("界\\\"", 10000)+"`) }\n")
	functions, err := inventory(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := selectPacket(functions, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	assertBounds(t, p)
	if len(p.Functions) != 2 || !p.Functions[0].Truncated || !p.Functions[1].Truncated {
		t.Fatal("oversized functions must remain in rotation with explicit truncation")
	}
	empty, err := selectPacket(nil, time.Now())
	if err != nil || len(empty.Functions) != 0 || len(empty.Targets) != 0 {
		t.Fatalf("empty inventory must produce honest empty packet: %+v %v", empty, err)
	}
}

func TestSourceLinesAndSerializedBytesLimitActualWorkload(t *testing.T) {
	functions := []function{}
	for i := range 30 {
		source := strings.Repeat("\"\t\\", maxExcerptBytes/3)
		if i < 15 {
			source = strings.Repeat("x\n", 79) + "x"
		}
		functions = append(functions, function{File: "pkg/large/all.go", Name: strconv.Itoa(i), Source: source})
	}
	batches, err := slices(functions)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, batch := range batches {
		p := packet{Functions: batch}
		measure(&p)
		assertBounds(t, p)
		ok, err := fits(p, true)
		if err != nil || !ok {
			t.Fatal("per-package budget exceeded")
		}
		total += len(batch)
	}
	if total != len(functions) || len(batches) <= 4 {
		t.Fatal("line/escaped JSON budgets must partition, never discard functions")
	}
}

func TestParseFailureIsNotEmptySuccess(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "pkg/broken/a.go", "package broken\nfunc {")
	if _, err := inventory(root); err == nil {
		t.Fatal("invalid Go source must fail explicitly")
	}
}
