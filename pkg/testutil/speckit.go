package testutil

// SpecKitBasicFeature is a test function to validate the spec-kit-execute workflow
// It returns a simple string to demonstrate that the workflow can:
// 1. Read specification files from .specify/specs/
// 2. Execute tasks in order following the implementation plan
// 3. Create implementation files following TDD
// 4. Generate pull requests with changes
func SpecKitBasicFeature() string {
	return "spec-kit test feature works"
}
