package parser

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var scheduleFuzzyScatterLog = logger.New("parser:schedule_fuzzy_scatter")

// This file contains fuzzy schedule scattering logic that deterministically
// distributes workflow execution times based on workflow identifiers.

// buildWeightedHourPool constructs the weighted pool of hours used for full-day scatter
// patterns. The pool reflects the following distribution:
//
//   - BEST  (weight 3): 02:00–05:59 UTC — low-traffic hours, preferred for maintenance
//   - BROAD (weight 1): 06:00–23:59 UTC — full daytime/evening window
//
// Pool size: 4×3 (BEST) + 18×1 (BROAD) = 12 + 18 = 30 slots.
// BEST represents 12/30 = 40% and BROAD represents 18/30 = 60% of the hour pool.
func buildWeightedHourPool() []int {
	var pool []int

	// BEST: hours 02–05, weight 3 (appear 3 times each)
	for h := 2; h <= 5; h++ {
		pool = append(pool, h, h, h)
	}

	// BROAD: hours 06–23, weight 1
	for h := 6; h <= 23; h++ {
		pool = append(pool, h)
	}

	return pool
}

// buildAvailableMinutes constructs the valid minute values used for the independent
// minute selection in daily scatter patterns. The pool pre-excludes:
//
//   - Hour-boundary windows [0–4] and [55–59] — high-traffic around each hour boundary
//   - EU morning peak [27–33] — ±3 minutes around :30 in hours 06–09
//   - US business-hours peaks [12–18] and [42–48] — ±3 minutes around :15 and :45
//
// Pre-excluding these ranges means avoidPeakMinutes does not need to remap pool
// values, which previously caused clustering: several raw minutes all collapsing to
// the same post-remap value (e.g. 27–33 → 34) and creating artificial collisions.
//
// Remaining valid minutes: [5–11, 19–26, 34–41, 49–54] = 29 values.
func buildAvailableMinutes() []int {
	var pool []int
	for m := 5; m <= 54; m++ {
		// Exclude EU morning peak zone (±3 of :30, affecting hours 06–09)
		if m >= 27 && m <= 33 {
			continue
		}
		// Exclude US business-hours peak zones (±3 of :15 and :45, hours 14–18)
		if m >= 12 && m <= 18 {
			continue
		}
		if m >= 42 && m <= 48 {
			continue
		}
		pool = append(pool, m)
	}
	return pool
}

// weightedHourPool is the pre-computed weighted pool of hours (BEST + BROAD tiers).
var weightedHourPool = buildWeightedHourPool()

// availableMinutes is the pre-computed curated set of valid minutes for scatter
// selection: 29 values spanning [5–11, 19–26, 34–41, 49–54] with hour-boundary
// and peak-traffic ranges pre-excluded (see buildAvailableMinutes).
var availableMinutes = buildAvailableMinutes()

// weightedDailyTimeSlot returns a deterministic (hour, minute) pair for the given
// workflow identifier using two hash operations — one for hour selection and one for
// minute selection — where the minute hash incorporates the hour-pool index as a
// disambiguation component.
//
// The original single-hash approach (972-slot flat pool) produced exact cron-time
// collisions for ~5 workflow pairs per 99 workflows (birthday paradox). Three-way
// collisions caused concurrent token-API bursts that exhausted the 60 req/min quota,
// silently losing safe-output writes.
//
// This implementation reduces collision probability by requiring two independent
// conditions to hold simultaneously for a full (hour, minute) collision:
//
//  1. Both workflows must resolve to the same hour value (not necessarily the same
//     pool index — different indices can yield the same hour via BEST-tier weight-3
//     duplication, e.g. indices 0 and 1 both resolve to hour 2).
//  2. The minute hash of a composite seed (identifier + ":" + hHash index string)
//     must produce the same minute value for both workflows.
//
// The composite seed in condition 2 means that even when two workflows share the same
// resolved hour, they typically receive different minute seeds as long as their hHash
// values differ. Only when both the resolved hour AND the composite-seed minute hash
// collide does a duplicate cron expression occur.
func weightedDailyTimeSlot(identifier string) (int, int) {
	// Hash 1: select hour from the weighted hour pool (preserves BEST/BROAD preference).
	hHash := stableHash(identifier, len(weightedHourPool))
	hour := weightedHourPool[hHash]

	// Hash 2: select minute using a composite seed that encodes the hour-pool index.
	// Incorporating hHash into the seed ensures two workflows that share the same
	// hour via different pool indices (a common outcome of the BEST-tier weight-3
	// duplication) still get different minute hashes as long as their hHash values
	// differ.  When hHash also coincides, the full identifier strings diverge, making
	// collisions on this second hash unlikely for distinct real-world workflow names.
	// avoidPeakMinutes is intentionally NOT called here because availableMinutes
	// already pre-excludes all peak ranges; calling it on pool values would remap
	// multiple distinct raw minutes to the same output, artificially increasing
	// collision counts.
	minuteSeed := fmt.Sprintf("%s:%d", identifier, hHash)
	minute := availableMinutes[stableHash(minuteSeed, len(availableMinutes))]

	return hour, minute
}

