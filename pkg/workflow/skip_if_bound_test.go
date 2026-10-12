//go:build !integration

package workflow

import (
	"fmt"
	"testing"
)

func TestExtractSkipIfBoundFromOn(t *testing.T) {
	compiler := NewCompiler()
	conditions := []struct {
		key, field, example string
		extract             func(map[string]any, ...*WorkflowData) (*skipIfBoundConfig, error)
	}{
		{"skip-if-match", "max", "is:issue is:open", func(f map[string]any, data ...*WorkflowData) (*skipIfBoundConfig, error) {
			config, err := compiler.extractSkipIfMatchFromOn(f, data...)
			if config == nil {
				return nil, err
			}
			return &skipIfBoundConfig{Query: config.Query, Bound: config.Max, Scope: config.Scope}, err
		}},
		{"skip-if-no-match", "min", "is:pr is:open", func(f map[string]any, data ...*WorkflowData) (*skipIfBoundConfig, error) {
			config, err := compiler.extractSkipIfNoMatchFromOn(f, data...)
			if config == nil {
				return nil, err
			}
			return &skipIfBoundConfig{Query: config.Query, Bound: config.Min, Scope: config.Scope}, err
		}},
	}
	for _, condition := range conditions {
		t.Run(condition.key, func(t *testing.T) {
			key, field, example := condition.key, condition.field, condition.example
			object := func(bound any) map[string]any {
				return map[string]any{"query": "custom query", field: bound, "scope": "none"}
			}
			frontmatter := func(raw any) map[string]any {
				return map[string]any{"on": map[string]any{key: raw}}
			}
			otherField := "min"
			if field == "min" {
				otherField = "max"
			}
			tests := []struct {
				name string
				raw  any
				want skipIfBoundConfig
				err  string
			}{
				{name: "string", raw: "custom query", want: skipIfBoundConfig{Query: "custom query", Bound: 1}},
				{name: "empty query", raw: "", want: skipIfBoundConfig{Bound: 1}},
				{name: "default bound", raw: map[string]any{"query": "custom query"}, want: skipIfBoundConfig{Query: "custom query", Bound: 1}},
				{name: "minimum bound", raw: object(1), want: skipIfBoundConfig{Query: "custom query", Bound: 1, Scope: "none"}},
				{name: "int", raw: object(3), want: skipIfBoundConfig{Query: "custom query", Bound: 3, Scope: "none"}},
				{name: "int64", raw: object(int64(4)), want: skipIfBoundConfig{Query: "custom query", Bound: 4, Scope: "none"}},
				{name: "uint64", raw: object(uint64(5)), want: skipIfBoundConfig{Query: "custom query", Bound: 5, Scope: "none"}},
				{name: "float64", raw: object(float64(6)), want: skipIfBoundConfig{Query: "custom query", Bound: 6, Scope: "none"}},
				{name: "fractional coercion", raw: object(3.9), want: skipIfBoundConfig{Query: "custom query", Bound: 3, Scope: "none"}},
				{name: "other bound ignored", raw: map[string]any{"query": "custom query", "max": 7, "min": 8}, want: skipIfBoundConfig{Query: "custom query", Bound: map[string]int{"max": 7, "min": 8}[field]}},
				{name: "other bound alone ignored", raw: map[string]any{"query": "custom query", otherField: 9}, want: skipIfBoundConfig{Query: "custom query", Bound: 1}},
				{name: "missing query", raw: map[string]any{}, err: fmt.Sprintf("%s object must have a 'query' field. Example:\n  %s:\n    query: %q\n    %s: 3", key, key, example, field)},
				{name: "invalid query", raw: map[string]any{"query": 3}, err: fmt.Sprintf("%s 'query' field must be a string, got int. Example:\n  %s:\n    query: %q", key, key, example)},
				{name: "invalid bound type", raw: object("3"), err: fmt.Sprintf("%s '%s' field must be an integer, got string. Example: %s: 3", key, field, field)},
				{name: "unsupported integer type", raw: object(int32(3)), err: fmt.Sprintf("%s '%s' field must be an integer, got int32. Example: %s: 3", key, field, field)},
				{name: "null bound", raw: object(nil), err: fmt.Sprintf("%s '%s' field must be an integer, got <nil>. Example: %s: 3", key, field, field)},
				{name: "zero bound", raw: object(0), err: fmt.Sprintf("%s '%s' field must be at least 1, got 0. Example:\n  %s:\n    query: %q\n    %s: 3", key, field, key, example, field)},
				{name: "negative bound", raw: object(-2), err: fmt.Sprintf("%s '%s' field must be at least 1, got -2. Example:\n  %s:\n    query: %q\n    %s: 3", key, field, key, example, field)},
				{name: "uint64 overflow coerced", raw: object(^uint64(0)), err: fmt.Sprintf("%s '%s' field must be at least 1, got -1. Example:\n  %s:\n    query: %q\n    %s: 3", key, field, key, example, field)},
				{name: "fraction coerces to zero", raw: object(0.9), err: fmt.Sprintf("%s '%s' field must be at least 1, got 0. Example:\n  %s:\n    query: %q\n    %s: 3", key, field, key, example, field)},
				{name: "invalid scope type", raw: map[string]any{"query": "custom query", "scope": true}, err: key + " 'scope' field must be a string, got bool. Example: scope: none"},
				{name: "invalid scope", raw: map[string]any{"query": "custom query", "scope": "repo"}, err: key + " 'scope' field must be \"none\" or omitted, got \"repo\". Example: scope: none"},
				{name: "invalid condition", raw: true, err: fmt.Sprintf("%s value must be a string or object, got bool. Example:\n  %s: %q\n  %s:\n    query: \"is:pr is:open\"\n    %s: 3", key, key, example, key, field)},
				{name: "null condition", raw: nil, err: fmt.Sprintf("%s value must be a string or object, got <nil>. Example:\n  %s: %q\n  %s:\n    query: \"is:pr is:open\"\n    %s: 3", key, key, example, key, field)},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					config, err := condition.extract(frontmatter(test.raw))
					if test.err != "" {
						if err == nil || err.Error() != test.err || config != nil {
							t.Fatalf("got config=%+v, err=%v; want nil config and %q", config, err, test.err)
						}
						return
					}
					if err != nil || config == nil || *config != test.want {
						t.Fatalf("got config=%+v, err=%v; want %+v", config, err, test.want)
					}
				})
			}
			t.Run("on resolution", func(t *testing.T) {
				tests := []struct {
					name string
					f    map[string]any
					data []*WorkflowData
					want *skipIfBoundConfig
					err  string
				}{
					{name: "absent"},
					{name: "string on", f: map[string]any{"on": "push"}},
					{name: "absent condition", f: map[string]any{"on": map[string]any{"push": nil}}},
					{name: "invalid on", f: map[string]any{"on": 3}, err: "invalid on: section format. Expected a string or an object of triggers. Example:\non:\n  issues:\n    types: [opened]"},
					{name: "null on", f: map[string]any{"on": nil}, err: "invalid on: section format. Expected a string or an object of triggers. Example:\non:\n  issues:\n    types: [opened]"},
					{name: "nil data", f: frontmatter("fallback"), data: []*WorkflowData{nil}, want: &skipIfBoundConfig{Query: "fallback", Bound: 1}},
					{name: "missing cache", f: frontmatter("fallback"), data: []*WorkflowData{{}}, want: &skipIfBoundConfig{Query: "fallback", Bound: 1}},
					{name: "nil cached on", f: frontmatter("fallback"), data: []*WorkflowData{{ParsedFrontmatter: &FrontmatterConfig{}}}, want: &skipIfBoundConfig{Query: "fallback", Bound: 1}},
					{name: "cached on wins", f: map[string]any{"on": 3}, data: []*WorkflowData{{ParsedFrontmatter: &FrontmatterConfig{On: map[string]any{key: object(9)}}}}, want: &skipIfBoundConfig{Query: "custom query", Bound: 9, Scope: "none"}},
					{name: "empty cached on wins", f: frontmatter("fallback"), data: []*WorkflowData{{ParsedFrontmatter: &FrontmatterConfig{On: map[string]any{}}}}},
					{name: "only first data used", f: frontmatter("fallback"), data: []*WorkflowData{nil, {ParsedFrontmatter: &FrontmatterConfig{On: map[string]any{key: "cached"}}}}, want: &skipIfBoundConfig{Query: "fallback", Bound: 1}},
				}
				for _, test := range tests {
					t.Run(test.name, func(t *testing.T) {
						config, err := condition.extract(test.f, test.data...)
						if test.err != "" {
							if err == nil || err.Error() != test.err || config != nil {
								t.Fatalf("got config=%+v, err=%v; want nil config and %q", config, err, test.err)
							}
							return
						}
						if err != nil || (config == nil) != (test.want == nil) || (config != nil && *config != *test.want) {
							t.Fatalf("got config=%+v, err=%v; want %+v", config, err, test.want)
						}
					})
				}
			})
		})
	}
}
