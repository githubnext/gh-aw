package workflow

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// Match command positions rather than arbitrary mentions in step names or messages.
// This is a static check for recognizable CLI usage, not a shell interpreter.
const engineShellCommand = `(?:[^ \t\n;"'|&()]*[/\\])?(?:bash|sh)[ \t]+(?:(?:--(?:login|noprofile|norc|posix)|[-+][abefhkmnptuvxBCEHPTilsrD]+)[ \t]+)*-[abefhkmnptuvxBCEHPTilsrD]*c[abefhkmnptuvxBCEHPTilsrD]*[ \t]+(?:--[ \t]+)?`
const engineCommandStart = `(?:^|[;&|]|\$\(|\x60)[ \t]*(?:\([ \t]*|(?:sudo|env|time|timeout|npx)[ \t]+(?:--?[A-Za-z-]+(?:=[^ \t\n]+)?[ \t]+)*|(?:exec|command|if|elif|while|until|then|do|!)[ \t]+(?:--[ \t]+)?|(?:npm|pnpm|yarn|bun)[ \t]+(?:exec|dlx|x)[ \t]+(?:--[ \t]+)?|` + engineShellCommand + `["']?|[A-Za-z_][A-Za-z0-9_]*=[^ \t\n]+[ \t]+|[0-9]+[smh]?[ \t]+)*`

var (
	agenticEngineCommandPattern   = regexp.MustCompile(`(?mi)` + engineCommandStart + `["']?(?:[^ \t\n;"'|&()]*[/\\])?(?:copilot|claude|codex|gemini|agy|pi)(?:\.exe)?["']?(?:[ \t\n;|&()]|$)`)
	agenticEnginePackagePattern   = regexp.MustCompile(`(?mi)` + engineCommandStart + `(?:npx|(?:npm|pnpm|yarn|bun)[ \t]+(?:global[ \t]+|--?[A-Za-z-]+[ \t]+)*(?:install|i|add|exec|dlx|x))[ \t]+[^\n;&|]*@(?:github/copilot|anthropic-ai/claude-code|openai/codex|google/gemini-cli|earendil-works/pi-coding-agent)(?:@[^\s;&|]+)?(?:[\s;"'|&]|$)`)
	agenticEngineGHPattern        = regexp.MustCompile(`(?mi)` + engineCommandStart + `gh[ \t]+(?:copilot(?:[ \t\n;|&]|$)|extension[ \t]+install[ \t]+(?:github/)?gh-copilot(?:[ \t\n;|&]|$))`)
	agenticEngineInstallerPattern = regexp.MustCompile(`(?mi)` + engineCommandStart +
		`(?:curl|wget|brew)[ \t]+[^\n;&|]*(?:github\.com/github/copilot-cli/(?:releases|raw)|raw\.githubusercontent\.com/github/copilot-cli/|claude\.ai/install\.(?:sh|ps1)|(?:install[ \t]+(?:--cask[ \t]+)?)(?:copilot-cli|claude-code|codex)(?:[\s;|&]|$))`)
	engineDataHeredocPattern = regexp.MustCompile(`(?mi)` + engineCommandStart + `(?:cat|tee)[ \t]+`)
	engineShellScriptPattern = regexp.MustCompile(`(?mi)` + engineCommandStart + engineShellCommand + `('[^']*'|"(?:\\.|[^"\\])*")`)
)

func engineDataHeredocHeader(header string) string {
	// Keep ambiguous bodies visible when several heredocs share a header.
	if strings.Count(header, "<<") != 1 {
		return ""
	}
	for _, location := range engineDataHeredocPattern.FindAllStringIndex(header, -1) {
		command := header[location[1]:] //nolint:uncheckedsliceindex // FindAllStringIndex returns pairs of valid offsets.
		if strings.Contains(command, "|") {
			continue
		}
		separators := strings.NewReplacer(">&", "__", "<&", "__", "&>", "__").Replace(command)
		if end := strings.IndexAny(separators, ";&"); end >= 0 {
			command = command[:end]
		}
		if strings.Contains(command, "<<") {
			return command
		}
	}
	return ""
}

