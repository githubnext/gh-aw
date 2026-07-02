//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiCheckoutCrossOrgGitHubTokens verifies that multiple checkouts targeting
// different organizations, each authenticated with a distinct github-token, produce
// separate checkout steps each with the correct per-org PAT, and that the checkout
// manifest env-vars are wired to the matching tokens.
func TestMultiCheckoutCrossOrgGitHubTokens(t *testing.T) {
	tmpDir := testutil.TempDir(t, "multi-checkout-tokens-test")

	content := `---
name: Multi-checkout cross-org PATs
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
checkout:
  - repository: org-a/repo-one
    path: repos/org-a
    github-token: ${{ secrets.ORG_A_PAT }}
  - repository: org-b/repo-two
    path: repos/org-b
    github-token: ${{ secrets.ORG_B_PAT }}
  - repository: org-c/repo-three
    path: repos/org-c
    github-token: ${{ secrets.ORG_C_PAT }}
---

# Multi-checkout cross-org PATs
`
	mdPath := filepath.Join(tmpDir, "multi-checkout-tokens.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdPath), "Workflow with multiple cross-org github-tokens should compile")

	lockPath := filepath.Join(tmpDir, "multi-checkout-tokens.lock.yml")
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	compiled := string(compiledBytes)

	// Each additional checkout step must be emitted with its correct target repository and path.
	assert.Contains(t, compiled, "name: Checkout org-a/repo-one into repos/org-a",
		"Checkout step for org-a/repo-one must be emitted")
	assert.Contains(t, compiled, "name: Checkout org-b/repo-two into repos/org-b",
		"Checkout step for org-b/repo-two must be emitted")
	assert.Contains(t, compiled, "name: Checkout org-c/repo-three into repos/org-c",
		"Checkout step for org-c/repo-three must be emitted")

	// Every cross-org checkout must set persist-credentials: false (no credential leakage).
	// We assert the token appears before persist-credentials: false in the expected block.
	assert.Contains(t, compiled, "repository: org-a/repo-one",
		"org-a checkout must declare its repository")
	assert.Contains(t, compiled, "repository: org-b/repo-two",
		"org-b checkout must declare its repository")
	assert.Contains(t, compiled, "repository: org-c/repo-three",
		"org-c checkout must declare its repository")

	// Each checkout must inject its org-specific PAT via token:.
	assert.Contains(t, compiled, "token: ${{ secrets.ORG_A_PAT }}",
		"org-a checkout must use ORG_A_PAT")
	assert.Contains(t, compiled, "token: ${{ secrets.ORG_B_PAT }}",
		"org-b checkout must use ORG_B_PAT")
	assert.Contains(t, compiled, "token: ${{ secrets.ORG_C_PAT }}",
		"org-c checkout must use ORG_C_PAT")

	// All cross-repo checkouts must disable credential persistence.
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-a/repo-one",
		"org-a checkout must disable credential persistence")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-b/repo-two",
		"org-b checkout must disable credential persistence")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-c/repo-three",
		"org-c checkout must disable credential persistence")

	// The checkout manifest must record all three repos so safe-outputs handlers can
	// resolve the correct on-disk path and token for each org without network calls.
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_MANIFEST_COUNT: "3"`,
		"Checkout manifest must declare three entries")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_0: "org-a/repo-one"`,
		"Manifest must record org-a/repo-one as entry 0")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_0: "repos/org-a"`,
		"Manifest must record repos/org-a as entry 0 path")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_0: ${{ secrets.ORG_A_PAT }}",
		"Manifest entry 0 must wire ORG_A_PAT")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_1: "org-b/repo-two"`,
		"Manifest must record org-b/repo-two as entry 1")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_1: "repos/org-b"`,
		"Manifest must record repos/org-b as entry 1 path")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_1: ${{ secrets.ORG_B_PAT }}",
		"Manifest entry 1 must wire ORG_B_PAT")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_2: "org-c/repo-three"`,
		"Manifest must record org-c/repo-three as entry 2")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_2: "repos/org-c"`,
		"Manifest must record repos/org-c as entry 2 path")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_2: ${{ secrets.ORG_C_PAT }}",
		"Manifest entry 2 must wire ORG_C_PAT")

	// No app-token minting steps should be emitted when only github-tokens are used.
	assert.NotContains(t, compiled, "id: checkout-app-token-",
		"No app-token minting steps should appear for github-token-only checkouts")
}

// TestMultiCheckoutCrossOrgGitHubApps verifies that multiple checkouts targeting
// different organizations, each authenticated via a distinct github-app, produce a
// separate app-token minting step per checkout, and that each checkout step and the
// manifest correctly reference the minted step output for the corresponding index.
func TestMultiCheckoutCrossOrgGitHubApps(t *testing.T) {
	tmpDir := testutil.TempDir(t, "multi-checkout-apps-test")

	content := `---
