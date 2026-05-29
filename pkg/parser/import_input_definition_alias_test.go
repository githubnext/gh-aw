//go:build !integration

package parser

import (
	"reflect"
	"testing"

	"github.com/github/gh-aw/pkg/types"
)

func TestImportInputDefinitionAliasesSharedType(t *testing.T) {
	if reflect.TypeFor[ImportInputDefinition]() != reflect.TypeFor[types.InputDefinition]() {
		t.Fatal("ImportInputDefinition should alias types.InputDefinition")
	}

	input := ImportInputDefinition{Default: true}
	if got := input.GetDefaultAsString(); got != "true" {
		t.Fatalf("GetDefaultAsString() through alias = %q, want %q", got, "true")
	}
}