// Preserve substitutions in unquoted data heredocs, but not literal example commands.
// Heredocs consumed by an interpreter or a pipeline remain subject to detection.
func maskEngineDataHeredocs(run string) string {
	lines := strings.Split(run, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i] //nolint:uncheckedsliceindex // The nonnegative cursor is loop-bounded.
		header := engineDataHeredocHeader(maskEngineNonExecutableText(line))
		if delim, ok := detectHeredocDelimiter(header); ok {
			_, after, _ := strings.Cut(header, "<<")
			after = strings.TrimSpace(strings.TrimPrefix(after, "-"))
			quoted := strings.HasPrefix(after, "'") || strings.HasPrefix(after, `"`)
			start, end := i+1, i+1
			for end < len(lines) && strings.TrimSpace(lines[end]) != delim { //nolint:uncheckedsliceindex // end starts nonnegative and is explicitly bounded.
				end++
			}
			body := strings.Join(lines[start:end], "\n")
			for j := start; j < end; j++ {
				lines[j] = "" //nolint:uncheckedsliceindex // start and end are bounded by len(lines).
			}
			if !quoted && start < end {
				lines[start] = "_" + maskEngineText(body, '"', true) //nolint:uncheckedsliceindex // start < end <= len(lines).
			}
			i = end
		}
	}
	return strings.Join(lines, "\n")
}

type engineSubstitution struct {
	quote, closing byte
	arithmetic     bool
}

func updateEngineSubstitution(ch, quote byte, substitutions []engineSubstitution) (byte, []engineSubstitution) {
	if quote == '\'' {
		return quote, substitutions
	}
	if quote == 0 && len(substitutions) > 0 {
		last := substitutions[len(substitutions)-1] //nolint:uncheckedsliceindex // The stack is nonempty.
		if ch == last.closing {
			return last.quote, substitutions[:len(substitutions)-1]
		}
		if ch == '(' {
			return 0, append(substitutions, engineSubstitution{closing: ')', arithmetic: last.arithmetic})
		}
	}
	if ch == '`' {
		return 0, append(substitutions, engineSubstitution{quote: quote, closing: '`'})
	}
	return quote, substitutions
}

func startEngineSubstitution(run string, masked []byte, i int, quote byte, substitutions []engineSubstitution) ([]engineSubstitution, int) {
	if i+2 < len(run) && run[i+2] == '(' { //nolint:uncheckedsliceindex // The caller provides a nonnegative cursor and i+2 is bounded.
		substitutions = append(substitutions,
			engineSubstitution{quote: quote, closing: ')', arithmetic: true},
			engineSubstitution{closing: ')', arithmetic: true})
		copy(masked[i:i+3], "___")
		return substitutions, i + 2
	}
	return append(substitutions, engineSubstitution{quote: quote, closing: ')'}), i + 1
}

// Mask separators in quoted text and comments so they cannot create command
// positions. Command substitutions remain executable even inside double quotes.
func maskEngineNonExecutableText(run string) string {
	return maskEngineText(run, 0, false)
}

func maskEngineText(run string, quote byte, heredoc bool) string {
	masked := []byte(run)
	comment := false
	var substitutions []engineSubstitution
	for i := 0; i < len(run); i++ {
		ch := run[i] //nolint:uncheckedsliceindex // The nonnegative cursor is bounded by the loop condition.
		if comment {
			if ch == '\n' {
				comment = false
			} else {
				masked[i] = '_' //nolint:uncheckedsliceindex // masked has the same length as run and i is loop-bounded.
			}
			continue
		}
		if ch == '\\' && quote != '\'' && i+1 < len(run) {
			i++
			if strings.ContainsRune(";&|\n`$'\"", rune(run[i])) { //nolint:uncheckedsliceindex // The branch checks i+1 before incrementing i.
				masked[i] = '_' //nolint:uncheckedsliceindex // The branch checks i+1 before incrementing i.
			}
			continue
		}
		if quote == 0 && ch == '#' && (i == 0 || strings.ContainsRune(" \t\n;&|()", rune(run[i-1]))) { //nolint:uncheckedsliceindex // Short-circuiting guards i-1 and i is loop-bounded.
			comment = true
			masked[i] = '_' //nolint:uncheckedsliceindex // masked has the same length as run and i is loop-bounded.
			continue
		}
		if quote != '\'' && ch == '$' && i+1 < len(run) && run[i+1] == '(' { //nolint:uncheckedsliceindex // i+1 is explicitly bounded.
			substitutions, i = startEngineSubstitution(run, masked, i, quote, substitutions)
			quote = 0
			continue
		}
		if len(substitutions) > 0 && substitutions[len(substitutions)-1].arithmetic && ch != '`' { //nolint:uncheckedsliceindex // The substitution stack is nonempty.
			masked[i] = '_' //nolint:uncheckedsliceindex // masked has the same length as run and i is loop-bounded.
		}
		quote, substitutions = updateEngineSubstitution(ch, quote, substitutions)
		if ch == '`' && quote != '\'' {
			continue
		}
		if heredoc && len(substitutions) == 0 && strings.ContainsRune("'\"#", rune(ch)) {
			masked[i] = '_' //nolint:uncheckedsliceindex // masked has the same length as run and i is loop-bounded.
			continue
		}
		if (ch == '\'' || ch == '"') && (!heredoc || len(substitutions) > 0) {
			switch quote {
			case 0:
				quote = ch
			case ch:
				quote = 0
			}
		}
		if quote != 0 && strings.ContainsRune(";&|\n`", rune(ch)) {
			masked[i] = '_' //nolint:uncheckedsliceindex // masked has the same length as run and i is loop-bounded.
		}
	}
	return string(masked)
}

