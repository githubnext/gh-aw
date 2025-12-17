package testutil

// TestFeature provides basic test functionality to validate spec-kit workflow
type TestFeature struct {
	name string
}

// NewTestFeature creates a new test feature instance
func NewTestFeature(name string) *TestFeature {
	return &TestFeature{name: name}
}

// GetName returns the feature name
func (tf *TestFeature) GetName() string {
	return tf.name
}

// Validate checks if the feature is valid
func (tf *TestFeature) Validate() bool {
	return tf.name != ""
}
