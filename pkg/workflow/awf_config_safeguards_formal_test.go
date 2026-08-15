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

func formalRetrievalWarningComplete(failedSources []string, failedAt time.Time) bool {
	return len(failedSources) > 0 && !failedAt.IsZero() && failedAt.Location() == time.UTC
}

func formalSafeguardDecision(canonicalRefreshSucceeded, snapshotFresh bool) (useSnapshot, degraded, allowDestructiveActions bool) {
	if canonicalRefreshSucceeded {
		return false, false, true
	}
	return snapshotFresh, true, false
}

func formalScheduledPersistenceThreshold(previousScheduledUnavailable, currentScheduledUnavailable, currentRunScheduled bool) bool {
	return previousScheduledUnavailable && currentScheduledUnavailable && currentRunScheduled
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

func TestAWFConfigSafeguard_TDRSAFE001_SnapshotExpiryBoundary(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	assert.True(t, formalSnapshotExpired(now.Add(-formalSnapshotMaxAge-time.Nanosecond), now))
	assert.False(t, formalSnapshotExpired(now.Add(-formalSnapshotMaxAge), now))
	assert.False(t, formalSnapshotExpired(now.Add(-167*time.Hour), now))
}

func TestAWFConfigSafeguard_TDRSAFE001_SnapshotShouldDeleteAt14Days(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	assert.True(t, formalSnapshotShouldDelete(now.Add(-formalSnapshotDeletionAge-time.Nanosecond), now))
	assert.False(t, formalSnapshotShouldDelete(now.Add(-formalSnapshotDeletionAge), now))
	assert.False(t, formalSnapshotShouldDelete(now.Add(-formalSnapshotMaxAge-time.Nanosecond), now))
}

func TestAWFConfigSafeguard_TDRSAFE001_SnapshotStoragePathSelection(t *testing.T) {
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

func TestAWFConfigSafeguard_TDRSAFE002_RetrievalWarning(t *testing.T) {
	assert.True(t, formalRetrievalWarningComplete(
		[]string{"docs/awf-config.schema.json"},
		time.Date(2026, 8, 15, 16, 0, 0, 0, time.UTC),
	))
	assert.False(t, formalRetrievalWarningComplete(nil, time.Now().UTC()))
	assert.False(t, formalRetrievalWarningComplete([]string{"docs/awf-config.schema.json"}, time.Time{}))
}

func TestAWFConfigSafeguard_TDRSAFE003_DegradedRunSafety(t *testing.T) {
	useSnapshot, degraded, allowDestructiveActions := formalSafeguardDecision(false, true)
	assert.True(t, useSnapshot)
	assert.True(t, degraded)
	assert.False(t, allowDestructiveActions)

	useSnapshot, degraded, allowDestructiveActions = formalSafeguardDecision(false, false)
	assert.False(t, useSnapshot)
	assert.True(t, degraded)
	assert.False(t, allowDestructiveActions)

	useSnapshot, degraded, allowDestructiveActions = formalSafeguardDecision(true, false)
	assert.False(t, useSnapshot)
	assert.False(t, degraded)
	assert.True(t, allowDestructiveActions)
}

func TestAWFConfigSafeguard_TDRSAFE004_ScheduledPersistence(t *testing.T) {
	assert.True(t, formalScheduledPersistenceThreshold(true, true, true))
	assert.False(t, formalScheduledPersistenceThreshold(true, true, false))
	assert.False(t, formalScheduledPersistenceThreshold(false, true, true))
	assert.False(t, formalScheduledPersistenceThreshold(true, false, true))
}