func hasAgenticEngineReference(content string) bool {
	return strings.Contains(content, "copilot") || strings.Contains(content, "claude") ||
		strings.Contains(content, "codex") || strings.Contains(content, "gemini") ||
		strings.Contains(content, "agy") || strings.Contains(content, "pi")
}

func stepInvokesAgenticEngine(step map[string]any) bool {
	if uses, ok := step["uses"].(string); ok {
		action, _, _ := strings.Cut(strings.ToLower(uses), "@")
		switch action {
		case "anthropics/claude-code-action", "openai/codex-action", "github/copilot-cli-action", "google-github-actions/run-gemini-cli":
			return true
		}
	}
	run, ok := step["run"].(string)
	if !ok {
		return false
	}
	// Join shell continuations before inspecting command positions.
	run = strings.ReplaceAll(run, "\\\r\n", "")
	run = strings.ReplaceAll(run, "\\\n", "")
	run = maskEngineDataHeredocs(run)
	masked := maskEngineNonExecutableText(run)
	for _, match := range engineShellScriptPattern.FindAllStringSubmatchIndex(masked, -1) {
		if len(match) != 4 {
			continue
		}
		start, end := match[2], match[3] //nolint:uncheckedsliceindex // Exactly four offsets include the quoted script capture.
		script := run[start+1 : end-1]
		if strings.HasPrefix(run[start:end], `"`) {
			script = strings.NewReplacer(`\"`, `"`, `\\`, `\`, `\$`, `$`, "\\`", "`").Replace(script)
		}
		if stepInvokesAgenticEngine(map[string]any{"run": script}) {
			return true
		}
	}
	run = masked
	return agenticEngineCommandPattern.MatchString(run) ||
		agenticEnginePackagePattern.MatchString(run) ||
		agenticEngineGHPattern.MatchString(run) ||
		agenticEngineInstallerPattern.MatchString(run)
}

// validateAgenticEngineSteps inspects the final jobs so imported steps, safe-output
// steps, and built-in job augmentations are covered as well as custom jobs.
func (c *Compiler) validateAgenticEngineSteps(workflow map[string]any, strict bool, markdownPath string) error {
	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		return nil
	}
	jobNames := make([]string, 0, len(jobs))
	for name := range jobs {
		jobNames = append(jobNames, name)
	}
	slices.Sort(jobNames)
	for _, jobName := range jobNames {
		switch jobName {
		case string(constants.AgentJobName), string(constants.DetectionJobName), string(constants.EvalsJobName):
			continue
		}
		job, ok := jobs[jobName].(map[string]any)
		if !ok {
			continue
		}
		steps, ok := job["steps"].([]any)
		if !ok {
			continue
		}
		for i, value := range steps {
			step, ok := value.(map[string]any)
			if !ok || !stepInvokesAgenticEngine(step) {
				continue
			}
			msg := fmt.Sprintf("agentic engine installation or invocation in job %q, step %d runs outside the agent sandbox; configure the engine with 'engine:' and run it in the agent job instead", jobName, i+1)
			if strict {
				return fmt.Errorf("strict mode: %s", msg)
			}
			fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", msg))
			c.IncrementWarningCount()
		}
	}
	return nil
}
