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
	t.Run("sets Host to the provided host value", func(t *testing.T) {
		opts := githubapi.ClientOptions("github.com", "token")
		assert.Equal(t, "github.com", opts.Host, "ClientOptions should set Host to the provided host")
	})

	t.Run("sets AuthToken to the provided authToken value", func(t *testing.T) {
		opts := githubapi.ClientOptions("github.com", "mytoken")
		assert.Equal(t, "mytoken", opts.AuthToken, "ClientOptions should set AuthToken to the provided token")
	})

	t.Run("sets Timeout to DefaultHTTPClientTimeout", func(t *testing.T) {
		opts := githubapi.ClientOptions("github.com", "token")
		assert.Equal(t, constants.DefaultHTTPClientTimeout, opts.Timeout, "ClientOptions should set Timeout to constants.DefaultHTTPClientTimeout")
	})

	t.Run("accepts a GHES host", func(t *testing.T) {
		opts := githubapi.ClientOptions("github.example.com", "token")
		assert.Equal(t, "github.example.com", opts.Host, "ClientOptions should accept a custom GHES host")
	})
}
