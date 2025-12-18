package testutil

// SpecKitTestFeature returns a status message indicating the test feature is working.
func SpecKitTestFeature() string {
	return "Spec-Kit Test Feature: OK"
}

// SpecKitTestFeatureGreeting returns a personalized greeting message.
func SpecKitTestFeatureGreeting(name string) string {
	return "Hello, " + name + "! This is a test feature."
}
