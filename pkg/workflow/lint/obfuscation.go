package lint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/goccy/go-yaml"
	"github.com/rhysd/actionlint"
)

func ObfuscationRule() Rule {
	return Rule{
		ID: "obfuscation", Description: "Replace literal-only string expressions with plain values",
		Severity: Warning, Targets: ActionsAST | AWAST, Check: checkObfuscation,
	}
}

func checkObfuscation(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	visit := func(path, text string) error {
		return taint.VisitInterpolations(ctx, text, path, func(_, _, line int, node actionlint.ExprNode) error {
			if _, literal := node.(*actionlint.StringNode); literal {
				diagnostic := Diagnostic{
					Path:    path,
					Message: "Literal string is hidden inside an Actions expression; use the plain quoted value instead",
				}
				if strings.HasSuffix(path, ".run") || strings.HasSuffix(path, ".with.script") {
					diagnostic.Line = line
				}
				diagnostics = append(diagnostics, diagnostic)
			}
			return ctx.Err()
		})
	}
	if input.Workflow != nil {
		root, err := actions.Lower(input.Workflow)
		if err != nil {
			return nil, errors.New("obfuscation cannot lower the Actions AST; correct the workflow before linting")
		}
		err = walkExpressionScalars(ctx, root, "", visit)
		return diagnostics, err
	}
	blocks, err := aw.ExecutionBlocks(ctx, input.Document)
	failures := []error{err}
	for _, block := range blocks {
		// Do not treat prompts, grader code, or unrelated AW configuration as
		// runner-interpolated text.
		for _, key := range []string{"name", "run-name", "env", "outputs", "with", "defaults", "concurrency", "runs-on", "strategy", "container", "services"} {
			failures = append(failures, walkExpressionScalars(ctx, block.Fields[key], block.FieldPath(key), visit))
		}
		for _, step := range block.Steps {
			failures = append(failures, walkExpressionScalars(ctx, step.Fields, step.Path, visit))
		}
	}
	return diagnostics, errors.Join(failures...)
}

func walkExpressionScalars(ctx context.Context, value any, path string, visit func(string, string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fieldPath := func(key string) string {
		if path == "" {
			return key
		}
		return path + "." + key
	}
	var failures []error
	switch value := value.(type) {
	case string:
		return visit(path, value)
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			failures = append(failures, walkExpressionScalars(ctx, value[key], fieldPath(key), visit))
		}
	case map[string]string:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			failures = append(failures, walkExpressionScalars(ctx, value[key], fieldPath(key), visit))
		}
	case yaml.MapSlice:
		for _, item := range value {
			key, ok := item.Key.(string)
			if !ok {
				return errors.New("obfuscation requires string mapping keys; correct the workflow before linting")
			}
			failures = append(failures, walkExpressionScalars(ctx, item.Value, fieldPath(key), visit))
		}
	case []any:
		for index, item := range value {
			failures = append(failures, walkExpressionScalars(ctx, item, fmt.Sprintf("%s[%d]", path, index), visit))
		}
	case []map[string]any:
		for index, item := range value {
			failures = append(failures, walkExpressionScalars(ctx, item, fmt.Sprintf("%s[%d]", path, index), visit))
		}
	case []string:
		for index, item := range value {
			failures = append(failures, walkExpressionScalars(ctx, item, fmt.Sprintf("%s[%d]", path, index), visit))
		}
	case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
	default:
		return fmt.Errorf("obfuscation cannot inspect a non-scalar configuration value at %s; correct the workflow before linting", path)
	}
	return errors.Join(failures...)
}
