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
const engineCommandStart = `(?:^|[;&|]|\$\(|\x60)[ \t]*(?:\([ \t]*)*(?:(?:sudo|env|time|timeout|npx)[ \t]+(?:--?[A-Za-z-]+(?:=[^ \t\n]+)?[ \t]+)*|(?:exec|command|then|do)[ \t]+(?:--[ \t]+)?|(?:npm|pnpm|yarn|bun)[ \t]+(?:exec|dlx|x)[ \t]+(?:--[ \t]+)?|(?:bash|sh)[ \t]+-c[ \t]+["']?|[A-Za-z_][A-Za-z0-9_]*=[^ \t\n]+[ \t]+|[0-9]+[smh]?[ \t]+)*`

var (
	agenticEngineCommandPattern   = regexp.MustCompile(`(?mi)` + engineCommandStart + `["']?(?:[^ \t\n;"'|&()]*[/\\])?(?:copilot|claude|codex)(?:\.exe)?["']?(?:[ \t\n;|&()]|$)`)
	agenticEnginePackagePattern   = regexp.MustCompile(`(?mi)` + engineCommandStart + `(?:npx|(?:npm|pnpm|yarn|bun)[ \t]+(?:global[ \t]+|--?[A-Za-z-]+[ \t]+)*(?:install|i|add|exec|dlx|x))[ \t]+[^\n;&|]*@(?:github/copilot|anthropic-ai/claude-code|openai/codex)(?:@[^\s;&|]+)?(?:[\s;"'|&]|$)`)
	agenticEngineGHPattern        = regexp.MustCompile(`(?mi)` + engineCommandStart + `gh[ \t]+(?:copilot(?:[ \t\n;|&]|$)|extension[ \t]+install[ \t]+(?:github/)?gh-copilot(?:[ \t\n;|&]|$))`)
	agenticEngineInstallerPattern = regexp.MustCompile(`(?mi)` + engineCommandStart +
		`(?:curl|wget|brew)[ \t]+[^\n;&|]*(?:github\.com/github/copilot-cli/(?:releases|raw)|raw\.githubusercontent\.com/github/copilot-cli/|claude\.ai/install\.(?:sh|ps1)|(?:install[ \t]+(?:--cask[ \t]+)?)(?:copilot-cli|claude-code|codex)(?:[\s;|&]|$))`)
)

func hasAgenticEngineReference(content string) bool {
	return strings.Contains(content, "copilot") || strings.Contains(content, "claude") || strings.Contains(content, "codex")
}

func stepInvokesAgenticEngine(step map[string]any) bool {
	if uses, ok := step["uses"].(string); ok {
		action, _, _ := strings.Cut(strings.ToLower(uses), "@")
		switch action {
		case "anthropics/claude-code-action", "openai/codex-action", "github/copilot-cli-action":
			return true
		}
	}
	run, ok := step["run"].(string)
	if !ok {
		return false
	}
	// Join shell continuations and ignore full-line comments before inspecting commands.
	run = strings.ReplaceAll(run, "\\\r\n", "")
	run = strings.ReplaceAll(run, "\\\n", "")
	lines := strings.Split(run, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			lines[i] = ""
		}
	}
	run = strings.Join(lines, "\n")
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
