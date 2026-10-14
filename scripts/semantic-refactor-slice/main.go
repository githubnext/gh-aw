package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxFiles        = 6
	maxFunctions    = 16
	maxSourceLines  = 600
	maxPacketBytes  = 48 * 1024
	maxExcerptLines = 80
	maxExcerptBytes = 4096
	defaultOutput   = "/tmp/gh-aw/agent/semantic-function-refactor/analysis.json"
)

type function struct {
	File      string `json:"file"`
	Package   string `json:"package"`
	Name      string `json:"name"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Source    string `json:"source"`
	Truncated bool   `json:"truncated"`
}

type packet struct {
	Date           string     `json:"date"`
	TotalFunctions int        `json:"total_functions"`
	RotationSlices int        `json:"rotation_slices"`
	CoverageDays   int        `json:"coverage_days"`
	Files          int        `json:"files"`
	SourceLines    int        `json:"source_lines"`
	Targets        []string   `json:"targets"`
	Functions      []function `json:"functions"`
}

func excerpt(source string) (string, bool) {
	lines := strings.SplitAfter(source, "\n")
	if len(lines) > maxExcerptLines {
		lines = lines[:maxExcerptLines]
	}
	bounded := strings.Join(lines, "")
	if len(bounded) > maxExcerptBytes {
		bounded = bounded[:maxExcerptBytes]
		for !utf8.ValidString(bounded) {
			bounded = bounded[:len(bounded)-1]
		}
	}
	bounded = strings.TrimSuffix(bounded, "\n")
	return bounded, len(bounded) != len(source)
}

func inventory(root string) ([]function, error) {
	functions := []function{}
	err := filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == "testdata" {
			return filepath.SkipDir
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, source, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse analysis source %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			start, end := fset.Position(fn.Pos()), fset.Position(fn.End())
			text, truncated := excerpt(string(source[start.Offset:end.Offset]))
			functions = append(functions, function{
				File: filepath.ToSlash(rel), Package: file.Name.Name, Name: fn.Name.Name,
				StartLine: start.Line, EndLine: end.Line, Source: text, Truncated: truncated,
			})
		}
		return nil
	})
	return functions, err
}

func measure(p *packet) {
	files := map[string]struct{}{}
	p.SourceLines = 0
	for _, fn := range p.Functions {
		files[fn.File] = struct{}{}
		p.SourceLines += strings.Count(fn.Source, "\n") + 1
	}
	p.Files = len(files)
}

func fits(p packet, perPackage bool) (bool, error) {
	measure(&p)
	encoded, err := json.MarshalIndent(p, "", "  ")
	files, functions, lines, bytes := maxFiles, maxFunctions, maxSourceLines, maxPacketBytes
	if perPackage {
		files, functions, lines, bytes = files/2, functions/2, lines/2, bytes/2-1024
	}
	return len(p.Functions) <= functions && p.Files <= files &&
		p.SourceLines <= lines && len(encoded)+1 <= bytes, err
}

func slices(functions []function) ([][]function, error) {
	result := [][]function{}
	current := []function{}
	for _, fn := range functions {
		candidate := append(append([]function{}, current...), fn)
		ok, err := fits(packet{Date: "2000-01-01", TotalFunctions: len(functions), Functions: candidate}, true)
		if err != nil {
			return nil, err
		}
		if !ok {
			if len(current) == 0 {
				return nil, fmt.Errorf("function %s:%d exceeds analysis packet bounds", fn.File, fn.StartLine)
			}
			result = append(result, current)
			current = []function{fn}
			ok, err = fits(packet{Date: "2000-01-01", TotalFunctions: len(functions), Functions: current}, true)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("function %s:%d exceeds analysis packet bounds", fn.File, fn.StartLine)
			}
		} else {
			current = candidate
		}
	}
	if len(current) > 0 {
		result = append(result, current)
	}
	return result, nil
}

type packageSlice struct {
	Target    string
	Functions []function
}

func rotation(functions []function) ([]packageSlice, error) {
	byPackage := map[string][]function{}
	for _, fn := range functions {
		dir := filepath.ToSlash(filepath.Dir(fn.File))
		byPackage[dir] = append(byPackage[dir], fn)
	}
	dirs := make([]string, 0, len(byPackage))
	for dir := range byPackage {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	bySlice := make(map[string][][]function, len(dirs))
	longest := 0
	for _, dir := range dirs {
		batches, err := slices(byPackage[dir])
		if err != nil {
			return nil, err
		}
		bySlice[dir] = batches
		longest = max(longest, len(batches))
	}
	result := []packageSlice{}
	// Interleave packages without repeating exhausted ones: skew cannot waste
	// coverage on the same tiny package while large packages still have work.
	for index := range longest {
		for _, dir := range dirs {
			batches := bySlice[dir]
			if index >= 0 && index < len(batches) {
				result = append(result, packageSlice{
					Target:    fmt.Sprintf("%s: slice %d/%d", dir, index+1, len(batches)),
					Functions: batches[index],
				})
			}
		}
	}
	return result, nil
}

func selectPacket(functions []function, date time.Time) (packet, error) {
	p := packet{Date: date.UTC().Format(time.DateOnly), TotalFunctions: len(functions), Targets: []string{}, Functions: []function{}}
	batches, err := rotation(functions)
	if err != nil {
		return packet{}, err
	}
	count := len(batches)
	if count == 0 {
		return p, nil
	}
	slots := min(2, count)
	p.RotationSlices = count
	p.CoverageDays = (count + slots - 1) / slots
	day := date.UTC().Unix() / 86400
	if day < 0 {
		return packet{}, errors.New("rotation date must be on or after 1970-01-01")
	}
	for offset := range slots {
		index := int((day*int64(slots) + int64(offset)) % int64(count))
		if index < 0 || index >= len(batches) {
			return packet{}, errors.New("rotation index outside analysis slices")
		}
		p.Targets = append(p.Targets, batches[index].Target)
		p.Functions = append(p.Functions, batches[index].Functions...)
	}
	measure(&p)
	ok, err := fits(p, false)
	if err != nil {
		return packet{}, err
	}
	if !ok {
		return packet{}, errors.New("combined analysis packet exceeds workload bounds")
	}
	return p, nil
}

func run() error {
	root := flag.String("root", ".", "repository root")
	output := flag.String("output", defaultOutput, "analysis packet path")
	dateArg := flag.String("date", time.Now().UTC().Format(time.DateOnly), "UTC rotation date (YYYY-MM-DD)")
	flag.Parse()
	date, err := time.Parse(time.DateOnly, *dateArg)
	if err != nil {
		return fmt.Errorf("invalid rotation date: %w", err)
	}
	functions, err := inventory(*root)
	if err != nil {
		return err
	}
	p, err := selectPacket(functions, date)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if len(encoded)+1 > maxPacketBytes {
		return fmt.Errorf("analysis packet exceeds %d bytes", maxPacketBytes)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(*output, append(encoded, '\n'), 0o644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
