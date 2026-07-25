//go:build !integration

package workflow

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain enforces goroutine-leak detection for all unit tests in the workflow package.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
