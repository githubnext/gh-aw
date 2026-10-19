package shellpattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstUnsafeExpansion(t *testing.T) {
	for _, tt := range []struct {
		name, script, category, match string
	}{
		{name: "empty"},
		{name: "ordinary text", script: "cat /tmp/value"},
		{name: "variable", script: `echo "$VALUE" "${VALUE}"`},
		{name: "default", script: `echo "${VALUE:-default}"`},
		{name: "arithmetic", script: `echo $((1 + 2))`},
		{name: "actions expression", script: `echo "${{ github.repository }}"`},
		{"command", `VALUE=$(cat /tmp/value)`, "command substitution", "$("},
		{"indirect", `echo "${!VALUE}"`, "indirect expansion", "${!V"},
		{"transform", `echo "${VALUE@P}"`, "parameter transformation", "${VALUE@P}"},
		{"upper transform", `echo "${VALUE@U}"`, "parameter transformation", "${VALUE@U}"},
		{"backtick", "VALUE=`cat /tmp/value`", "backtick command substitution", "`cat /tmp/value`"},
		{"after arithmetic", "echo $((1 + 2))\nVALUE=$(cat /tmp/value)", "command substitution", "$("},
		{"nested in arithmetic", "echo $((1 + $(cat /tmp/value)))", "command substitution", "$("},
		{"lexical quotes", `echo '$(not-executed)'`, "command substitution", "$("},
		{"lexical comment", "# $(not-executed)", "command substitution", "$("},
	} {
		t.Run(tt.name, func(t *testing.T) {
			expansion, found := FirstUnsafeExpansion(tt.script)
			assert.Equal(t, tt.category != "", found)
			if found {
				assert.Contains(t, expansion.Description, tt.category)
				assert.Equal(t, tt.match, tt.script[expansion.Start:expansion.End])
			} else {
				assert.Equal(t, Expansion{}, expansion)
			}
		})
	}
}
