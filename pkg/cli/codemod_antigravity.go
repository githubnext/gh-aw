package cli

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

const antigravityManualMigrationMarker = "MANUAL MIGRATION"

var antigravityCodemodLog = logger.New("cli:codemod_antigravity")

var antigravityTokenReplacer = strings.NewReplacer(
	"GEMINI_API_KEY", "ANTIGRAVITY_API_KEY",
	"GEMINI_MODEL", "ANTIGRAVITY_MODEL",
	"GEMINI_API_BASE_URL", "ANTIGRAVITY_API_BASE_URL",
	"GEMINI_CLI_TRUST_WORKSPACE", "ANTIGRAVITY_CLI_TRUST_WORKSPACE",
	"parse_gemini_log", "parse_antigravity_log",
	"convert_gateway_config_gemini", "convert_gateway_config_antigravity",
	"gemini-client-error", "antigravity-client-error",
	".gemini/", ".antigravity/",
	".gemini", ".antigravity",
	"GEMINI.md", "ANTIGRAVITY.md",
)

var antigravityLineReplacements = []struct {
	pattern *regexp.Regexp
	repl    string
}{
	{regexp.MustCompile(`^(\s*engine:\s*)gemini(\s*(?:#.*)?)$`), `${1}antigravity${2}`},
	{regexp.MustCompile(`^(\s*id:\s*)gemini(\s*(?:#.*)?)$`), `${1}antigravity${2}`},
	{regexp.MustCompile(`^(\s*runtime-id:\s*)gemini(\s*(?:#.*)?)$`), `${1}antigravity${2}`},
	{regexp.MustCompile(`("engine"\s*:\s*")gemini(")`), `${1}antigravity${2}`},
	{regexp.MustCompile(`("id"\s*:\s*")gemini(")`), `${1}antigravity${2}`},
	{regexp.MustCompile(`("runtime-id"\s*:\s*")gemini(")`), `${1}antigravity${2}`},
	{regexp.MustCompile(`^(\s*run:\s*)gemini(\s|$)`), `${1}agy${2}`},
	{regexp.MustCompile(`^(\s*)gemini(\s|$)`), `${1}agy${2}`},
}

var antigravityGenericTextReplacements = []struct {
	pattern *regexp.Regexp
	repl    string
}{
	{regexp.MustCompile(`\bGemini CLI\b`), `Antigravity CLI`},
	{regexp.MustCompile(`\bGoogle Gemini\b`), `Antigravity`},
	{regexp.MustCompile(`\bGemini\b`), `Antigravity`},
	{regexp.MustCompile(`\bgemini\b`), `antigravity`},
}

var antigravityLegacyModelPattern = regexp.MustCompile(`(^|[\s"'` + "`" + `])gemini-[A-Za-z0-9._-]+`)
var antigravityModelFieldPattern = regexp.MustCompile(`(^|\s)model:\s*(gemini|gemini-[A-Za-z0-9._-]+)(\s*(#.*)?)?$`)

// getGeminiToAntigravityCodemod migrates legacy Gemini workflow references to Antigravity.
func getGeminiToAntigravityCodemod() Codemod {
	return Codemod{
		ID:           "gemini-to-antigravity",
		Name:         "Migrate Gemini engine to Antigravity",
		Description:  "Rewrites legacy Gemini engine identifiers, credentials, CLI invocations, and workflow text to their Antigravity equivalents",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !strings.Contains(strings.ToLower(content), "gemini") {
				return content, false, nil
			}

			hadTrailingNewline := strings.HasSuffix(content, "\n")
			lines := strings.Split(content, "\n")
			result := make([]string, 0, len(lines)+1)
			modified := false

			for _, line := range lines {
				if strings.Contains(line, antigravityManualMigrationMarker) {
					result = append(result, line)
					continue
				}

				manualMigration := requiresManualAntigravityModelMigration(line)
				if manualMigration && !lastLineHasAntigravityManualMarker(result) {
					result = append(result, buildAntigravityManualMigrationComment(line))
					modified = true
				}

				rewritten := rewriteAntigravityLine(line, manualMigration)
				if rewritten != line {
					modified = true
				}
				result = append(result, rewritten)
			}

			if !modified {
				return content, false, nil
			}

			updated := strings.Join(result, "\n")
			if hadTrailingNewline && !strings.HasSuffix(updated, "\n") {
				updated += "\n"
			}
			if !hadTrailingNewline && strings.HasSuffix(updated, "\n") {
				updated = strings.TrimSuffix(updated, "\n")
			}

			antigravityCodemodLog.Print("Applied Gemini to Antigravity migration codemod")
			return updated, true, nil
		},
	}
}

func rewriteAntigravityLine(line string, skipGenericGemini bool) string {
	rewritten := antigravityTokenReplacer.Replace(line)

	for _, replacement := range antigravityLineReplacements {
		rewritten = replacement.pattern.ReplaceAllString(rewritten, replacement.repl)
	}
	if skipGenericGemini {
		return rewritten
	}
	for _, replacement := range antigravityGenericTextReplacements {
		rewritten = replacement.pattern.ReplaceAllString(rewritten, replacement.repl)
	}

	return rewritten
}

func requiresManualAntigravityModelMigration(line string) bool {
	if strings.Contains(line, antigravityManualMigrationMarker) {
		return false
	}
	return antigravityLegacyModelPattern.MatchString(line) || antigravityModelFieldPattern.MatchString(strings.TrimSpace(line))
}

func lastLineHasAntigravityManualMarker(lines []string) bool {
	if len(lines) == 0 {
		return false
	}
	return strings.Contains(lines[len(lines)-1], antigravityManualMigrationMarker)
}

func buildAntigravityManualMigrationComment(line string) string {
	indent := getIndentation(line)
	return indent + "# " + antigravityManualMigrationMarker + ": review former Gemini model mapping for Antigravity."
}
