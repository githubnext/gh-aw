package lint

import (
	"context"
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/taint"
)

func GraderFileIORule() Rule {
	return Rule{
		ID: "grader-file-io", Description: "Refuse filesystem access and module loading in custom JavaScript graders",
		Severity: Error, Targets: AWAST, Check: checkGraderFileIO,
	}
}

type graderJavaScript struct {
	path   string
	source string
}

func checkGraderFileIO(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Document == nil {
		return nil, nil
	}
	scripts, err := customGraderJavaScript(input)
	if err != nil || len(scripts) == 0 {
		return nil, err
	}
	if input.JavaScript == nil || input.JavaScript.Parser == nil {
		var diagnostics []Diagnostic
		for _, script := range scripts {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: Warning, Path: script.path,
				Message: "Custom grader file-I/O analysis unavailable; configure --javascript-module with the pinned native TypeScript compiler module",
			})
		}
		return diagnostics, nil
	}
	results, err := parseGraderJavaScript(ctx, scripts, input.JavaScript.Parser)
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	var failures []error
	for i, result := range results {
		if i < 0 || i >= len(scripts) {
			return diagnostics, errors.New("custom grader JavaScript parser returned an invalid batch")
		}
		script := scripts[i]
		if result.Err != nil || result.Node == nil || result.Node.Kind != "SourceFile" {
			failures = append(failures, fmt.Errorf("cannot analyze %s: JavaScript AST unavailable; correct the script syntax and configure the pinned native TypeScript compiler", script.path))
			continue
		}
		found, err := graderFileIOAST(ctx, result.Node, script.path)
		diagnostics = append(diagnostics, found...)
		failures = append(failures, err)
	}
	return diagnostics, errors.Join(failures...)
}

func customGraderJavaScript(input Input) ([]graderJavaScript, error) {
	graders, err := policyMapping(input.Document.Frontmatter["graders"], "graders")
	if err != nil {
		return nil, err
	}
	var scripts []graderJavaScript
	for _, id := range sliceutil.SortedKeys(graders) {
		fields, mapping := graders[id].(map[string]any)
		if !mapping {
			continue // Built-in graders accept scalar configuration.
		}
		value, exists := fields["script"]
		if !exists {
			continue
		}
		path := "graders." + id + ".script"
		source, valid := value.(string)
		if !valid {
			return nil, fmt.Errorf("grader-file-io requires a JavaScript string at %s", path)
		}
		scripts = append(scripts, graderJavaScript{path: path, source: source})
	}
	return scripts, nil
}

func parseGraderJavaScript(ctx context.Context, scripts []graderJavaScript, parser taint.JavaScriptParser) ([]taint.JavaScriptParseResult, error) {
	var results []taint.JavaScriptParseResult
	if batch, ok := parser.(taint.JavaScriptBatchParser); ok {
		sources := make([]string, 0, len(scripts))
		for _, script := range scripts {
			sources = append(sources, script.source)
		}
		var err error
		results, err = batch.ParseBatch(ctx, sources)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, errors.New("custom grader JavaScript AST parsing failed; configure the pinned native TypeScript compiler")
		}
	} else {
		for _, script := range scripts {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			node, err := parser.Parse(ctx, script.source)
			results = append(results, taint.JavaScriptParseResult{Node: node, Err: err})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(results) != len(scripts) {
		return nil, errors.New("custom grader JavaScript parser returned an invalid batch")
	}
	return results, nil
}
