package lint

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"mvdan.cc/sh/v3/syntax"
)

// GitHubCLISyntaxRule reports observed CLI contract mistakes without executing gh
// or assuming that every command generated later by an agent is observable.
func GitHubCLISyntaxRule() Rule {
	return Rule{
		ID: "github-cli-syntax", Description: "Warn about unsupported gh pager flags and PR JSON fields",
		Severity: Warning, Targets: ActionsAST | AWAST,
		Check: func(ctx context.Context, input Input) ([]Diagnostic, error) {
			return checkGitHubCLIContracts(ctx, input, false)
		},
	}
}

func GitHubProxyDiffRule() Rule {
	return Rule{
		ID: "github-proxy-diff", Description: "Warn when an agent prompt relies on raw gh pr diff through gh-proxy",
		Severity: Warning, Targets: AWAST,
		Check: func(ctx context.Context, input Input) ([]Diagnostic, error) {
			return checkGitHubCLIContracts(ctx, input, true)
		},
	}
}

type githubCLIScript struct {
	path, text string
	line       int
}

func checkGitHubCLIContracts(ctx context.Context, input Input, proxyOnly bool) ([]Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if proxyOnly {
		if input.Document == nil {
			return nil, nil
		}
		tools, toolsOK := input.Document.Frontmatter["tools"].(map[string]any)
		github, githubOK := tools["github"].(map[string]any)
		if !toolsOK || !githubOK || github["mode"] != "gh-proxy" {
			return nil, nil
		}
	}
	scripts, err := githubCLIExecutionScripts(ctx, input, proxyOnly)
	if err != nil {
		return nil, err
	}
	if input.Document != nil {
		scripts = append(scripts, githubCLIPromptScripts(input.Document.Markdown)...)
	}
	var diagnostics []Diagnostic
	for _, script := range scripts {
		if err := ctx.Err(); err != nil {
			return diagnostics, err
		}
		if !strings.Contains(script.text, "gh") {
			continue
		}
		tree, err := syntax.NewParser().Parse(strings.NewReader(script.text), "")
		if err != nil {
			if !proxyOnly {
				diagnostics = append(diagnostics, Diagnostic{Path: script.path, Line: script.line, Message: "Incomplete GitHub CLI syntax analysis: shell text could not be parsed; check the script with shellcheck"})
			}
			continue
		}
		syntax.Walk(tree, func(node syntax.Node) bool {
			if ctx.Err() != nil {
				return false
			}
			call, ok := node.(*syntax.CallExpr)
			if !ok {
				return true
			}
			if len(call.Args) > 0 {
				if githubCLIWord(call.Args[0]) != "gh" {
					return true
				}
				args := make([]string, 0, len(call.Args)-1)
				for _, word := range call.Args[1:] {
					args = append(args, githubCLIWord(word))
				}
				for _, message := range githubCLIContractMessages(args, proxyOnly) {
					diagnostics = append(diagnostics, Diagnostic{Path: script.path, Line: script.line + int(call.Pos().Line()) - 1, Message: message})
				}
			}
			return true
		})
	}
	return diagnostics, ctx.Err()
}

func githubCLIContractMessages(args []string, proxyOnly bool) []string {
	if end := slices.Index(args, "--"); end >= 0 {
		args = args[:end]
	}
	var messages []string
	if !proxyOnly && slices.Contains(args, "--no-pager") {
		messages = append(messages, "gh does not support --no-pager; remove the flag and configure GH_PAGER=cat in the step environment")
	}
	if len(args) < 2 {
		return messages
	}
	command := args[:2]
	if proxyOnly {
		if slices.Equal(command, []string{"pr", "diff"}) && !slices.Contains(args, "--name-only") {
			return []string{"Agent prompt requires a raw PR diff through gh-proxy, which may return JSON metadata instead; pre-fetch a unified diff in a host step and pass its file path to the delegate"}
		}
		return nil
	}
	if !slices.Equal(command, []string{"pr", "view"}) && !slices.Equal(command, []string{"pr", "list"}) {
		return messages
	}
	jsonValueNext := false
	for _, arg := range args {
		fields, jsonFlag := strings.CutPrefix(arg, "--json=")
		if jsonValueNext {
			fields, jsonFlag = arg, true
		}
		jsonValueNext = arg == "--json"
		if jsonFlag && slices.Contains(strings.Split(fields, ","), "authorAssociation") {
			messages = append(messages, "gh pr --json does not expose authorAssociation; use a host-side gh api PR read for the REST author_association field")
			break
		}
	}
	return messages
}

