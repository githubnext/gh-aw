//go:build !integration

package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestHelpExamplesAreValid(t *testing.T) {
	var checkExamples func(cmd *cobra.Command)
	checkExamples = func(cmd *cobra.Command) {
		for _, example := range extractCommandExamples(cmd) {
			t.Run(cmd.CommandPath()+" example "+example, func(t *testing.T) {
				validateExample(t, example)
			})
		}
		for _, sub := range cmd.Commands() {
			checkExamples(sub)
		}
	}
	checkExamples(rootCmd)
}

func extractCommandExamples(cmd *cobra.Command) []string {
	examples := make([]string, 0)

	appendFromLine := func(line string) {
		withoutComment := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if withoutComment == "" {
			return
		}
		if !strings.HasPrefix(withoutComment, "gh aw ") {
			return
		}
		examples = append(examples, withoutComment)
	}

	if cmd.Example != "" {
		for _, line := range strings.Split(cmd.Example, "\n") {
			appendFromLine(line)
		}
	}

	inExamples := false
	for _, line := range strings.Split(cmd.Long, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "Examples:") {
			inExamples = true
			continue
		}
		if !inExamples {
			continue
		}
		appendFromLine(line)
	}

	return examples
}

func validateExample(t *testing.T, example string) {
	t.Helper()

	tokens := strings.Fields(example)
	if len(tokens) < 3 {
		t.Fatalf("example %q is too short", example)
	}

	if tokens[0] != "gh" || tokens[1] != "aw" {
		t.Fatalf("example %q must start with 'gh aw'", example)
	}

	cmd, consumed, err := resolveExampleCommand(tokens)
	if err != nil {
		t.Fatalf("example %q has invalid command syntax: %v", example, err)
	}

	validateExampleTokens(t, cmd, tokens[consumed:])
}

func resolveExampleCommand(tokens []string) (*cobra.Command, int, error) {
	current := rootCmd
	consumed := 2 // "gh aw"

	for consumed < len(tokens) {
		token := tokens[consumed]
		if strings.HasPrefix(token, "-") {
			break
		}
		var next *cobra.Command
		for _, sub := range current.Commands() {
			if sub.Name() == token {
				next = sub
				break
			}
		}
		if next == nil {
			break
		}
		current = next
		consumed++
	}

	if current == rootCmd {
		return nil, consumed, pflag.ErrHelp
	}

	return current, consumed, nil
}

func validateExampleTokens(t *testing.T, cmd *cobra.Command, tokens []string) {
	t.Helper()

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if !strings.HasPrefix(token, "-") || token == "-" {
			validatePathToken(t, token)
			continue
		}

		if strings.HasPrefix(token, "--") {
			nameValue := strings.TrimPrefix(token, "--")
			name, _, hasValue := strings.Cut(nameValue, "=")
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				flag = cmd.InheritedFlags().Lookup(name)
			}
			if flag == nil {
				t.Fatalf("unknown flag %q in example for command %q", "--"+name, cmd.CommandPath())
			}
			if hasValue {
				validatePathToken(t, strings.SplitN(nameValue, "=", 2)[1])
				continue
			}
			if flag.Value.Type() == "bool" {
				continue
			}
			if i+1 >= len(tokens) {
				t.Fatalf("flag %q in example for command %q is missing a value", "--"+name, cmd.CommandPath())
			}
			i++
			validatePathToken(t, tokens[i])
			continue
		}

		shorthand := strings.TrimPrefix(token, "-")
		for _, ch := range shorthand {
			flag := cmd.Flags().ShorthandLookup(string(ch))
			if flag == nil {
				flag = cmd.InheritedFlags().ShorthandLookup(string(ch))
			}
			if flag == nil {
				t.Fatalf("unknown shorthand flag %q in example for command %q", "-"+string(ch), cmd.CommandPath())
			}
		}
	}
}

func validatePathToken(t *testing.T, token string) {
	t.Helper()

	trimmed := strings.Trim(token, `"'`)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return
	}
	if strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "[") {
		return
	}
	if !looksLikePath(trimmed) {
		return
	}
	if filepath.IsAbs(trimmed) {
		t.Fatalf("path %q in example must be relative", trimmed)
	}
	if strings.Contains(trimmed, `\`) {
		t.Fatalf("path %q in example must use forward slashes", trimmed)
	}
	if strings.Contains(trimmed, "..") {
		t.Fatalf("path %q in example must not contain parent-directory traversal", trimmed)
	}
}

func looksLikePath(token string) bool {
	return strings.Contains(token, "/") ||
		strings.HasSuffix(token, ".md") ||
		strings.HasSuffix(token, ".yml") ||
		strings.HasSuffix(token, ".yaml") ||
		strings.HasSuffix(token, ".json")
}
