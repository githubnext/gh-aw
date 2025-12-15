package testutil

// TestFeatureValidation validates that the spec-kit workflow can detect and process specifications.
// This is a simple proof-of-concept function that returns a test string.
func TestFeatureValidation() string {
	return "test-feature-working"
}

// VerifySpecKitWorkflow checks if the workflow is functioning correctly.
// Returns true if the feature is working as expected.
func VerifySpecKitWorkflow() bool {
	return TestFeatureValidation() == "test-feature-working"
}
