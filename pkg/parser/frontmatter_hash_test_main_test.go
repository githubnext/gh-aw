//go:build !integration

package parser

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain enforces goroutine-leak detection for all unit tests in the parser package.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("net/http.(*http2ClientConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*http2clientConnReadLoop).run"),
	)
}
