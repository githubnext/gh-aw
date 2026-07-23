//go:build !integration

package githubapi

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestClientOptions(t *testing.T) {
	opts := ClientOptions("github.com", "token")
	if opts.Host != "github.com" {
		t.Fatalf("ClientOptions() host = %q, want %q", opts.Host, "github.com")
	}
	if opts.AuthToken != "token" {
		t.Fatalf("ClientOptions() auth token = %q, want %q", opts.AuthToken, "token")
	}
	if opts.Timeout != constants.DefaultHTTPClientTimeout {
		t.Fatalf("ClientOptions() timeout = %s, want %s", opts.Timeout, constants.DefaultHTTPClientTimeout)
	}
}
