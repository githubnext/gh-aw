package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var bashSingleQuotedArgsCodemodLog = logger.New("cli:codemod_bash_single_quoted_args")

// getBashSingleQuotedArgsCodemod rewrites tools.bash entries that contain
// single-quoted shell arguments into equivalent double-quoted forms so Copilot
// shell allow-tool generation does not truncate them to a prefix.
func getBashSingleQuotedArgsCodemod() Codemod {
	return Codemod{
		ID:           "bash-single-quoted-args-rewrite",
		Name:         "Rewrite single-quoted bash tool args",
		Description:  "Rewrites tools.bash entries like grep -n 'foo' to grep -n \"foo\" when safe, reducing Copilot shell() truncation warnings.",
		IntroducedIn: "0.39.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			bashValue, hasBash := toolsMap["bash"]
			if !hasBash {
				return content, false, nil
			}

			bashCommands, ok := bashValue.([]any)
			if !ok {
				return content, false, nil
			}

			updated := make([]any, len(bashCommands))
			copy(updated, bashCommands)

			changed := false
			var unsafeCommands []string
			for i, cmd := range bashCommands {
				cmdStr, ok := cmd.(string)
				if !ok {
					continue
				}

				rewritten, safe, rewrittenChanged := rewriteSingleQuotedBashArgs(cmdStr)
				if !safe {
					unsafeCommands = append(unsafeCommands, cmdStr)
					continue
				}
				if rewrittenChanged {
					updated[i] = rewritten
					changed = true
				}
			}

			for _, cmd := range unsafeCommands {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
					fmt.Sprintf("tools.bash entry %q contains unmatched single quotes and could not be safely rewritten; left unchanged", cmd)))
			}

			if !changed {
				return content, false, nil
			}

			toolsMap["bash"] = updated
			frontmatter["tools"] = toolsMap

			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter for rewrite: %w", err)
			}

			updatedFrontmatter, err := workflow.MarshalWithFieldOrder(frontmatter, constants.PriorityWorkflowFields)
			if err != nil {
				return content, false, fmt.Errorf("failed to marshal frontmatter after rewrite: %w", err)
			}

			frontmatterStr := strings.TrimSuffix(string(updatedFrontmatter), "\n")
			frontmatterStr = workflow.UnquoteYAMLKey(frontmatterStr, "on")

			var lines []string
			lines = append(lines, "---")
			if frontmatterStr != "" {
				lines = append(lines, strings.Split(frontmatterStr, "\n")...)
			}
			lines = append(lines, "---")
			if result.Markdown != "" {
				lines = append(lines, "")
				lines = append(lines, result.Markdown)
			}

			bashSingleQuotedArgsCodemodLog.Print("Rewrote single-quoted tools.bash arguments to safe double-quoted forms")
			return strings.Join(lines, "\n"), true, nil
		},
	}
}

// rewriteSingleQuotedBashArgs rewrites single-quoted shell segments to
// double-quoted segments with escaping that preserves literal semantics.
// Returns rewritten command, whether rewrite was safe, and whether it changed.
func rewriteSingleQuotedBashArgs(cmd string) (string, bool, bool) {
	if !strings.Contains(cmd, "'") {
		return cmd, true, false
	}

	var b strings.Builder
	b.Grow(len(cmd) + 8)
	changed := false

	for i := 0; i < len(cmd); {
		if cmd[i] != '\'' {
			b.WriteByte(cmd[i])
			i++
			continue
		}

		j := i + 1
		for j < len(cmd) && cmd[j] != '\'' {
			j++
		}
		if j >= len(cmd) {
			return cmd, false, false
		}

		content := cmd[i+1 : j]
		b.WriteByte('"')
		for k := 0; k < len(content); k++ {
			switch content[k] {
			case '\\', '"', '$', '`':
				b.WriteByte('\\')
			}
			b.WriteByte(content[k])
		}
		b.WriteByte('"')

		changed = true
		i = j + 1
	}

	rewritten := b.String()
	if !changed || rewritten == cmd {
		return cmd, true, false
	}
	return rewritten, true, true
}
