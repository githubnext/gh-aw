package workflow

// boolPtr returns a pointer to a bool value.
// This is a shared helper used by both unit and integration tests.
func boolPtr(b bool) *bool {
	return &b
}

// strPtr returns a pointer to a string value.
// This is a shared helper used by tests to create *string values for templatable fields.
func strPtr(s string) *string {
	return &s
}

// mockValidationError helps create validation errors for testing.
// This is a shared helper used by both unit and integration tests.
type mockValidationError struct {
	msg string
}

func (m *mockValidationError) Error() string {
	return m.msg
}
