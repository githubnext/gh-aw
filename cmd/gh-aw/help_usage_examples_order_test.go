//go:build !integration

package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandsUseExampleFieldForHelpOrdering(t *testing.T) {
	testCases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "new", cmd: newCmd},
		{name: "remove", cmd: removeCmd},
		{name: "enable", cmd: enableCmd},
		{name: "disable", cmd: disableCmd},
		{name: "compile", cmd: compileCmd},
		{name: "run", cmd: runCmd},
		{name: "version", cmd: versionCmd},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.cmd.Example, "command should define examples via Example field")
			assert.NotContains(t, tt.cmd.Long, "\nExamples:\n", "command long help should not embed an Examples section")
			assert.Contains(t, tt.cmd.Example, "gh aw ", "command examples should contain gh aw invocations")
			assert.False(t, strings.HasPrefix(tt.cmd.Example, "Examples:"), "command Example field should contain example lines only")
		})
	}
}
