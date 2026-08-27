//go:build !integration

package linters_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestCompatJSONConformsToSchema(t *testing.T) {
	schemaJSON, err := os.ReadFile("../../.github/aw/compat.schema.json")
	require.NoError(t, err)

	schema, err := parser.CompileSchema(string(schemaJSON), "https://github.com/github/gh-aw/.github/aw/compat.schema.json")
	require.NoError(t, err)

	configJSON, err := os.ReadFile("../../.github/aw/compat.json")
	require.NoError(t, err)

	var config any
	require.NoError(t, json.Unmarshal(configJSON, &config))
	require.NoError(t, schema.Validate(config))
}
