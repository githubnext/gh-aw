//go:build !integration

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestUsageAppearsBeforeExamplesInHelpOutput(t *testing.T) {
	commands := []*cobra.Command{
		compileCmd,
		disableCmd,
		enableCmd,
		newCmd,
		removeCmd,
		runCmd,
		versionCmd,
	}

	for _, cmd := range commands {
		t.Run(cmd.CommandPath(), func(t *testing.T) {
			var out bytes.Buffer
			originalOut := cmd.OutOrStdout()
			originalErr := cmd.ErrOrStderr()
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			t.Cleanup(func() {
				cmd.SetOut(originalOut)
				cmd.SetErr(originalErr)
			})

			if err := cmd.Help(); err != nil {
				t.Fatalf("help failed for %q: %v", cmd.CommandPath(), err)
			}

			help := out.String()
			usagePos := strings.Index(help, "\nUsage:")
			examplesPos := strings.Index(help, "\nExamples:")

			if usagePos == -1 {
				t.Fatalf("help output for %q is missing Usage section", cmd.CommandPath())
			}
			if examplesPos == -1 {
				t.Fatalf("help output for %q is missing Examples section", cmd.CommandPath())
			}
			if usagePos > examplesPos {
				t.Fatalf("help output for %q has Examples before Usage", cmd.CommandPath())
			}
		})
	}
}
