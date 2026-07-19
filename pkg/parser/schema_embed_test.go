//go:build !integration

package parser

import (
	"encoding/json"
	"testing"
)

// TestEmbeddedSchemasAreValid verifies that every schema file embedded via //go:embed
// is both valid JSON and a compilable JSON schema.
//
// This test acts as a fast guard: because //go:embed re-reads the files at compile
// time, running `go test ./pkg/parser/...` (or `make test-unit`) is equivalent to
// rebuilding the binary with respect to schema embedding.  Any change to a file
// under pkg/parser/schemas/ that introduces malformed JSON or an invalid schema
// will therefore fail here before the broken schema can reach a deployed binary.
func TestEmbeddedSchemasAreValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		schemaVar string
		schemaURL string
	}{
		{
			name:      "main_workflow_schema",
			schemaVar: mainWorkflowSchema,
			schemaURL: "http://contoso.com/main-workflow-schema.json",
		},
		{
			name:      "mcp_config_schema",
			schemaVar: mcpConfigSchema,
			schemaURL: "http://contoso.com/mcp-config-schema.json",
		},
		{
			name:      "repo_config_schema",
			schemaVar: RepoConfigSchema,
			schemaURL: "http://contoso.com/repo-config-schema.json",
		},
		{
			name:      "aw_manifest_schema",
			schemaVar: awManifestSchema,
			schemaURL: "http://contoso.com/aw-manifest-schema.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 1. Verify the embedded content is non-empty.
			if len(tc.schemaVar) == 0 {
				t.Fatalf("embedded schema %q is empty; check the //go:embed directive and the file path", tc.name)
			}

			// 2. Verify the embedded content is valid JSON.
			var doc any
			if err := json.Unmarshal([]byte(tc.schemaVar), &doc); err != nil {
				t.Fatalf("embedded schema %q is not valid JSON: %v\nRun `make build` after editing files under pkg/parser/schemas/", tc.name, err)
			}

			// 3. Verify the schema can be compiled by the JSON schema compiler.
			if _, err := CompileSchema(tc.schemaVar, tc.schemaURL); err != nil {
				t.Fatalf("embedded schema %q failed to compile: %v\nRun `make build` after editing files under pkg/parser/schemas/", tc.name, err)
			}
		})
	}
}
