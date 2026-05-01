//go:build !integration

package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCoolDownFlag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "default 7d",
			input: "7d",
			want:  7 * 24 * time.Hour,
		},
		{
			name:  "0d disables cooldown",
			input: "0d",
			want:  0,
		},
		{
			name:  "0 disables cooldown",
			input: "0",
			want:  0,
		},
		{
			name:  "hours format",
			input: "168h",
			want:  168 * time.Hour,
		},
		{
			name:  "single day",
			input: "1d",
			want:  24 * time.Hour,
		},
		{
			name:    "negative days",
			input:   "-1d",
			wantErr: true,
		},
		{
			name:    "negative duration",
			input:   "-1h",
			wantErr: true,
		},
		{
			name:    "invalid string",
			input:   "foobar",
			wantErr: true,
		},
		{
			name:    "invalid days value",
			input:   "xd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCoolDownFlag(tt.input)
			if tt.wantErr {
				assert.Error(t, err, "parseCoolDownFlag(%q) should return error", tt.input)
				return
			}
			require.NoError(t, err, "parseCoolDownFlag(%q) unexpected error", tt.input)
			assert.Equal(t, tt.want, got, "parseCoolDownFlag(%q) duration mismatch", tt.input)
		})
	}
}

func TestIsExemptFromCoolDown(t *testing.T) {
	tests := []struct {
		repo string
		want bool
	}{
		{"actions/checkout", true},
		{"actions/setup-node", true},
		{"actions/cache/restore", true},
		{"github/codeql-action", true},
		{"github/gh-aw", true},
		{"github/codeql-action/upload-sarif", true},
		{"owner/repo", false},
		{"myorg/my-action", false},
		{"notactions/checkout", false},
		{"notgithub/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			got := isExemptFromCoolDown(tt.repo)
			assert.Equal(t, tt.want, got, "isExemptFromCoolDown(%q) result mismatch", tt.repo)
		})
	}
}

func TestCheckReleaseCoolDown_Disabled(t *testing.T) {
	result := checkReleaseCoolDown(context.Background(), "owner/repo", "v1.2.0", 0)
	assert.False(t, result.InCoolDown, "cooldown should be disabled when duration is 0")
}

func TestCheckReleaseCoolDown_OldRelease(t *testing.T) {
	orig := getReleasePublishedAtFn
	defer func() { getReleasePublishedAtFn = orig }()

	// Simulate a release published 10 days ago (older than 7-day cooldown)
	getReleasePublishedAtFn = func(_ context.Context, _, _ string) (time.Time, error) {
		return time.Now().Add(-10 * 24 * time.Hour), nil
	}

	result := checkReleaseCoolDown(context.Background(), "owner/repo", "v1.2.0", 7*24*time.Hour)
	assert.False(t, result.InCoolDown, "release older than cooldown period should not be blocked")
}

func TestCheckReleaseCoolDown_RecentRelease(t *testing.T) {
	orig := getReleasePublishedAtFn
	defer func() { getReleasePublishedAtFn = orig }()

	// Simulate a release published 2 days ago (within 7-day cooldown)
	getReleasePublishedAtFn = func(_ context.Context, _, _ string) (time.Time, error) {
		return time.Now().Add(-2 * 24 * time.Hour), nil
	}

	result := checkReleaseCoolDown(context.Background(), "owner/repo", "v1.2.0", 7*24*time.Hour)
	assert.True(t, result.InCoolDown, "release within cooldown period should be blocked")
	assert.Contains(t, result.Message, "v1.2.0", "message should mention the release tag")
	assert.Contains(t, result.Message, "cool down", "message should mention cooldown")
	assert.False(t, result.PublishedAt.IsZero(), "PublishedAt should be populated for caching")
}

func TestCheckReleaseCoolDown_OldReleaseReturnsPublishedAt(t *testing.T) {
	orig := getReleasePublishedAtFn
	defer func() { getReleasePublishedAtFn = orig }()

	published := time.Now().Add(-10 * 24 * time.Hour)
	getReleasePublishedAtFn = func(_ context.Context, _, _ string) (time.Time, error) {
		return published, nil
	}

	result := checkReleaseCoolDown(context.Background(), "owner/repo", "v1.2.0", 7*24*time.Hour)
	assert.False(t, result.InCoolDown, "old release should not be blocked")
	assert.True(t, result.PublishedAt.Equal(published), "PublishedAt should be populated even when not in cooldown")
}

func TestCheckReleaseCoolDownWithDate_InCoolDown(t *testing.T) {
	publishedAt := time.Now().Add(-2 * 24 * time.Hour)
	result := checkReleaseCoolDownWithDate("owner/repo", "v1.2.0", publishedAt, 7*24*time.Hour)
	assert.True(t, result.InCoolDown, "release 2d old should be in cooldown with 7d window")
	assert.Contains(t, result.Message, "v1.2.0", "message should mention the tag")
}

func TestCheckReleaseCoolDownWithDate_NotInCoolDown(t *testing.T) {
	publishedAt := time.Now().Add(-10 * 24 * time.Hour)
	result := checkReleaseCoolDownWithDate("owner/repo", "v1.2.0", publishedAt, 7*24*time.Hour)
	assert.False(t, result.InCoolDown, "release 10d old should not be in cooldown with 7d window")
}

func TestCheckReleaseCoolDownWithDate_ZeroDuration(t *testing.T) {
	publishedAt := time.Now()
	result := checkReleaseCoolDownWithDate("owner/repo", "v1.2.0", publishedAt, 0)
	assert.False(t, result.InCoolDown, "zero cooldown should always allow update")
}

func TestCheckReleaseCoolDownWithDate_FutureTimestamp(t *testing.T) {
	// Simulate a future published_at (clock skew / API returning future time).
	// The release should be treated as just-published and kept in cooldown.
	publishedAt := time.Now().Add(1 * time.Hour)
	result := checkReleaseCoolDownWithDate("owner/repo", "v1.2.0", publishedAt, 7*24*time.Hour)
	assert.True(t, result.InCoolDown, "future timestamp should be treated as just-published and kept in cooldown")
}

func TestCheckReleaseCoolDown_FetchError(t *testing.T) {
	orig := getReleasePublishedAtFn
	defer func() { getReleasePublishedAtFn = orig }()

	// Simulate API error
	getReleasePublishedAtFn = func(_ context.Context, _, _ string) (time.Time, error) {
		return time.Time{}, errors.New("network error")
	}

	// Fail-open: when date can't be fetched, allow update
	result := checkReleaseCoolDown(context.Background(), "owner/repo", "v1.2.0", 7*24*time.Hour)
	assert.False(t, result.InCoolDown, "should allow update when published date cannot be fetched")
}

func TestFormatCoolDownDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{7 * 24 * time.Hour, "7d"},
		{7*24*time.Hour + 12*time.Hour, "7d12h"},
		{24 * time.Hour, "1d"},
		{48 * time.Hour, "2d"},
		{12 * time.Hour, "12h"},
		{1 * time.Hour, "1h"},
		{30 * time.Minute, "< 1h"},
		{0, "< 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCoolDownDuration(tt.duration)
			assert.Equal(t, tt.want, got, "formatCoolDownDuration(%v) mismatch", tt.duration)
		})
	}
}