func githubCLIWord(word *syntax.Word) string {
	var text strings.Builder
	for _, part := range word.Parts {
		switch part := part.(type) {
		case *syntax.Lit:
			text.WriteString(part.Value)
		case *syntax.SglQuoted:
			text.WriteString(part.Value)
		case *syntax.DblQuoted:
			for _, nested := range part.Parts {
				literal, ok := nested.(*syntax.Lit)
				if !ok {
					return ""
				}
				text.WriteString(literal.Value)
			}
		default:
			return ""
		}
	}
	return text.String()
}

func githubCLIExecutionScripts(ctx context.Context, input Input, proxyOnly bool) ([]githubCLIScript, error) {
	if proxyOnly {
		// Authored pre-agent steps use the host client, not the agent's proxy.
		return nil, nil
	}
	var scripts []githubCLIScript
	add := func(fields map[string]any, path string) {
		shell, shellOK := fields["shell"].(string)
		if fields["shell"] != nil && !shellOK {
			return
		}
		if shell != "" && !strings.HasPrefix(shell, "bash") && !strings.HasPrefix(shell, "sh") {
			return
		}
		if run, ok := fields["run"].(string); ok {
			scripts = append(scripts, githubCLIScript{path: path + ".run", text: run, line: 1})
		}
	}
	if input.Workflow != nil {
		for _, job := range input.Workflow.Jobs {
			for index, step := range job.Steps {
				if err := ctx.Err(); err != nil {
					return scripts, err
				}
				fields, err := actions.LowerStep(step, job.ID, input.Workflow)
				if err != nil {
					return scripts, errors.New("GitHub CLI syntax analysis requires valid Actions steps; correct the step before linting")
				}
				add(fields, fmt.Sprintf("jobs.%s.steps[%d]", job.ID, index))
			}
		}
	} else if input.Document != nil {
		blocks, err := aw.ExecutionBlocks(ctx, input.Document)
		if err != nil {
			return scripts, err
		}
		for _, block := range blocks {
			for _, step := range block.Steps {
				add(step.Fields, step.Path)
			}
		}
	}
	return scripts, nil
}

var githubCLIRunInstruction = regexp.MustCompile(`(?i)^(?:[-*+]\s+|[0-9]+[.)]\s+)?(?:please\s+|you must\s+|must\s+)?run\s+`)
var githubCLINegativeInstruction = regexp.MustCompile(`(?i)\b(?:not|never|avoid|examples?|forbidden|without)\b`)
var githubCLIComment = regexp.MustCompile(`(?s)<!--.*?-->`)

// Only positive Run instructions are executable examples. Quoted prose,
// negative examples, arbitrary code fences and dynamically chosen tools defer.
func githubCLIPromptScripts(markdown string) []githubCLIScript {
	markdown = githubCLIComment.ReplaceAllStringFunc(markdown, func(comment string) string {
		return strings.Repeat("\n", strings.Count(comment, "\n"))
	})
	var scripts []githubCLIScript
	var fence string
	var lines []string
	pending, active, start := false, false, 0
	for index, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			if fence != "" {
				if strings.HasPrefix(line, fence) {
					if active {
						scripts = append(scripts, githubCLIScript{path: "prompt", text: strings.Join(lines, "\n"), line: start})
					}
					fence, lines, active = "", nil, false
				}
			} else {
				fence = line[:3]
				language := strings.TrimSpace(line[3:])
				active = pending && slices.Contains([]string{"bash", "sh", "shell"}, language)
				start = index + 2
			}
			pending = false
			continue
		}
		if fence != "" {
			if active {
				lines = append(lines, raw)
			}
			continue
		}
		if line == "" {
			continue
		}
		pending = githubCLIRunInstruction.MatchString(line) && !githubCLINegativeInstruction.MatchString(line)
		if !pending {
			continue
		}
		command := githubCLIRunInstruction.ReplaceAllString(line, "")
		if strings.HasPrefix(command, "`") {
			if end := strings.IndexByte(command[1:], '`'); end >= 0 {
				scripts = append(scripts, githubCLIScript{path: "prompt", text: command[1 : end+1], line: index + 1})
			}
		} else if strings.HasPrefix(command, "gh ") {
			scripts = append(scripts, githubCLIScript{path: "prompt", text: strings.TrimSuffix(command, "."), line: index + 1})
		}
	}
	return scripts
}
