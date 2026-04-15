package workflow

import (
	"fmt"
	"hash/fnv"
)

// generateMaintenanceCron generates a cron schedule based on the minimum expires value in days
// Schedule runs at minimum required frequency to check expirations at appropriate intervals
// Returns cron expression and description.
func generateMaintenanceCron(minExpiresDays int) (string, string) {
	// Use a pseudo-random but deterministic minute (37) to avoid load spikes at :00
	minute := 37

	// Determine frequency based on minimum expires value (in days)
	// Run at least as often as the shortest expiration would need
	if minExpiresDays <= 1 {
		// For 1 day or less, run every 2 hours
		return fmt.Sprintf("%d */2 * * *", minute), "Every 2 hours"
	} else if minExpiresDays == 2 {
		// For 2 days, run every 6 hours
		return fmt.Sprintf("%d */6 * * *", minute), "Every 6 hours"
	} else if minExpiresDays <= 4 {
		// For 3-4 days, run every 12 hours
		return fmt.Sprintf("%d */12 * * *", minute), "Every 12 hours"
	}

	// For more than 4 days, run daily
	return fmt.Sprintf("%d %d * * *", minute, 0), "Daily"
}

// sideRepoCronSeed derives a deterministic 64-bit seed from a repository slug
// using FNV-1a hashing. The seed is used to scatter cron offsets across
// multiple side-repo maintenance workflows so they don't all fire at once.
func sideRepoCronSeed(repoSlug string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(repoSlug))
	return h.Sum64()
}

// generateSideRepoMaintenanceCron generates a scattered cron schedule for a
// side-repo maintenance workflow. The minute (and start hour for sub-daily
// schedules) are derived deterministically from the repository slug so that
// multiple side-repos are spread across the clock face rather than all firing
// at the same moment.
func generateSideRepoMaintenanceCron(repoSlug string, minExpiresDays int) (string, string) {
	seed := sideRepoCronSeed(repoSlug)
	// Derive a deterministic minute in 0-59 from the seed.
	minute := int(seed % 60)

	if minExpiresDays <= 1 {
		// Every 2 hours — vary the starting minute only.
		return fmt.Sprintf("%d */2 * * *", minute), "Every 2 hours"
	} else if minExpiresDays == 2 {
		// Every 6 hours — vary the starting hour within the 6-hour window.
		startHour := int((seed >> 8) % 6)
		return fmt.Sprintf("%d %d,%d,%d,%d * * *", minute, startHour, startHour+6, startHour+12, startHour+18), "Every 6 hours"
	} else if minExpiresDays <= 4 {
		// Every 12 hours — vary the starting hour within the 12-hour window.
		startHour := int((seed >> 8) % 12)
		return fmt.Sprintf("%d %d,%d * * *", minute, startHour, startHour+12), "Every 12 hours"
	}

	// Daily — vary the hour of day (0-23) to spread load.
	hour := int((seed >> 8) % 24)
	return fmt.Sprintf("%d %d * * *", minute, hour), "Daily"
}
