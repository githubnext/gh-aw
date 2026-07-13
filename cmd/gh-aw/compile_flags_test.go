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
}

func TestCompileValidateFlag(t *testing.T) {
	validateFlag := compileCmd.Flags().Lookup("validate")
	if validateFlag == nil {
		t.Fatal("expected --validate flag on compile command")
	}
	if validateFlag.DefValue != "false" {
		t.Fatalf("expected --validate default to be false, got %s", validateFlag.DefValue)
	}
	if validateFlag.Shorthand != "" {
		t.Fatalf("expected --validate to have no shorthand, got -%s", validateFlag.Shorthand)
	}
}

func TestCompileWatchFlag(t *testing.T) {
	watchFlag := compileCmd.Flags().Lookup("watch")
	if watchFlag == nil {
		t.Fatal("expected --watch flag on compile command")
	}
	if watchFlag.Shorthand != "w" {
		t.Fatalf("expected --watch shorthand to be -w, got -%s", watchFlag.Shorthand)
	}
	if watchFlag.DefValue != "false" {
		t.Fatalf("expected --watch default to be false, got %s", watchFlag.DefValue)
	}
}
