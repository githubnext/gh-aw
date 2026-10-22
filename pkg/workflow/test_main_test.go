//go:build !integration

package workflow

import (
	"net/http"
	"testing"

	"go.uber.org/goleak"
)

// TestMain enforces goroutine-leak detection for all unit tests in the workflow package.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(workflowTestMain{m},
		goleak.IgnoreAnyFunction("net/http.(*http2ClientConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*http2clientConnReadLoop).run"),
	)
}

type workflowTestMain struct {
	goleak.TestingM
}

func (m workflowTestMain) Run() int {
	code := m.TestingM.Run()
	// Catalog downloads share the default transport; close its idle connections
	// before goleak checks for goroutines that outlive the test suite.
	http.DefaultClient.CloseIdleConnections()
	return code
}
