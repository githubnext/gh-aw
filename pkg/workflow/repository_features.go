// This file defines the RepositoryFeatures struct shared by both the native
// (repository_features_validation.go) and WASM/playground (repository_features_validation_wasm.go)
// build variants. It has no build tags so both variants compile against a single
// definition, preventing the two implementations' field lists from drifting apart.

package workflow

// RepositoryFeatures holds cached information about repository capabilities.
// In WASM builds its fields are never populated because feature queries require
// GitHub API access; see repository_features_validation_wasm.go for details.
type RepositoryFeatures struct {
	HasDiscussions bool
	HasIssues      bool
}