name: Multi-checkout cross-org GitHub Apps
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
checkout:
  - repository: org-a/repo-one
    path: repos/org-a
    github-app:
      app-id: ${{ vars.ORG_A_APP_ID }}
      private-key: ${{ secrets.ORG_A_PRIVATE_KEY }}
  - repository: org-b/repo-two
    path: repos/org-b
    github-app:
      app-id: ${{ vars.ORG_B_APP_ID }}
      private-key: ${{ secrets.ORG_B_PRIVATE_KEY }}
---

# Multi-checkout cross-org GitHub Apps
`
	mdPath := filepath.Join(tmpDir, "multi-checkout-apps.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdPath), "Workflow with multiple cross-org github-apps should compile")

	lockPath := filepath.Join(tmpDir, "multi-checkout-apps.lock.yml")
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	compiled := string(compiledBytes)

	// One app-token minting step must be emitted per github-app checkout, each with a
	// unique step ID derived from its position in the checkout list.
	assert.Contains(t, compiled, "id: checkout-app-token-0",
		"App token minting step for checkout index 0 must be emitted")
	assert.Contains(t, compiled, "id: checkout-app-token-1",
		"App token minting step for checkout index 1 must be emitted")

	// The minting steps must reference the correct app credentials for each org.
	assert.Contains(t, compiled, "client-id: ${{ vars.ORG_A_APP_ID }}",
		"App token minting for org-a must use ORG_A_APP_ID")
	assert.Contains(t, compiled, "private-key: ${{ secrets.ORG_A_PRIVATE_KEY }}",
		"App token minting for org-a must use ORG_A_PRIVATE_KEY")
	assert.Contains(t, compiled, "client-id: ${{ vars.ORG_B_APP_ID }}",
		"App token minting for org-b must use ORG_B_APP_ID")
	assert.Contains(t, compiled, "private-key: ${{ secrets.ORG_B_PRIVATE_KEY }}",
		"App token minting for org-b must use ORG_B_PRIVATE_KEY")

	// The owner: field must be derived from the repository's org component so each
	// minting step requests a token scoped to the correct organization.
	assert.Contains(t, compiled, "owner: org-a",
		"App token minting step for checkout 0 must scope to org-a")
	assert.Contains(t, compiled, "owner: org-b",
		"App token minting step for checkout 1 must scope to org-b")

	// Each checkout step must consume the minted token from its corresponding step output.
	assert.Contains(t, compiled, "token: ${{ steps.checkout-app-token-0.outputs.token }}",
		"org-a/repo-one checkout must use the minted token from checkout-app-token-0")
	assert.Contains(t, compiled, "token: ${{ steps.checkout-app-token-1.outputs.token }}",
		"org-b/repo-two checkout must use the minted token from checkout-app-token-1")

	// All cross-org checkout steps must disable credential persistence after cloning.
	assert.Contains(t, compiled, "name: Checkout org-a/repo-one into repos/org-a",
		"Checkout step for org-a must be emitted")
	assert.Contains(t, compiled, "name: Checkout org-b/repo-two into repos/org-b",
		"Checkout step for org-b must be emitted")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-a/repo-one",
		"org-a checkout must disable credential persistence")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-b/repo-two",
		"org-b checkout must disable credential persistence")

	// The checkout manifest must wire the app-minted tokens for each entry so that
	// safe-outputs handlers can authenticate against the correct org repo.
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_MANIFEST_COUNT: "2"`,
		"Checkout manifest must declare two entries")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_0: "org-a/repo-one"`,
		"Manifest must record org-a/repo-one as entry 0")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_0: "repos/org-a"`,
		"Manifest must record repos/org-a as entry 0 path")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_0: ${{ steps.checkout-app-token-0.outputs.token }}",
		"Manifest entry 0 must reference the app-minted token for org-a")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_1: "org-b/repo-two"`,
		"Manifest must record org-b/repo-two as entry 1")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_1: "repos/org-b"`,
		"Manifest must record repos/org-b as entry 1 path")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_1: ${{ steps.checkout-app-token-1.outputs.token }}",
		"Manifest entry 1 must reference the app-minted token for org-b")
}