// avoidHourBoundary remaps a minute value to avoid the 5-minute window before
// and after each hour (minutes 0–4 and 55–59). These windows are subject to
// usage peaks on GitHub Actions, especially at 00:00 UTC.
// Minutes [0, 4] are shifted to [5, 9] and minutes [55, 59] are shifted to [50, 54],
// keeping all results within [5, 54].
//
// The input is expected to be in the range [0, 59] (a valid minute value).
// Values outside this range are not remapped.
func avoidHourBoundary(minute int) int {
	if minute < 5 {
		return minute + 5
	}
	if minute > 54 {
		return minute - 5
	}
	return minute
}

// avoidPeakMinutes shifts minute values that fall within 3 minutes of known high-traffic
// peak minutes during busy UTC hours:
//
//   - EU morning peak (06:00–09:59 UTC): avoids minutes [27, 33] (±3 around :30),
//     shifting any value in that window to 34 (first minute clearly outside the window)
//   - US business hours (14:00–18:59 UTC): avoids minutes [12, 18] (±3 around :15)
//     and [42, 48] (±3 around :45), shifting to 19 and 49 respectively
//
// All replacement values stay within [5, 54]. This is applied after avoidHourBoundary
// for targeted-scatter patterns where the hour is determined by a user-specified target.
func avoidPeakMinutes(hour, minute int) int {
	// EU morning peak: stay 3 minutes away from :30 in hours 06–09
	if hour >= 6 && hour <= 9 && minute >= 27 && minute <= 33 {
		return 34
	}
	// US business hours (moderate): stay 3 minutes away from :15 and :45 in hours 14–18
	if hour >= 14 && hour <= 18 {
		if minute >= 12 && minute <= 18 {
			return 19
		}
		if minute >= 42 && minute <= 48 {
			return 49
		}
	}
	return minute
}

// stableHash returns a deterministic hash value in the range [0, modulo)
// using FNV-1a hash algorithm, which is stable across platforms and Go versions.
func stableHash(s string, modulo int) int {
	h := fnv.New32a()
	// hash.Hash.Write never returns an error in practice, but check to satisfy gosec G104
	if _, err := h.Write([]byte(s)); err != nil {
		// Return 0 (safe fallback) if write somehow fails
		scheduleFuzzyScatterLog.Printf("Warning: hash write failed: %v", err)
		return 0
	}
	return int(h.Sum32() % uint32(modulo))
}

