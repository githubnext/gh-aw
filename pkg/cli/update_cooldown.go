package cli

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var cooldownLog = logger.New("cli:update_cooldown")

const coolDownFlagUsage = "Cooldown period before applying a new release (e.g. 7d, 24h, 0 to disable). Does not apply to actions/* or github/* repositories"

// parseCoolDownFlag parses a cooldown duration string.
// Accepts day-suffix notation ("7d") or Go duration format ("168h", "0").
// Returns 0 for "0", "0d", or "0h" (cooldown disabled).
func parseCoolDownFlag(s string) (time.Duration, error) {
	if daysStr, ok := strings.CutSuffix(s, "d"); ok {
		days, err := strconv.Atoi(daysStr)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid cooldown value %q: expected a non-negative number of days (e.g. 7d)", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid cooldown value %q: %w", s, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("invalid cooldown value %q: duration must be non-negative", s)
	}
	return d, nil
}

// isExemptFromCoolDown returns true for repositories that bypass the cooldown period.
// Repositories under the "actions/" and "github/" namespaces are always updated immediately.
func isExemptFromCoolDown(repo string) bool {
	base := gitutil.ExtractBaseRepo(repo)
	return strings.HasPrefix(base, "actions/") || strings.HasPrefix(base, "github/")
}

// githubReleaseInfo holds the publication date from a GitHub release API response.
type githubReleaseInfo struct {
	PublishedAt time.Time `json:"published_at"`
}

type coolDownDeps struct {
	getReleasePublishedAt func(ctx context.Context, repo, tag string) (time.Time, error)
}

func defaultCoolDownDeps() coolDownDeps {
	return coolDownDeps{
		getReleasePublishedAt: getReleasePublishedAt,
	}
}

func getReleasePublishedAt(ctx context.Context, repo, tag string) (time.Time, error) {
	client, err := api.NewRESTClient(api.ClientOptions{})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	var release githubReleaseInfo
	if err := client.Get(fmt.Sprintf("repos/%s/releases/tags/%s", repo, url.PathEscape(tag)), &release); err != nil {
		return time.Time{}, fmt.Errorf("failed to fetch release info for %s@%s: %w", repo, tag, err)
	}
	return release.PublishedAt, nil
}

// coolDownCheckResult holds the result of a single cooldown check.
type coolDownCheckResult struct {
	// InCoolDown is true when the release is too recent to apply.
	InCoolDown bool
	// Message is a human-readable explanation when InCoolDown is true.
	Message string
	// PublishedAt is the fetched release publication date. It is populated when
	// checkReleaseCoolDown makes an API call so callers can cache the date.
	PublishedAt time.Time
}

// checkReleaseCoolDownWithDate checks cooldown using an already-known publication date.
// This variant skips the API call and is used when the date is already cached locally.
// A negative age (clock skew / future timestamp) is clamped to 0 so such releases
// are treated as just-published rather than mistakenly allowed past the cooldown.
func checkReleaseCoolDownWithDate(repo, tag string, publishedAt time.Time, coolDown time.Duration) coolDownCheckResult {
	if coolDown <= 0 {
		return coolDownCheckResult{}
	}
	age := max(time.Since(publishedAt), 0) // clamp to 0 for future timestamps (clock skew)
	if age >= coolDown {
		return coolDownCheckResult{}
	}
	remaining := coolDown - age
	ageStr := formatCoolDownDuration(age)
	remainingStr := formatCoolDownDuration(remaining)
	periodStr := formatCoolDownDuration(coolDown)
	msg := fmt.Sprintf("%s@%s was published %s ago and needs to cool down (%s remaining out of %s cooldown period)",
		repo, tag, ageStr, remainingStr, periodStr)
	return coolDownCheckResult{InCoolDown: true, Message: msg}
}

// checkReleaseCoolDown returns a result indicating whether a release tag is
// within the cooldown window and should be skipped.
// The result always includes the fetched PublishedAt date (when available) so that
// callers can cache the date for future runs.
//
// The function is fail-open: if the publication date cannot be fetched (e.g.
// due to network issues), the update is allowed and InCoolDown is false.
func checkReleaseCoolDown(ctx context.Context, repo, tag string, coolDown time.Duration) coolDownCheckResult {
	return checkReleaseCoolDownWithDeps(ctx, defaultCoolDownDeps(), repo, tag, coolDown)
}

func checkReleaseCoolDownWithDeps(ctx context.Context, deps coolDownDeps, repo, tag string, coolDown time.Duration) coolDownCheckResult {
	if coolDown <= 0 {
		return coolDownCheckResult{}
	}

	base := gitutil.ExtractBaseRepo(repo)
	publishedAt, err := deps.getReleasePublishedAt(ctx, base, tag)
	if err != nil {
		cooldownLog.Printf("Failed to get published date for %s@%s (allowing update): %v", repo, tag, err)
		return coolDownCheckResult{}
	}

	r := checkReleaseCoolDownWithDate(repo, tag, publishedAt, coolDown)
	r.PublishedAt = publishedAt
	return r
}

// formatCoolDownDuration formats a duration for display in cooldown messages.
// Uses "Xd Yh" format for multi-day durations, "Xh" for sub-day durations.
func formatCoolDownDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd%dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case int(d.Hours()) > 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return "< 1h"
	}
}
