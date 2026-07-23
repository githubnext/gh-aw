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

	noModelsDevLookupFlag := compileCmd.Flags().Lookup("no-models-dev-lookup")
	if noModelsDevLookupFlag == nil {
		t.Fatal("expected --no-models-dev-lookup flag on compile command")
	}
	if noModelsDevLookupFlag.DefValue != "false" {
		t.Fatalf("expected --no-models-dev-lookup default to be false, got %s", noModelsDevLookupFlag.DefValue)
	}

	grantFlag := compileCmd.Flags().Lookup("grant")
	if grantFlag == nil {
		t.Fatal("expected --grant flag on compile command")
	}
	if grantFlag.DefValue != "false" {
		t.Fatalf("expected --grant default to be false, got %s", grantFlag.DefValue)
	}
}
