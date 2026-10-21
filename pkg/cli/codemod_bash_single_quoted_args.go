package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
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
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				bashValue, _ := f.Lookup("tools", "bash")
				bashCommands, ok := bashValue.([]any)
				if !ok {
					return nil
				}

				updated := make([]any, len(bashCommands))
				copy(updated, bashCommands)

				changed := false
				var unsafeCommands []string
				for i, cmd := range updated {
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
						fmt.Sprintf("tools.bash entry %q contains an unclosed single-quoted segment and could not be safely rewritten; left unchanged", cmd)))
				}

				if !changed {
					return nil
				}

				bashSingleQuotedArgsCodemodLog.Print("Rewrote single-quoted tools.bash arguments to safe double-quoted forms")
				return f.Set([]string{"tools", "bash"}, updated)
			})
		},
	}
}

// rewriteSingleQuotedBashArgs rewrites single-quoted shell segments to
// double-quoted segments with escaping that preserves literal semantics.
// Returns rewritten command, whether rewrite was safe, and whether it changed.
//
//nolint:largefunc // Keep the existing shell quote state machine together; its parser behavior is unchanged by the AST migration.
func rewriteSingleQuotedBashArgs(cmd string) (string, bool, bool) {
	if !strings.Contains(cmd, "'") {
		return cmd, true, false
	}

	var b strings.Builder
	b.Grow(len(cmd) + 8)
	changed := false
	inDoubleQuotes := false
	escaped := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i] //nolint:uncheckedsliceindex // i starts at zero and advances only to a matched quote within cmd; the loop checks the upper bound.
		if inDoubleQuotes {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuotes = false
			}
			continue
		}

		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			b.WriteByte(ch)
			escaped = true
			continue
		case '"':
			b.WriteByte(ch)
			inDoubleQuotes = true
			continue
		case '\'':
			j := i + 1
			for j < len(cmd) && cmd[j] != '\'' { //nolint:uncheckedsliceindex // j starts after the in-bounds i, and the left condition checks the upper bound.
				j++
			}
			if j >= len(cmd) {
				return cmd, false, false
			}

			content := cmd[i+1 : j]
			b.WriteByte('"')
			for _, contentCh := range []byte(content) {
				switch contentCh {
				case '\\', '"', '$', '`':
					b.WriteByte('\\')
				}
				b.WriteByte(contentCh)
			}
			b.WriteByte('"')

			changed = true
			i = j
			continue
		default:
			b.WriteByte(ch)
		}
	}

	rewritten := b.String()
	if !changed || rewritten == cmd {
		return cmd, true, false
	}
	return rewritten, true, true
}