// TestMultiCheckoutCrossOrgMixedAuth verifies that a workflow with a mix of
// github-token and github-app checkouts across multiple organizations correctly
// emits only the necessary app-token minting steps (one per github-app entry),
// wires each checkout with its specific credential, and propagates the correct
// token expressions into the checkout manifest for safe-outputs handlers.
func TestMultiCheckoutCrossOrgMixedAuth(t *testing.T) {
	tmpDir := testutil.TempDir(t, "multi-checkout-mixed-auth-test")

	content := `---
name: Multi-checkout cross-org mixed auth
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
checkout:
  - repository: org-a/repo-one
    path: repos/org-a
    github-token: ${{ secrets.ORG_A_PAT }}
  - repository: org-b/repo-two
    path: repos/org-b
    github-app:
      app-id: ${{ vars.ORG_B_APP_ID }}
      private-key: ${{ secrets.ORG_B_PRIVATE_KEY }}
  - repository: org-c/repo-three
    path: repos/org-c
    github-app:
      app-id: ${{ vars.ORG_C_APP_ID }}
      private-key: ${{ secrets.ORG_C_PRIVATE_KEY }}
---

# Multi-checkout cross-org mixed auth
`
	mdPath := filepath.Join(tmpDir, "multi-checkout-mixed-auth.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdPath), "Workflow with mixed github-token and github-app should compile")

	lockPath := filepath.Join(tmpDir, "multi-checkout-mixed-auth.lock.yml")
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	compiled := string(compiledBytes)

	// The github-token checkout (index 0) must NOT produce an app-token minting step;
	// app-token steps are only emitted for github-app entries.
	assert.NotContains(t, compiled, "id: checkout-app-token-0",
		"No app token minting step for PAT-authenticated checkout at index 0")

	// The two github-app checkouts (indices 1 and 2) must each get a minting step.
	assert.Contains(t, compiled, "id: checkout-app-token-1",
		"App token minting step for github-app checkout at index 1 must be emitted")
	assert.Contains(t, compiled, "id: checkout-app-token-2",
		"App token minting step for github-app checkout at index 2 must be emitted")

	// Minting steps must use the correct credentials and org-scoped owners.
	assert.Contains(t, compiled, "client-id: ${{ vars.ORG_B_APP_ID }}",
		"App token minting for org-b must use ORG_B_APP_ID")
	assert.Contains(t, compiled, "private-key: ${{ secrets.ORG_B_PRIVATE_KEY }}",
		"App token minting for org-b must use ORG_B_PRIVATE_KEY")
	assert.Contains(t, compiled, "owner: org-b",
		"App token minting for checkout 1 must scope to org-b")

	assert.Contains(t, compiled, "client-id: ${{ vars.ORG_C_APP_ID }}",
		"App token minting for org-c must use ORG_C_APP_ID")
	assert.Contains(t, compiled, "private-key: ${{ secrets.ORG_C_PRIVATE_KEY }}",
		"App token minting for org-c must use ORG_C_PRIVATE_KEY")
	assert.Contains(t, compiled, "owner: org-c",
		"App token minting for checkout 2 must scope to org-c")

	// Each checkout step must be emitted and use the correct credential expression.
	assert.Contains(t, compiled, "name: Checkout org-a/repo-one into repos/org-a",
		"Checkout step for org-a must be emitted")
	assert.Contains(t, compiled, "token: ${{ secrets.ORG_A_PAT }}",
		"org-a checkout must inject the PAT directly (no intermediate minting step)")

	assert.Contains(t, compiled, "name: Checkout org-b/repo-two into repos/org-b",
		"Checkout step for org-b must be emitted")
	assert.Contains(t, compiled, "token: ${{ steps.checkout-app-token-1.outputs.token }}",
		"org-b checkout must consume the minted token from checkout-app-token-1")

	assert.Contains(t, compiled, "name: Checkout org-c/repo-three into repos/org-c",
		"Checkout step for org-c must be emitted")
	assert.Contains(t, compiled, "token: ${{ steps.checkout-app-token-2.outputs.token }}",
		"org-c checkout must consume the minted token from checkout-app-token-2")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-a/repo-one",
		"org-a checkout must disable credential persistence")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-b/repo-two",
		"org-b checkout must disable credential persistence")
	assert.Contains(t, compiled, "persist-credentials: false\n          repository: org-c/repo-three",
		"org-c checkout must disable credential persistence")

	// The checkout manifest must carry the correct three entries with their respective
	// credential expressions so safe-outputs handlers can authenticate per-repo.
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_MANIFEST_COUNT: "3"`,
		"Checkout manifest must declare three entries")

	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_0: "org-a/repo-one"`,
		"Manifest entry 0 must be org-a/repo-one")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_0: "repos/org-a"`,
		"Manifest entry 0 must record repos/org-a")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_0: ${{ secrets.ORG_A_PAT }}",
		"Manifest entry 0 must carry the ORG_A_PAT directly")

	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_1: "org-b/repo-two"`,
		"Manifest entry 1 must be org-b/repo-two")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_1: "repos/org-b"`,
		"Manifest entry 1 must record repos/org-b")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_1: ${{ steps.checkout-app-token-1.outputs.token }}",
		"Manifest entry 1 must reference the minted token for org-b")

	assert.Contains(t, compiled, `GH_AW_CHECKOUT_REPO_2: "org-c/repo-three"`,
		"Manifest entry 2 must be org-c/repo-three")
	assert.Contains(t, compiled, `GH_AW_CHECKOUT_PATH_2: "repos/org-c"`,
		"Manifest entry 2 must record repos/org-c")
	assert.Contains(t, compiled, "GH_AW_CHECKOUT_TOKEN_2: ${{ steps.checkout-app-token-2.outputs.token }}",
		"Manifest entry 2 must reference the minted token for org-c")

	// App-token minting steps must appear before their corresponding checkout steps.
	mintIdx1 := strings.Index(compiled, "id: checkout-app-token-1")
	mintIdx2 := strings.Index(compiled, "id: checkout-app-token-2")
	checkoutOrgB := strings.Index(compiled, "name: Checkout org-b/repo-two into repos/org-b")
	checkoutOrgC := strings.Index(compiled, "name: Checkout org-c/repo-three into repos/org-c")
	require.Greater(t, mintIdx1, -1, "checkout-app-token-1 minting step must be present")
	require.Greater(t, mintIdx2, -1, "checkout-app-token-2 minting step must be present")
	require.Greater(t, checkoutOrgB, -1, "Checkout step for org-b must be present")
	require.Greater(t, checkoutOrgC, -1, "Checkout step for org-c must be present")
	assert.Less(t, mintIdx1, checkoutOrgB,
		"checkout-app-token-1 minting step must appear before the org-b checkout step")
	assert.Less(t, mintIdx2, checkoutOrgC,
		"checkout-app-token-2 minting step must appear before the org-c checkout step")
}

// TestMultiCheckoutCrossOrgStepOrdering verifies that app-token minting steps are
// emitted BEFORE their corresponding checkout steps so that each checkout can
// reference the minted token in its token: field without a forward-reference error.
func TestMultiCheckoutCrossOrgStepOrdering(t *testing.T) {
	tmpDir := testutil.TempDir(t, "multi-checkout-ordering-test")

	content := `---
