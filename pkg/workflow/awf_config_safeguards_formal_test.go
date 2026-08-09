//go:build !integration

package workflow

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	formalSnapshotMaxAge      = 7 * 24 * time.Hour
	formalSnapshotDeletionAge = 14 * 24 * time.Hour
	formalSelfHostedSnapshot  = "~/.cache/gh-aw/schema-consistency/last-known-snapshot/"
	formalEphemeralSnapshot   = "/tmp/gh-aw/agent/schema-consistency/last-known-snapshot/"
)

func formalSnapshotExpired(lastRefresh, now time.Time) bool {
	return now.Sub(lastRefresh) > formalSnapshotMaxAge
}

func formalSnapshotShouldDelete(lastRefresh, now time.Time) bool {
	return now.Sub(lastRefresh) > formalSnapshotDeletionAge
}

func formalSnapshotStoragePath(ephemeral bool) string {
	if ephemeral {
		return formalEphemeralSnapshot
	}
	return formalSelfHostedSnapshot
}

func formalEscalationOwner(lastMaintainer, onCallMaintainer string) string {
	if lastMaintainer != "" {
		return lastMaintainer
	}
	return onCallMaintainer
}

func formalEscalationOwnerNonEmpty(owner string) bool {
	return owner != ""
}

func formalAddBusinessDay(start time.Time) time.Time {
	next := start.UTC().AddDate(0, 0, 1)
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func formalEscalationAcknowledgedWithinOneBusinessDay(assignedAt, acknowledgedAt time.Time) bool {
	return !acknowledgedAt.After(formalAddBusinessDay(assignedAt))
}

func formalCoverageVerificationEveryRun(schemaProperties, cliMappedProperties []string) bool {
	mapped := make(map[string]struct{}, len(cliMappedProperties))
	for _, property := range cliMappedProperties {
		mapped[property] = struct{}{}
	}
	for _, property := range schemaProperties {
		if _, ok := mapped[property]; !ok {
			return false
		}
	}
	return true
}

func TestFormalP11_SnapshotExpiryBoundary(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	assert.True(t, formalSnapshotExpired(now.Add(-formalSnapshotMaxAge-time.Nanosecond), now))
	assert.False(t, formalSnapshotExpired(now.Add(-formalSnapshotMaxAge), now))
	assert.False(t, formalSnapshotExpired(now.Add(-167*time.Hour), now))
}

func TestFormalP11_SnapshotShouldDeleteAt14Days(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	assert.True(t, formalSnapshotShouldDelete(now.Add(-formalSnapshotDeletionAge-time.Nanosecond), now))
	assert.False(t, formalSnapshotShouldDelete(now.Add(-formalSnapshotDeletionAge), now))
	assert.False(t, formalSnapshotShouldDelete(now.Add(-formalSnapshotMaxAge-time.Nanosecond), now))
}

func TestFormalP12_SnapshotStoragePathSelection(t *testing.T) {
	assert.Equal(t, formalSelfHostedSnapshot, formalSnapshotStoragePath(false))
	assert.Equal(t, formalEphemeralSnapshot, formalSnapshotStoragePath(true))
	assert.NotEqual(t, filepath.Clean(formalSnapshotStoragePath(false)), filepath.Clean(formalSnapshotStoragePath(true)))
}

func TestFormalP13_EscalationOwnerAssignmentFallbackChain(t *testing.T) {
	assert.Equal(t, "@last-maintainer", formalEscalationOwner("@last-maintainer", "@on-call"))
	assert.Equal(t, "@on-call", formalEscalationOwner("", "@on-call"))
}

func TestFormalP14_EscalationOwnerMustNotBeUnassigned(t *testing.T) {
	assert.True(t, formalEscalationOwnerNonEmpty(formalEscalationOwner("@last-maintainer", "@on-call")))
	assert.True(t, formalEscalationOwnerNonEmpty(formalEscalationOwner("", "@on-call")))
	assert.False(t, formalEscalationOwnerNonEmpty(formalEscalationOwner("", "")))
}

func TestFormalP15_EscalationAcknowledgementWindow(t *testing.T) {
	assignedFriday := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	mondayDeadline := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)

	assert.True(t, formalEscalationAcknowledgedWithinOneBusinessDay(assignedFriday, mondayDeadline))
	assert.False(t, formalEscalationAcknowledgedWithinOneBusinessDay(assignedFriday, mondayDeadline.Add(time.Nanosecond)))
}

func TestFormalP16_CoverageVerificationEveryRun(t *testing.T) {
	schemaProperties := []string{"apiProxy", "container", "mcp"}

	assert.True(t, formalCoverageVerificationEveryRun(schemaProperties, []string{"apiProxy", "container", "mcp"}))
	assert.False(t, formalCoverageVerificationEveryRun(schemaProperties, []string{"apiProxy", "container"}))
}