// ScatterSchedule takes a fuzzy cron expression and a workflow identifier
// and returns a deterministic scattered time for that workflow
func ScatterSchedule(fuzzyCron, workflowIdentifier string) (string, error) {
	scheduleFuzzyScatterLog.Printf("Scattering schedule: fuzzyCron=%s, workflowId=%s", fuzzyCron, workflowIdentifier)
	if !IsFuzzyCron(fuzzyCron) {
		scheduleFuzzyScatterLog.Printf("Invalid fuzzy cron expression: %s", fuzzyCron)
		return "", fmt.Errorf("not a fuzzy schedule: %s", fuzzyCron)
	}

	// For FUZZY:DAILY_AROUND_WEEKDAYS:HH:MM * * *, scatter around the target time on weekdays
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY_AROUND_WEEKDAYS:") {
		// Extract the target hour and minute from FUZZY:DAILY_AROUND_WEEKDAYS:HH:MM
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy daily around weekdays pattern: %s", fuzzyCron)
		}

		// Parse the target time from FUZZY:DAILY_AROUND_WEEKDAYS:HH:MM
		timePart := strings.TrimPrefix(parts[0], "FUZZY:DAILY_AROUND_WEEKDAYS:")
		timeParts := strings.Split(timePart, ":")
		if len(timeParts) != 2 {
			return "", fmt.Errorf("invalid time format in fuzzy daily around weekdays pattern: %s", fuzzyCron)
		}

		targetHour, err := strconv.Atoi(timeParts[0])
		if err != nil || targetHour < 0 || targetHour > 23 {
			return "", fmt.Errorf("invalid target hour in fuzzy daily around weekdays pattern: %s", fuzzyCron)
		}

		targetMinute, err := strconv.Atoi(timeParts[1])
		if err != nil || targetMinute < 0 || targetMinute > 59 {
			return "", fmt.Errorf("invalid target minute in fuzzy daily around weekdays pattern: %s", fuzzyCron)
		}

		// Calculate target time in minutes since midnight
		targetMinutes := targetHour*60 + targetMinute

		// Define the scattering window: ±1 hour (120 minutes total range)
		windowSize := 120 // Total window is 2 hours (±1 hour)

		// Use a stable hash to get a deterministic offset within the window
		hash := stableHash(workflowIdentifier, windowSize)

		// Calculate offset from target time: range is [-60, +59] minutes
		offset := hash - (windowSize / 2)

		// Apply offset to target time
		scatteredMinutes := targetMinutes + offset

		// Handle wrap-around (keep within 0-1439 minutes, which is 0:00-23:59)
		for scatteredMinutes < 0 {
			scatteredMinutes += 24 * 60
		}
		for scatteredMinutes >= 24*60 {
			scatteredMinutes -= 24 * 60
		}

		hour := scatteredMinutes / 60
		minute := avoidPeakMinutes(hour, avoidHourBoundary(scatteredMinutes%60))

		result := fmt.Sprintf("%d %d * * 1-5", minute, hour)
		scheduleFuzzyScatterLog.Printf("FUZZY:DAILY_AROUND_WEEKDAYS scattered: original=%d:%d, scattered=%d:%d, result=%s", targetHour, targetMinute, hour, minute, result)
		// Return scattered daily cron with weekday restriction: minute hour * * 1-5
		return result, nil
	}

	// For FUZZY:DAILY_BETWEEN_WEEKDAYS:START_H:START_M:END_H:END_M * * *, scatter within the time range on weekdays
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY_BETWEEN_WEEKDAYS:") {
		// Extract the start and end times from FUZZY:DAILY_BETWEEN_WEEKDAYS:START_H:START_M:END_H:END_M
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy daily between weekdays pattern: %s", fuzzyCron)
		}

		// Parse the times from FUZZY:DAILY_BETWEEN_WEEKDAYS:START_H:START_M:END_H:END_M
		timePart := strings.TrimPrefix(parts[0], "FUZZY:DAILY_BETWEEN_WEEKDAYS:")
		timeParts := strings.Split(timePart, ":")
		if len(timeParts) != 4 {
			return "", fmt.Errorf("invalid time format in fuzzy daily between weekdays pattern: %s", fuzzyCron)
		}

		startHour, err := strconv.Atoi(timeParts[0])
		if err != nil || startHour < 0 || startHour > 23 {
			return "", fmt.Errorf("invalid start hour in fuzzy daily between weekdays pattern: %s", fuzzyCron)
		}

		startMinute, err := strconv.Atoi(timeParts[1])
		if err != nil || startMinute < 0 || startMinute > 59 {
			return "", fmt.Errorf("invalid start minute in fuzzy daily between weekdays pattern: %s", fuzzyCron)
		}

		endHour, err := strconv.Atoi(timeParts[2])
		if err != nil || endHour < 0 || endHour > 23 {
			return "", fmt.Errorf("invalid end hour in fuzzy daily between weekdays pattern: %s", fuzzyCron)
		}

		endMinute, err := strconv.Atoi(timeParts[3])
		if err != nil || endMinute < 0 || endMinute > 59 {
			return "", fmt.Errorf("invalid end minute in fuzzy daily between weekdays pattern: %s", fuzzyCron)
		}

		// Calculate start and end times in minutes since midnight
		startMinutes := startHour*60 + startMinute
		endMinutes := endHour*60 + endMinute

		// Calculate the range size, handling ranges that cross midnight
		var rangeSize int
		if endMinutes > startMinutes {
			// Normal case: range within a single day (e.g., 9:00 to 17:00)
			rangeSize = endMinutes - startMinutes
		} else {
			// Range crosses midnight (e.g., 22:00 to 02:00)
			rangeSize = (24*60 - startMinutes) + endMinutes
		}

		// Use a stable hash to get a deterministic offset within the range
		hash := stableHash(workflowIdentifier, rangeSize)

		// Calculate the scattered time by adding hash offset to start time
		scatteredMinutes := startMinutes + hash

		// Handle wrap-around for ranges that cross midnight
		if scatteredMinutes >= 24*60 {
			scatteredMinutes -= 24 * 60
		}

		hour := scatteredMinutes / 60
		minute := avoidPeakMinutes(hour, avoidHourBoundary(scatteredMinutes%60))

		result := fmt.Sprintf("%d %d * * 1-5", minute, hour)
		scheduleFuzzyScatterLog.Printf("FUZZY:DAILY_BETWEEN_WEEKDAYS scattered: start=%d:%d, end=%d:%d, scattered=%d:%d, result=%s", startHour, startMinute, endHour, endMinute, hour, minute, result)
		// Return scattered daily cron with weekday restriction: minute hour * * 1-5
		return result, nil
	}

	// For FUZZY:DAILY_AROUND:HH:MM * * *, scatter around the target time
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY_AROUND:") {
		// Extract the target hour and minute from FUZZY:DAILY_AROUND:HH:MM
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy daily around pattern: %s", fuzzyCron)
		}

		// Parse the target time from FUZZY:DAILY_AROUND:HH:MM
		timePart := strings.TrimPrefix(parts[0], "FUZZY:DAILY_AROUND:")
		timeParts := strings.Split(timePart, ":")
		if len(timeParts) != 2 {
			return "", fmt.Errorf("invalid time format in fuzzy daily around pattern: %s", fuzzyCron)
		}

		targetHour, err := strconv.Atoi(timeParts[0])
		if err != nil || targetHour < 0 || targetHour > 23 {
			return "", fmt.Errorf("invalid target hour in fuzzy daily around pattern: %s", fuzzyCron)
		}

		targetMinute, err := strconv.Atoi(timeParts[1])
		if err != nil || targetMinute < 0 || targetMinute > 59 {
			return "", fmt.Errorf("invalid target minute in fuzzy daily around pattern: %s", fuzzyCron)
		}

		// Calculate target time in minutes since midnight
		targetMinutes := targetHour*60 + targetMinute

		// Define the scattering window: ±1 hour (120 minutes total range)
		windowSize := 120 // Total window is 2 hours (±1 hour)

		// Use a stable hash to get a deterministic offset within the window
		hash := stableHash(workflowIdentifier, windowSize)

		// Calculate offset from target time: range is [-60, +59] minutes
		offset := hash - (windowSize / 2)

		// Apply offset to target time
		scatteredMinutes := targetMinutes + offset

		// Handle wrap-around (keep within 0-1439 minutes, which is 0:00-23:59)
		for scatteredMinutes < 0 {
			scatteredMinutes += 24 * 60
		}
		for scatteredMinutes >= 24*60 {
			scatteredMinutes -= 24 * 60
		}

		hour := scatteredMinutes / 60
		minute := avoidPeakMinutes(hour, avoidHourBoundary(scatteredMinutes%60))

		result := fmt.Sprintf("%d %d * * *", minute, hour)
		scheduleFuzzyScatterLog.Printf("FUZZY:DAILY_AROUND scattered: original=%d:%d, scattered=%d:%d, result=%s", targetHour, targetMinute, hour, minute, result)
		// Return scattered daily cron: minute hour * * *
		return result, nil
	}

	// For FUZZY:DAILY_BETWEEN:START_H:START_M:END_H:END_M * * *, scatter within the time range
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY_BETWEEN:") {
		// Extract the start and end times from FUZZY:DAILY_BETWEEN:START_H:START_M:END_H:END_M
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy daily between pattern: %s", fuzzyCron)
		}

		// Parse the times from FUZZY:DAILY_BETWEEN:START_H:START_M:END_H:END_M
		timePart := strings.TrimPrefix(parts[0], "FUZZY:DAILY_BETWEEN:")
		timeParts := strings.Split(timePart, ":")
		if len(timeParts) != 4 {
			return "", fmt.Errorf("invalid time format in fuzzy daily between pattern: %s", fuzzyCron)
		}

		startHour, err := strconv.Atoi(timeParts[0])
		if err != nil || startHour < 0 || startHour > 23 {
			return "", fmt.Errorf("invalid start hour in fuzzy daily between pattern: %s", fuzzyCron)
		}

		startMinute, err := strconv.Atoi(timeParts[1])
		if err != nil || startMinute < 0 || startMinute > 59 {
			return "", fmt.Errorf("invalid start minute in fuzzy daily between pattern: %s", fuzzyCron)
		}

		endHour, err := strconv.Atoi(timeParts[2])
		if err != nil || endHour < 0 || endHour > 23 {
			return "", fmt.Errorf("invalid end hour in fuzzy daily between pattern: %s", fuzzyCron)
		}

		endMinute, err := strconv.Atoi(timeParts[3])
		if err != nil || endMinute < 0 || endMinute > 59 {
			return "", fmt.Errorf("invalid end minute in fuzzy daily between pattern: %s", fuzzyCron)
		}

		// Calculate start and end times in minutes since midnight
		startMinutes := startHour*60 + startMinute
		endMinutes := endHour*60 + endMinute

		// Calculate the range size, handling ranges that cross midnight
		var rangeSize int
		if endMinutes > startMinutes {
			// Normal case: range within a single day (e.g., 9:00 to 17:00)
			rangeSize = endMinutes - startMinutes
		} else {
			// Range crosses midnight (e.g., 22:00 to 02:00)
			rangeSize = (24*60 - startMinutes) + endMinutes
		}

		// Use a stable hash to get a deterministic offset within the range
		hash := stableHash(workflowIdentifier, rangeSize)

		// Calculate the scattered time by adding hash offset to start time
		scatteredMinutes := startMinutes + hash

		// Handle wrap-around for ranges that cross midnight
		if scatteredMinutes >= 24*60 {
			scatteredMinutes -= 24 * 60
		}

		hour := scatteredMinutes / 60
		minute := avoidPeakMinutes(hour, avoidHourBoundary(scatteredMinutes%60))

		result := fmt.Sprintf("%d %d * * *", minute, hour)
		scheduleFuzzyScatterLog.Printf("FUZZY:DAILY_BETWEEN scattered: start=%d:%d, end=%d:%d, scattered=%d:%d, result=%s", startHour, startMinute, endHour, endMinute, hour, minute, result)
		// Return scattered daily cron: minute hour * * *
		return result, nil
	}

	// For FUZZY:DAILY_WEEKDAYS * * *, scatter across the preferred daily time windows on weekdays
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY_WEEKDAYS") {
		hour, minute := weightedDailyTimeSlot(workflowIdentifier)

		result := fmt.Sprintf("%d %d * * 1-5", minute, hour)
		scheduleFuzzyScatterLog.Printf("FUZZY:DAILY_WEEKDAYS scattered: result=%s", result)
		// Return scattered daily cron with weekday restriction: minute hour * * 1-5
		return result, nil
	}

	// For FUZZY:DAILY * * *, scatter across the preferred daily time windows
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY") {
		hour, minute := weightedDailyTimeSlot(workflowIdentifier)

		result := fmt.Sprintf("%d %d * * *", minute, hour)
		scheduleFuzzyScatterLog.Printf("FUZZY:DAILY scattered: result=%s", result)
		// Return scattered daily cron: minute hour * * *
		return result, nil
	}

	// For FUZZY:HOURLY_WEEKDAYS/N * * *, we scatter the minute offset within the hour on weekdays only
	if strings.HasPrefix(fuzzyCron, "FUZZY:HOURLY_WEEKDAYS/") {
		// Extract the interval from FUZZY:HOURLY_WEEKDAYS/N
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy hourly weekdays pattern: %s", fuzzyCron)
		}

		hourlyPart := parts[0]
		intervalStr := strings.TrimPrefix(hourlyPart, "FUZZY:HOURLY_WEEKDAYS/")
		interval, err := strconv.Atoi(intervalStr)
		if err != nil {
			return "", fmt.Errorf("invalid interval in fuzzy hourly weekdays pattern: %s", fuzzyCron)
		}

		// Use 50 valid minutes per hour (avoiding the 5-minute window around each
		// hour boundary) to get a deterministic minute offset in [5, 54].
		minute := stableHash(workflowIdentifier, 50) + 5

		result := fmt.Sprintf("%d */%d * * 1-5", minute, interval)
		scheduleFuzzyScatterLog.Printf("FUZZY:HOURLY_WEEKDAYS/%d scattered: minute=%d, result=%s", interval, minute, result)
		// Return scattered hourly cron with weekday restriction: minute */N * * 1-5
		return result, nil
	}

	// For FUZZY:HOURLY/N * * *, we scatter the minute offset within the hour
	if strings.HasPrefix(fuzzyCron, "FUZZY:HOURLY/") {
		// Extract the interval from FUZZY:HOURLY/N
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy hourly pattern: %s", fuzzyCron)
		}

		hourlyPart := parts[0]
		intervalStr := strings.TrimPrefix(hourlyPart, "FUZZY:HOURLY/")
		interval, err := strconv.Atoi(intervalStr)
		if err != nil {
			return "", fmt.Errorf("invalid interval in fuzzy hourly pattern: %s", fuzzyCron)
		}

		// Use 50 valid minutes per hour (avoiding the 5-minute window around each
		// hour boundary) to get a deterministic minute offset in [5, 54].
		minute := stableHash(workflowIdentifier, 50) + 5

		result := fmt.Sprintf("%d */%d * * *", minute, interval)
		scheduleFuzzyScatterLog.Printf("FUZZY:HOURLY/%d scattered: minute=%d, result=%s", interval, minute, result)
		// Return scattered hourly cron: minute */N * * *
		return result, nil
	}

	// For FUZZY:WEEKLY_AROUND:DOW:HH:MM * * *, scatter around the target time on specific weekday
	if strings.HasPrefix(fuzzyCron, "FUZZY:WEEKLY_AROUND:") {
		// Extract the weekday and target time from FUZZY:WEEKLY_AROUND:DOW:HH:MM
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy weekly around pattern: %s", fuzzyCron)
		}

		// Parse the weekday and time from FUZZY:WEEKLY_AROUND:DOW:HH:MM
		timePart := strings.TrimPrefix(parts[0], "FUZZY:WEEKLY_AROUND:")
		timeParts := strings.Split(timePart, ":")
		if len(timeParts) != 3 {
			return "", fmt.Errorf("invalid format in fuzzy weekly around pattern: %s", fuzzyCron)
		}

		weekday := timeParts[0]
		targetHour, err := strconv.Atoi(timeParts[1])
		if err != nil || targetHour < 0 || targetHour > 23 {
			return "", fmt.Errorf("invalid target hour in fuzzy weekly around pattern: %s", fuzzyCron)
		}

		targetMinute, err := strconv.Atoi(timeParts[2])
		if err != nil || targetMinute < 0 || targetMinute > 59 {
			return "", fmt.Errorf("invalid target minute in fuzzy weekly around pattern: %s", fuzzyCron)
		}

		// Calculate target time in minutes since midnight
		targetMinutes := targetHour*60 + targetMinute

		// Define the scattering window: ±1 hour (120 minutes total range)
		windowSize := 120 // Total window is 2 hours (±1 hour)

		// Use a stable hash to get a deterministic offset within the window
		hash := stableHash(workflowIdentifier, windowSize)

		// Calculate offset from target time: range is [-60, +59] minutes
		offset := hash - (windowSize / 2)

		// Apply offset to target time
		scatteredMinutes := targetMinutes + offset

		// Handle wrap-around (keep within 0-1439 minutes, which is 0:00-23:59)
		for scatteredMinutes < 0 {
			scatteredMinutes += 24 * 60
		}
		for scatteredMinutes >= 24*60 {
			scatteredMinutes -= 24 * 60
		}

		hour := scatteredMinutes / 60
		minute := avoidPeakMinutes(hour, avoidHourBoundary(scatteredMinutes%60))

		result := fmt.Sprintf("%d %d * * %s", minute, hour, weekday)
		scheduleFuzzyScatterLog.Printf("FUZZY:WEEKLY_AROUND scattered: weekday=%s, target=%d:%d, scattered=%d:%d, result=%s", weekday, targetHour, targetMinute, hour, minute, result)
		// Return scattered weekly cron: minute hour * * DOW
		return result, nil
	}

	// For FUZZY:WEEKLY:DOW * * *, we scatter time on specific weekday
	if strings.HasPrefix(fuzzyCron, "FUZZY:WEEKLY:") {
		// Extract the weekday from FUZZY:WEEKLY:DOW
		parts := strings.Split(fuzzyCron, " ")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid fuzzy weekly pattern: %s", fuzzyCron)
		}

		weekdayPart := strings.TrimPrefix(parts[0], "FUZZY:WEEKLY:")
		weekday := weekdayPart

		hour, minute := weightedDailyTimeSlot(workflowIdentifier)

		result := fmt.Sprintf("%d %d * * %s", minute, hour, weekday)
		scheduleFuzzyScatterLog.Printf("FUZZY:WEEKLY:%s scattered: result=%s", weekday, result)
		// Return scattered weekly cron: minute hour * * DOW
		return result, nil
	}

	// For FUZZY:WEEKLY * * *, scatter the weekday deterministically and pick a
	// preferred time from the weighted daily pool.
	if strings.HasPrefix(fuzzyCron, "FUZZY:WEEKLY") {
		weekday := stableHash(workflowIdentifier, 7) // Which day of the week (0-6)
		hour, minute := weightedDailyTimeSlot(workflowIdentifier)

		result := fmt.Sprintf("%d %d * * %d", minute, hour, weekday)
		scheduleFuzzyScatterLog.Printf("FUZZY:WEEKLY scattered: weekday=%d, time=%d:%d, result=%s", weekday, hour, minute, result)
		// Return scattered weekly cron: minute hour * * DOW
		return result, nil
	}

	// For FUZZY:BI_WEEKLY * * *, schedule every 14 days at a preferred time
	if strings.HasPrefix(fuzzyCron, "FUZZY:BI_WEEKLY") {
		hour, minute := weightedDailyTimeSlot(workflowIdentifier)

		result := fmt.Sprintf("%d %d */%d * *", minute, hour, 14)
		scheduleFuzzyScatterLog.Printf("FUZZY:BI_WEEKLY scattered: time=%d:%d, result=%s", hour, minute, result)
		// Convert to cron: We use day-of-month pattern with 14-day interval
		// Schedule every 14 days at the scattered time
		return result, nil
	}

	// For FUZZY:TRI_WEEKLY * * *, schedule every 21 days at a preferred time
	if strings.HasPrefix(fuzzyCron, "FUZZY:TRI_WEEKLY") {
		hour, minute := weightedDailyTimeSlot(workflowIdentifier)

		result := fmt.Sprintf("%d %d */%d * *", minute, hour, 21)
		scheduleFuzzyScatterLog.Printf("FUZZY:TRI_WEEKLY scattered: time=%d:%d, result=%s", hour, minute, result)
		// Convert to cron: We use day-of-month pattern with 21-day interval
		// Schedule every 21 days at the scattered time
		return result, nil
	}

	scheduleFuzzyScatterLog.Printf("Unsupported fuzzy schedule type: %s", fuzzyCron)
	return "", fmt.Errorf("unsupported fuzzy schedule type: %s", fuzzyCron)
}
