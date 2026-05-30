//go:build !integration

package errorutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/errorutil"
)

// TestSpec_PublicAPI_IsNotFoundError validates the documented behavior of
// IsNotFoundError as described in the errorutil README.md.
//
// Specification:
//   - Returns true when err indicates a "not found" condition by matching
//     case-insensitive "404" or "not found" text.
//   - Returns false for nil and non-matching errors.
func TestSpec_PublicAPI_IsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "documented: nil returns false", err: nil, want: false},
		{name: "documented: numeric 404 match", err: errors.New("HTTP 404: Not Found"), want: true},
		{name: "documented: lowercase not found", err: errors.New("not found"), want: true},
		{name: "documented: case-insensitive uppercase NOT FOUND", err: errors.New("RESOURCE NOT FOUND"), want: true},
		{name: "documented: case-insensitive mixed Not Found", err: errors.New("Resource Not Found"), want: true},
		{name: "documented: wrapped not found", err: fmt.Errorf("ctx: %w", errors.New("not found")), want: true},
		{name: "documented: non-matching error returns false", err: errors.New("something else went wrong"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorutil.IsNotFoundError(tt.err)
			assert.Equal(t, tt.want, got, "IsNotFoundError(%v) mismatch for: %s", tt.err, tt.name)
		})
	}
}

// TestSpec_PublicAPI_IsForbiddenError validates the documented behavior of
// IsForbiddenError as described in the errorutil README.md.
//
// Specification:
//   - Returns true when err indicates an HTTP-style 403/"forbidden" response
//     by matching case-insensitive patterns like "HTTP 403" or "403 Forbidden".
//   - Returns false for nil and non-matching errors.
//   - Design note: requires HTTP-style status context so unrelated phrases
//     like "forbidden character" are not misclassified.
func TestSpec_PublicAPI_IsForbiddenError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "documented: nil returns false", err: nil, want: false},
		{name: "documented: HTTP 403 pattern", err: errors.New("HTTP 403: Forbidden"), want: true},
		{name: "documented: 403 Forbidden pattern", err: errors.New("403 Forbidden"), want: true},
		{name: "documented: case-insensitive http 403", err: errors.New("http 403: forbidden"), want: true},
		{name: "documented: case-insensitive HTTP 403 FORBIDDEN", err: errors.New("HTTP 403: FORBIDDEN"), want: true},
		{name: "documented: wrapped HTTP 403", err: fmt.Errorf("api: %w", errors.New("HTTP 403: Forbidden")), want: true},
		{name: "documented design note: 'forbidden character' is not misclassified", err: errors.New("invalid forbidden character in query"), want: false},
		{name: "documented: non-matching error returns false", err: errors.New("some other failure"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorutil.IsForbiddenError(tt.err)
			assert.Equal(t, tt.want, got, "IsForbiddenError(%v) mismatch for: %s", tt.err, tt.name)
		})
	}
}

// TestSpec_PublicAPI_IsGoneError validates the documented behavior of
// IsGoneError as described in the errorutil README.md.
//
// Specification:
//   - Returns true when err indicates an HTTP-style 410/"gone" response
//     by matching case-insensitive patterns like "HTTP 410" or "410 Gone".
//   - Returns false for nil and non-matching errors.
//   - Design note: requires HTTP-style status context so unrelated phrases
//     like "gone away" are not misclassified.
func TestSpec_PublicAPI_IsGoneError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "documented: nil returns false", err: nil, want: false},
		{name: "documented: HTTP 410 pattern", err: errors.New("HTTP 410: Gone"), want: true},
		{name: "documented: 410 Gone pattern", err: errors.New("410 Gone"), want: true},
		{name: "documented: case-insensitive http 410", err: errors.New("http 410: gone"), want: true},
		{name: "documented: case-insensitive HTTP 410 GONE", err: errors.New("HTTP 410: GONE"), want: true},
		{name: "documented: wrapped HTTP 410", err: fmt.Errorf("api: %w", errors.New("HTTP 410: Gone")), want: true},
		{name: "documented design note: 'gone away' is not misclassified", err: errors.New("connection has gone away"), want: false},
		{name: "documented: non-matching error returns false", err: errors.New("totally unrelated"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorutil.IsGoneError(tt.err)
			assert.Equal(t, tt.want, got, "IsGoneError(%v) mismatch for: %s", tt.err, tt.name)
		})
	}
}

// TestSpec_UsageExample_ErrorClassifiers validates that the documented usage
// example pattern compiles and runs.
//
// Specification (Usage Examples):
//
//	if errorutil.IsNotFoundError(err) { ... }
//	if errorutil.IsForbiddenError(err) { ... }
//	if errorutil.IsGoneError(err) { ... }
func TestSpec_UsageExample_ErrorClassifiers(t *testing.T) {
	notFound := errors.New("HTTP 404: Not Found")
	forbidden := errors.New("HTTP 403: Forbidden")
	gone := errors.New("HTTP 410: Gone")

	assert.True(t, errorutil.IsNotFoundError(notFound), "usage example: 404 path triggered")
	assert.True(t, errorutil.IsForbiddenError(forbidden), "usage example: 403 path triggered")
	assert.True(t, errorutil.IsGoneError(gone), "usage example: 410 path triggered")

	assert.False(t, errorutil.IsForbiddenError(notFound), "documented: classifiers are exclusive — 404 is not forbidden")
	assert.False(t, errorutil.IsGoneError(notFound), "documented: classifiers are exclusive — 404 is not gone")
	assert.False(t, errorutil.IsNotFoundError(forbidden), "documented: classifiers are exclusive — 403 is not not-found")
}
