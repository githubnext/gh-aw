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
		name string
		repo string
		want bool
	}{
		{name: "actions namespace", repo: "actions/checkout", want: true},
		{name: "actions namespace with version", repo: "actions/setup-node", want: true},
		{name: "actions namespace with subpath", repo: "actions/cache/restore", want: true},
		{name: "github namespace", repo: "github/codeql-action", want: true},
		{name: "github namespace own repo", repo: "github/gh-aw", want: true},
		{name: "github namespace with subpath", repo: "github/codeql-action/upload-sarif", want: true},
		{name: "unrecognized org", repo: "owner/repo", want: false},
		{name: "custom org action", repo: "myorg/my-action", want: false},
		{name: "not-actions prefix", repo: "notactions/checkout", want: false},
		{name: "not-github prefix", repo: "notgithub/repo", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExemptFromCoolDown(tt.repo)
			assert.Equal(t, tt.want, got, "isExemptFromCoolDown(%q) result mismatch", tt.repo)
		})
	}
}

func TestCheckReleaseCoolDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coolDown        time.Duration
		publishedAgo    time.Duration
		fetchErr        error
		wantInCoolDown  bool
		wantPublishedAt bool
		wantMessage     string
	}{
		{
			name:           "disabled when duration is 0",
			coolDown:       0,
			wantInCoolDown: false,
		},
		{
			name:            "old release not in cooldown",
			coolDown:        7 * 24 * time.Hour,
			publishedAgo:    10 * 24 * time.Hour,
			wantInCoolDown:  false,
			wantPublishedAt: true,
		},
		{
			name:            "recent release in cooldown",
			coolDown:        7 * 24 * time.Hour,
			publishedAgo:    2 * 24 * time.Hour,
			wantInCoolDown:  true,
			wantPublishedAt: true,
			wantMessage:     "v1.2.0",
		},
		{
			name:           "fetch error allows update",
			coolDown:       7 * 24 * time.Hour,
			fetchErr:       errors.New("network error"),
			wantInCoolDown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var published time.Time
			deps := defaultCoolDownDeps()
			if tt.fetchErr != nil {
				deps.getReleasePublishedAt = func(_ context.Context, _, _ string) (time.Time, error) {
					return time.Time{}, tt.fetchErr
				}
			} else if tt.coolDown > 0 {
				published = time.Now().Add(-tt.publishedAgo)
				deps.getReleasePublishedAt = func(_ context.Context, _, _ string) (time.Time, error) {
					return published, nil
				}
			}

			result := checkReleaseCoolDownWithDeps(context.Background(), deps, "owner/repo", "v1.2.0", tt.coolDown)
			assert.Equal(t, tt.wantInCoolDown, result.InCoolDown, "InCoolDown mismatch for %q", tt.name)
			if tt.wantPublishedAt {
				assert.True(t, result.PublishedAt.Equal(published), "PublishedAt should match the fetched value for %q", tt.name)
			} else {
				assert.True(t, result.PublishedAt.IsZero(), "PublishedAt should be zero for %q", tt.name)
			}
			if tt.wantMessage != "" {
				assert.Contains(t, result.Message, tt.wantMessage, "message should contain %q for %q", tt.wantMessage, tt.name)
			}
		})
	}
}

func TestCheckReleaseCoolDownWithDate_InCoolDown(t *testing.T) {
	publishedAt := time.Now().Add(-2 * 24 * time.Hour)
	result := checkReleaseCoolDownWithDate("owner/repo", "v1.2.0", publishedAt, 7*24*time.Hour)
	assert.True(t, result.InCoolDown, "release 2d old should be in cooldown with 7d window")
	assert.Contains(t, result.Message, "v1.2.0", "message should mention the tag")
	assert.Contains(t, result.Message, "cool down", "message should mention cooldown")
	assert.Contains(t, result.Message, "remaining", "message should mention remaining time")
	assert.Contains(t, result.Message, "owner/repo", "message should mention the repository")
}

func TestCheckReleaseCoolDownWithDate_NotInCoolDown(t *testing.T) {
	publishedAt := time.Now().Add(-10 * 24 * time.Hour)
	result := checkReleaseCoolDownWithDate("owner/repo", "v1.2.0", publishedAt, 7*24*time.Hour)
	assert.False(t, result.InCoolDown, "release 10d old should not be in cooldown with 7d window")
	assert.True(t, result.PublishedAt.IsZero(), "PublishedAt should be zero when not in cooldown")
	assert.Empty(t, result.Message, "Message should be empty when not in cooldown")
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
		{-1 * time.Hour, "< 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCoolDownDuration(tt.duration)
			assert.Equal(t, tt.want, got, "formatCoolDownDuration(%v) mismatch", tt.duration)
		})
	}
}
