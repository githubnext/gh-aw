//go:build !integration

package main

import "testing"

func TestCompileCommandShortFlags(t *testing.T) {
	t.Parallel()
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

	grantFlag := compileCmd.Flags().Lookup("grant")
	if grantFlag == nil {
		t.Fatal("expected --grant flag on compile command")
	}
	if grantFlag.DefValue != "false" {
		t.Fatalf("expected --grant default to be false, got %s", grantFlag.DefValue)
	}

	refreshContainerPinsFlag := compileCmd.Flags().Lookup("refresh-container-pins")
	if refreshContainerPinsFlag == nil {
		t.Fatal("expected --refresh-container-pins flag on compile command")
	}
	if refreshContainerPinsFlag.DefValue != "false" {
		t.Fatalf("expected --refresh-container-pins default to be false, got %s", refreshContainerPinsFlag.DefValue)
	}
}

func TestCompileOptionsPropagateRefreshContainerPins(t *testing.T) {
	config := (&compileCmdOptions{refreshContainerPins: true}).toCompileConfig(nil)
	if !config.RefreshContainerPins {
		t.Fatal("expected RefreshContainerPins to be propagated to CompileConfig")
	}
}
