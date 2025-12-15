package pkg

// TestFeature demonstrates basic functionality for spec-kit workflow validation
type TestFeature struct {
	Name    string
	Enabled bool
}

// NewTestFeature creates a new test feature instance
func NewTestFeature(name string) *TestFeature {
	return &TestFeature{
		Name:    name,
		Enabled: true,
	}
}

// IsEnabled checks if the test feature is enabled
func (tf *TestFeature) IsEnabled() bool {
	return tf.Enabled
}

// GetName returns the feature name
func (tf *TestFeature) GetName() string {
	return tf.Name
}
