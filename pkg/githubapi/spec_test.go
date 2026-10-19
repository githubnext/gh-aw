//go:build !integration

package githubapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/githubapi"
)

// TestSpec_PublicAPI_ClientOptions validates the documented behavior of
// ClientOptions as described in the githubapi README.md specification.
func TestSpec_PublicAPI_ClientOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		host      string
		authToken string
	}{
		{name: "GitHub.com values", host: "github.com", authToken: "token"},
		{name: "GHES values", host: "github.example.com", authToken: "app-token"},
		{name: "empty values", host: "", authToken: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := githubapi.ClientOptions(tt.host, tt.authToken)
			assert.Equal(t, tt.host, opts.Host,
				"ClientOptions should preserve the provided host value")
			assert.Equal(t, tt.authToken, opts.AuthToken,
				"ClientOptions should preserve the provided auth token value")
			assert.Equal(t, constants.DefaultHTTPClientTimeout, opts.Timeout,
				"ClientOptions should set Timeout to constants.DefaultHTTPClientTimeout")
		})
	}
}
