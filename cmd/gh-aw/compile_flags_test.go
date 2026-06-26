//go:build !integration

package main

import "testing"

func TestCompileCommandShortFlags(t *testing.T) {
	forceFlag := compileCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag on compile command")
	}
	if forceFlag.Shorthand != "f" {
		t.Fatalf("expected --force shorthand to be -f, got -%s", forceFlag.Shorthand)
	}

	logicalRepoFlag := compileCmd.Flags().Lookup("logical-repo")
	if logicalRepoFlag == nil {
		t.Fatal("expected --logical-repo flag on compile command")
	}
	if logicalRepoFlag.Shorthand != "l" {
		t.Fatalf("expected --logical-repo shorthand to be -l, got -%s", logicalRepoFlag.Shorthand)
	}
}
