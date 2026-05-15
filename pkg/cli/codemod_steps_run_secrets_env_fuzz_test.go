//go:build !integration

package cli

import (
	"strings"
	"testing"
)

func FuzzStepsRunSecretsToEnvCodemod(f *testing.F) {
	f.Add(uint8(0), "RUNTIME_TOKEN", "RUNTIME_TOKEN", true, false, true, true, false)
	f.Add(uint8(1), "abc123", "lower_case", false, false, true, false, true)
	f.Add(uint8(2), "TOKEN_2", "TOKEN_2", true, true, false, true, true)
	f.Add(uint8(3), "A", "B", false, false, false, false, false)

	f.Fuzz(func(t *testing.T, sectionSelector uint8, secretNameRaw, envNameRaw string, includeSecret, includeComplexSecret, includeEnvExpression, includeGitHubToken, preseedBindings bool) {
		secretName := sanitizeHoistName(secretNameRaw)
		envName := sanitizeHoistName(envNameRaw)

		section := []string{"pre-steps", "steps", "post-steps", "pre-agent-steps"}[int(sectionSelector)%4]
		run, expectedVars := buildHoistFuzzRun(includeSecret, includeComplexSecret, includeEnvExpression, includeGitHubToken, secretName, envName)

		content := buildHoistFuzzContent(section, run, expectedVars, secretName, envName, preseedBindings)
		frontmatter := map[string]any{
			"on":       "push",
			section:    []any{map[string]any{"run": run}},
			"workflow": "fuzz",
		}

		result, applied, err := getStepsRunSecretsToEnvCodemod().Apply(content, frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(expectedVars) == 0 {
			if applied {
				t.Fatalf("expected no mutation for run=%q", run)
			}
			if result != content {
				t.Fatalf("content changed unexpectedly for no-op input")
			}
			return
		}

		if !applied {
			t.Fatalf("expected mutation for run=%q", run)
		}
		runLine := extractFuzzRunLine(result)
		if strings.Contains(runLine, "${{ secrets.") || strings.Contains(runLine, "${{ env.") || strings.Contains(runLine, "${{ github.token") {
			t.Fatalf("run line still contains expression interpolation: %q", runLine)
		}
		for _, variable := range expectedVars {
			if !strings.Contains(runLine, "$"+variable) {
				t.Fatalf("run line missing rewritten variable %q: %q", variable, runLine)
			}
			if strings.HasSuffix(variable, "_") {
				if countEnvBindingKeyPrefix(result, variable) != 1 {
					t.Fatalf("expected exactly one env binding with prefix %s", variable)
				}
				continue
			}
			if countEnvBindingKey(result, variable) != 1 {
				t.Fatalf("expected exactly one env binding for %s", variable)
			}
		}
	})
}

func buildHoistFuzzRun(includeSecret, includeComplexSecret, includeEnvExpr, includeGitHubToken bool, secretName, envName string) (string, []string) {
	parts := make([]string, 0, 3)
	expected := make([]string, 0, 4)

	if includeSecret {
		parts = append(parts, "${{ secrets."+secretName+" }}")
		expected = append(expected, secretName)
	}
	if includeComplexSecret {
		parts = append(parts, "${{ secrets."+secretName+" || 'fallback' }}")
		expected = append(expected, "GH_AW_SECRET_"+secretName+"_")
	}
	if includeEnvExpr {
		parts = append(parts, "${{ env."+envName+" }}")
		expected = append(expected, "GH_AW_ENV_"+envName)
	}
	if includeGitHubToken {
		parts = append(parts, "${{ github.token }}")
		expected = append(expected, "GH_AW_GITHUB_TOKEN")
	}
	if len(parts) == 0 {
		return `echo "ok"`, nil
	}
	duplicated := append(append([]string(nil), parts...), parts...)
	return `echo "` + strings.Join(duplicated, " ") + `"`, expected
}

func buildHoistFuzzContent(section, run string, expectedVars []string, secretName, envName string, preseedBindings bool) string {
	lines := []string{
		"---",
		"on: push",
		section + ":",
		"  - name: fuzz",
	}

	if preseedBindings && len(expectedVars) > 0 {
		lines = append(lines, "    env:")
		for _, variable := range expectedVars {
			switch variable {
			case "GH_AW_GITHUB_TOKEN":
				lines = append(lines, "      "+variable+": ${{ github.token }}")
			case "GH_AW_ENV_" + envName:
				lines = append(lines, "      "+variable+": ${{ env."+envName+" }}")
			case secretName:
				lines = append(lines, "      "+variable+": ${{ secrets."+secretName+" }}")
			}
		}
	}

	lines = append(lines, "    run: "+run, "---")
	return strings.Join(lines, "\n") + "\n"
}

func extractFuzzRunLine(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "run: ") {
			return trimmed
		}
	}
	return ""
}

func countEnvBindingKey(content, key string) int {
	count := 0
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), key+": ") {
			count++
		}
	}
	return count
}

// countEnvBindingKeyPrefix counts env binding keys by prefix for hashed names
// where only the deterministic prefix is known in advance.
func countEnvBindingKeyPrefix(content, keyPrefix string) int {
	count := 0
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), keyPrefix) {
			count++
		}
	}
	return count
}

// sanitizeHoistName converts arbitrary fuzz input into a valid env-var style token
// ([A-Z0-9_], max 20 chars) and ensures the name does not start with a digit.
func sanitizeHoistName(raw string) string {
	if raw == "" {
		return "TOKEN"
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		}
		if b.Len() >= 20 {
			break
		}
	}
	s := b.String()
	if s == "" {
		return "TOKEN"
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "T_" + s
	}
	return s
}
