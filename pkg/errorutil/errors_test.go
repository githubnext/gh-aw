package errorutil_test

import (
	"errors"
	"fmt"
	"testing"

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
			if got != tt.want {
				t.Errorf("IsNotFoundError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
