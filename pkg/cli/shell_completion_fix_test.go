package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixCompletionScriptBash(t *testing.T) {
	// Sample bash completion script excerpt with original "gh" references
	originalScript := `# bash completion V2 for gh                                   -*- shell-script -*-

__gh_debug()
{
    echo "debug"
}

__gh_init_completion()
{
    echo "init"
}

__gh_handle_completion()
{
    requestComp="${words[0]} __complete ${args[*]}"
    requestComp="GH_ACTIVE_HELP=0 ${words[0]} __completeNoDesc ${args[*]}"
}

_gh()
{
    echo "main completion"
}
`

	fixed := FixCompletionScript(originalScript, "bash")

	// Verify function names are updated
	assert.Contains(t, fixed, "__gh_aw_debug", "Should rename __gh_debug to __gh_aw_debug")
	assert.Contains(t, fixed, "__gh_aw_init_completion", "Should rename __gh_init_completion to __gh_aw_init_completion")
	assert.Contains(t, fixed, "__gh_aw_handle_completion", "Should rename __gh_handle_completion to __gh_aw_handle_completion")
	assert.Contains(t, fixed, "_gh_aw(", "Should rename _gh( to _gh_aw(")

	// Verify completion header is updated
	assert.Contains(t, fixed, "# bash completion V2 for gh aw ", "Should update completion header to 'gh aw'")

	// Verify requestComp uses "gh aw"
	assert.Contains(t, fixed, `requestComp="gh aw __complete`, "Should use 'gh aw' in requestComp for v2")
	assert.Contains(t, fixed, `requestComp="GH_ACTIVE_HELP=0 gh aw __completeNoDesc`, "Should use 'gh aw' in requestComp for v1")

	// Verify old patterns are gone
	assert.NotContains(t, fixed, `${words[0]} __complete`, "Should not contain ${words[0]} in requestComp")
	assert.NotContains(t, fixed, `${words[0]} __completeNoDesc`, "Should not contain ${words[0]} in requestComp")
}

func TestFixCompletionScriptZsh(t *testing.T) {
	// Sample zsh completion script excerpt with original "gh" references
	originalScript := `#compdef gh
compdef _gh gh

# zsh completion for gh                                   -*- shell-script -*-

__gh_debug()
{
    echo "debug"
}

_gh()
{
    requestComp="${words[1]} __complete ${words[2,-1]}"
}
`

	fixed := FixCompletionScript(originalScript, "zsh")

	// Verify function names are updated
	assert.Contains(t, fixed, "__gh_aw_debug", "Should rename __gh_debug to __gh_aw_debug")
	assert.Contains(t, fixed, "_gh_aw(", "Should rename _gh( to _gh_aw(")

	// Verify compdef is updated
	assert.Contains(t, fixed, "compdef _gh_aw gh", "Should register _gh_aw for gh command")
	assert.Contains(t, fixed, "# Register completion for 'gh aw'", "Should include comment about gh aw")

	// Verify completion header is updated
	assert.Contains(t, fixed, "# zsh completion for gh aw ", "Should update completion header to 'gh aw'")

	// Verify requestComp uses "gh aw"
	assert.Contains(t, fixed, `requestComp="gh aw __complete ${words[2,-1]}"`, "Should use 'gh aw' in requestComp")

	// Verify old pattern is gone
	assert.NotContains(t, fixed, `${words[1]} __complete`, "Should not contain ${words[1]} in requestComp")
}

func TestFixCompletionScriptFish(t *testing.T) {
	// Sample fish completion script excerpt
	originalScript := `# fish completion for gh
complete -c gh -n "__fish_use_subcommand" -l help -d "help"
complete -c gh -n "__fish_use_subcommand" -f -a version
`

	fixed := FixCompletionScript(originalScript, "fish")

	// Verify header comment is added
	assert.True(t, strings.HasPrefix(fixed, "# Fish completion for gh aw\n"), "Should add gh aw header comment")

	// Verify completion commands are modified to check for 'aw' subcommand
	assert.Contains(t, fixed, "complete -c gh -n '__fish_seen_subcommand_from aw'", "Should add condition for 'aw' subcommand")

	// Verify original commands are still present but modified
	assert.Contains(t, fixed, "-l help", "Should retain original flags")
	assert.Contains(t, fixed, "version", "Should retain original commands")
}

func TestFixCompletionScriptInvalidShell(t *testing.T) {
	originalScript := "# test script"

	// Should not panic and should return the script unchanged for unknown shell types
	fixed := FixCompletionScript(originalScript, "unknown")

	assert.Equal(t, originalScript, fixed, "Should return unchanged script for unknown shell type")
}

func TestFixCompletionScriptPreservesContent(t *testing.T) {
	// Verify that fix doesn't corrupt the script structure
	originalScript := `# bash completion V2 for gh
__gh_init_completion()
{
    COMPREPLY=()
}

__gh_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

_gh()
{
    local cur prev words cword
    requestComp="${words[0]} __complete ${args[*]}"
}
`

	fixed := FixCompletionScript(originalScript, "bash")

	// Verify structure is maintained
	assert.Contains(t, fixed, "COMPREPLY=()", "Should preserve COMPREPLY initialization")
	assert.Contains(t, fixed, "${BASH_COMP_DEBUG_FILE-}", "Should preserve bash variable syntax")
	assert.Contains(t, fixed, "local cur prev words cword", "Should preserve variable declarations")

	// Verify only targeted replacements were made
	lines := strings.Split(fixed, "\n")
	assert.Greater(t, len(lines), 5, "Should preserve script structure with multiple lines")
}