name: Multi-checkout step ordering
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
checkout:
  - repository: org-a/repo-one
    path: repos/org-a
    github-app:
      app-id: ${{ vars.ORG_A_APP_ID }}
      private-key: ${{ secrets.ORG_A_PRIVATE_KEY }}
  - repository: org-b/repo-two
    path: repos/org-b
    github-app:
      app-id: ${{ vars.ORG_B_APP_ID }}
      private-key: ${{ secrets.ORG_B_PRIVATE_KEY }}
---

# Multi-checkout step ordering
`
	mdPath := filepath.Join(tmpDir, "multi-checkout-ordering.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdPath), "Workflow should compile successfully")

	lockPath := filepath.Join(tmpDir, "multi-checkout-ordering.lock.yml")
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	compiled := string(compiledBytes)

	// Verify that app-token minting for each checkout index appears BEFORE the
	// corresponding checkout step in the compiled output. The token expression
	// ${{ steps.checkout-app-token-N.outputs.token }} must resolve at runtime,
	// so the minting step must precede the checkout step that consumes its output.
	mintIdx0 := strings.Index(compiled, "id: checkout-app-token-0")
	mintIdx1 := strings.Index(compiled, "id: checkout-app-token-1")
	checkoutOrgA := strings.Index(compiled, "name: Checkout org-a/repo-one into repos/org-a")
	checkoutOrgB := strings.Index(compiled, "name: Checkout org-b/repo-two into repos/org-b")

	require.Greater(t, mintIdx0, -1, "checkout-app-token-0 minting step must be present")
	require.Greater(t, mintIdx1, -1, "checkout-app-token-1 minting step must be present")
	require.Greater(t, checkoutOrgA, -1, "Checkout step for org-a must be present")
	require.Greater(t, checkoutOrgB, -1, "Checkout step for org-b must be present")

	assert.Less(t, mintIdx0, checkoutOrgA,
		"checkout-app-token-0 minting step must appear before the org-a checkout step")
	assert.Less(t, mintIdx1, checkoutOrgB,
		"checkout-app-token-1 minting step must appear before the org-b checkout step")
	assert.Less(t, mintIdx0, mintIdx1,
		"checkout-app-token steps must be emitted in declaration order")
}
