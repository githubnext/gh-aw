//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// engineDocSyncPath is the path from the pkg/workflow package directory to
// the engines reference document.
const engineDocSyncPath = "../../docs/src/content/docs/reference/engines.md"

// TestEngineDocSync verifies that every built-in engine definition in
// data/engines/ has its engine ID referenced in the engines reference
// documentation.
//
// This test is the structural safeguard against "doc drift": when a new engine
// is added to data/engines/*.md but the docs are not updated, this test fails
// at PR time rather than being discovered and manually patched after the fact.
//
// To fix a failure: add the engine ID (e.g. `antigravity`) to the Available
// Coding Agents table in docs/src/content/docs/reference/engines.md.
func TestEngineDocSync(t *testing.T) {
	// Read the engines reference doc.
	docPath := filepath.Join("..", "..", "docs", "src", "content", "docs", "reference", "engines.md")
	docContent, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("failed to read engines reference doc at %s: %v\n\n"+
			"If the docs file was moved, update engineDocSyncPath in engine_doc_sync_test.go.",
			engineDocSyncPath, err)
	}
	docStr := string(docContent)

	// Load all built-in engine definitions using the same loader used at runtime.
	definitions := loadBuiltinEngineDefinitions()

	var missing []string
	for _, def := range definitions {
		// Check that the engine ID appears in the docs as a backtick-quoted value
		// (e.g. `antigravity`) — the canonical form used in the Available Coding
		// Agents table and throughout the engines reference page.
		if !strings.Contains(docStr, "`"+def.ID+"`") {
			missing = append(missing, def.ID)
		}
	}

	if len(missing) > 0 {
		t.Errorf("the following engine IDs are registered in data/engines/ but missing from the engines reference doc (%s):\n\n  %s\n\n"+
			"Add each missing engine to the Available Coding Agents table in that file.\n"+
			"This check prevents engine-doc drift from recurring.",
			engineDocSyncPath, strings.Join(missing, "\n  "))
	}
}
