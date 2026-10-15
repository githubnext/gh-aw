package cli

import (
	"errors"
	"fmt"
	"time"
)

func resolveHealthTimeWindow(config HealthConfig, now time.Time) (time.Time, time.Time, error) {
	if config.StartDate == "" && config.Days != 7 && config.Days != 30 && config.Days != 90 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid days value: %d. Must be 7, 30, or 90", config.Days)
	}
	startDate, endDate, err := resolveLogsDateRange(config.StartDate, config.EndDate, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	resolve := func(flag, resolved string) (time.Time, error) {
		parsed, err := parseFilterDate(resolved)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --%s %q: use YYYY-MM-DD, an RFC3339 timestamp, or a relative time such as -24h: %w", flag, resolved, err)
		}
		if flag == "end-date" && len(resolved) == len("2006-01-02") {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Second)
		}
		return parsed.UTC(), nil
	}
	end := now.UTC().Truncate(time.Second)
	if endDate != "" {
		var err error
		end, err = resolve("end-date", endDate)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	start := end.AddDate(0, 0, -config.Days)
	if startDate != "" {
		var err error
		start, err = resolve("start-date", startDate)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, errors.New("--start-date must be at or before --end-date (default: now)")
	}
	return start, end, nil
}

func (config HealthConfig) healthPeriod() string {
	if !config.explicitTimeWindow {
		return fmt.Sprintf("Current status (up to %d days)", config.Days)
	}
	return fmt.Sprintf("Current status (runs created from %s to %s)", config.StartDate, config.EndDate)
}
