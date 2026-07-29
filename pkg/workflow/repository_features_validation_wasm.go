//go:build js || wasm

// This file provides a no-op implementation of repository feature validation for
// WASM/playground builds (build tags: js || wasm).
//
// # Why validation is skipped in WASM builds
//
// The native (non-WASM) build validates that repository features such as discussions
// and issues are enabled before compiling a workflow that depends on them.  That
// validation requires network access to the GitHub API (GraphQL for discussions, REST
// for issues) and a valid gh CLI credential context – neither of which is available
// inside the browser-based WASM playground.
//
// Skipping validation in this build target is therefore intentional:
//   - The WASM playground runs entirely client-side with no outbound network access to
//     the GitHub API.
//   - There is no authenticated gh CLI session available in the browser environment.
//   - Feature validation is a compile-time advisory check, not a security gate.  Any
//     actual enforcement happens at workflow runtime on the native host.
//
// If the playground ever gains the ability to make authenticated GitHub API calls, this
// stub can be replaced with a real implementation that calls those APIs.
//
// For the native implementation, see repository_features_validation.go.

package workflow

// RepositoryFeatures holds information about repository capabilities.
// In WASM builds this struct is kept for API compatibility with the native build; its
// fields are never populated because feature queries require GitHub API access.
type RepositoryFeatures struct {
	HasDiscussions bool
	HasIssues      bool
}

// validateRepositoryFeatures is a no-op in WASM/playground builds.
//
// The native implementation queries the GitHub API to verify that features such as
// discussions and issues are enabled on the target repository.  Because the WASM
// playground has no network access to the GitHub API and no authenticated gh CLI
// session, that check cannot be performed and is intentionally skipped here.
// Validation will still occur at workflow runtime on the native host.
func (c *Compiler) validateRepositoryFeatures(workflowData *WorkflowData) error {
	return nil
}
