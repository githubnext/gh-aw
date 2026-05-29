//go:build !integration

package errorutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/errorutil"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "404 numeric literal", err: errors.New("HTTP 404: Not Found"), want: true},
		{name: "lowercase not found", err: errors.New("failed to fetch file: not found"), want: true},
		{name: "uppercase NOT FOUND", err: errors.New("RESOURCE NOT FOUND"), want: true},
		{name: "wrapped lowercase not found", err: fmt.Errorf("request failed: %w", errors.New("not found")), want: true},
		{name: "bare 404 in message", err: errors.New("server returned 404"), want: true},
		{name: "Could not resolve (DNS)", err: errors.New("Could not resolve host"), want: false},
		{name: "401 Unauthorized", err: errors.New("HTTP 401: Unauthorized"), want: false},
		{name: "500 Internal Server Error", err: errors.New("HTTP 500: Internal Server Error"), want: false},
		{name: "generic error", err: errors.New("something went wrong"), want: false},
		{name: "410 Gone", err: errors.New("HTTP 410: Gone"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorutil.IsNotFoundError(tt.err)
			assert.Equal(t, tt.want, got, "IsNotFoundError result mismatch for subtest %s", tt.name)
		})
	}
}

func TestIsForbiddenError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "http 403 forbidden", err: errors.New("HTTP 403: Forbidden"), want: true},
		{name: "parenthesized http 403", err: errors.New("gh: API rate limit exceeded (HTTP 403)"), want: true},
		{name: "status 403", err: errors.New("request failed with status 403"), want: true},
		{name: "wrapped http 403", err: fmt.Errorf("request failed: %w", errors.New("HTTP 403: access denied")), want: true},
		{name: "forbidden without http status", err: errors.New("request forbidden"), want: false},
		{name: "forbidden character", err: errors.New("invalid forbidden character in query"), want: false},
		{name: "bare 403 in message", err: errors.New("server returned 403"), want: false},
		{name: "404 Not Found", err: errors.New("HTTP 404: Not Found"), want: false},
		{name: "generic error", err: errors.New("something went wrong"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorutil.IsForbiddenError(tt.err)
			assert.Equal(t, tt.want, got, "IsForbiddenError result mismatch for subtest %s", tt.name)
		})
	}
}

func TestIsGoneError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "http 410 gone", err: errors.New("HTTP 410: Gone"), want: true},
		{name: "parenthesized http 410", err: errors.New("gh: workflow logs expired (HTTP 410)"), want: true},
		{name: "status 410", err: errors.New("request failed with status 410"), want: true},
		{name: "wrapped http 410", err: fmt.Errorf("request failed: %w", errors.New("HTTP 410: logs unavailable")), want: true},
		{name: "gone without http status", err: errors.New("artifact gone"), want: false},
		{name: "gone away", err: errors.New("connection has gone away"), want: false},
		{name: "bare 410 in message", err: errors.New("server returned 410"), want: false},
		{name: "403 Forbidden", err: errors.New("HTTP 403: Forbidden"), want: false},
		{name: "generic error", err: errors.New("something went wrong"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorutil.IsGoneError(tt.err)
			assert.Equal(t, tt.want, got, "IsGoneError result mismatch for subtest %s", tt.name)
		})
	}
}
