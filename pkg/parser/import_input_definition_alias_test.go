//go:build !integration

package parser

import (
	"reflect"
	"testing"

	"github.com/github/gh-aw/pkg/types"
)

func TestImportInputDefinitionAliasesSharedType(t *testing.T) {
	if reflect.TypeOf(ImportInputDefinition{}) != reflect.TypeOf(types.InputDefinition{}) {
		t.Fatal("ImportInputDefinition should alias types.InputDefinition")
	}
}
