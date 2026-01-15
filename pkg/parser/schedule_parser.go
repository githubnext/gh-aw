package parser

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var scheduleLog = logger.New("parser:schedule_parser")

// ScheduleParser parses human-friendly schedule expressions into cron expressions
type ScheduleParser struct {
	input  string
	tokens []string
	pos    int
}

// ParseSchedule converts a human-friendly schedule expression into a cron expression
// Returns the cron expression and the original friendly format for comments
func ParseSchedule(input string) (cron string, original string, err error) {
	scheduleLog.Printf("Parsing schedule expression: %s", input)
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("schedule expression cannot be empty")
	}

	// If it's already a cron expression (5 fields separated by spaces), return as-is
	if IsCronExpression(input) {
		scheduleLog.Printf("Input is already a valid cron expression: %s", input)
		return input, "", nil
	}

	parser := &ScheduleParser{
		input: input,
	}

	// Tokenize the input
	if err := parser.tokenize(); err != nil {
		scheduleLog.Printf("Tokenization failed: %s", err)
		return "", "", err
	}

	// Parse the tokens
	cronExpr, err := parser.parse()
	if err != nil {
		scheduleLog.Printf("Parsing failed: %s", err)
		return "", "", err
	}

	scheduleLog.Printf("Successfully parsed schedule to cron: %s", cronExpr)
	return cronExpr, input, nil
}

// IsDailyCron checks if a cron expression represents a daily schedule at a fixed time
// (e.g., "0 0 * * *", "30 14 * * *", etc.)
func IsDailyCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	// Daily pattern: minute hour * * *
	// The minute and hour must be specific values (numbers), not wildcards
	// The day-of-month (3rd field) and month (4th field) must be "*"
	// The day-of-week (5th field) must be "*"

	// Check if minute and hour are numeric (not wildcards)
	minute := fields[0]
	hour := fields[1]

	// Minute and hour should be digits only (no *, /, -, ,)
	for _, ch := range minute {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	for _, ch := range hour {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return fields[2] == "*" && fields[3] == "*" && fields[4] == "*"
}

// IsHourlyCron checks if a cron expression represents an hourly interval with a fixed minute
// (e.g., "0 */1 * * *", "30 */2 * * *", etc.)
func IsHourlyCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	// Hourly pattern: minute */N * * * or minute *N * * *
	// The minute must be a specific value (number), not a wildcard
	// The hour must be an interval pattern (*/N)

	minute := fields[0]
	hour := fields[1]

	// Minute should be digits only (no *, /, -, ,)
	for _, ch := range minute {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	// Hour should be an interval pattern like */N
	if !strings.HasPrefix(hour, "*/") {
		return false
	}

	// Check remaining fields are wildcards
	return fields[2] == "*" && fields[3] == "*" && fields[4] == "*"
}

// IsWeeklyCron checks if a cron expression represents a weekly schedule at a fixed time
// (e.g., "0 0 * * 1", "30 14 * * 5", etc.)
func IsWeeklyCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	// Weekly pattern: minute hour * * DOW
	// The minute and hour must be specific values (numbers), not wildcards
	// The day-of-month (3rd field) and month (4th field) must be "*"
	// The day-of-week (5th field) must be a specific day (0-6)

	// Check if minute and hour are numeric (not wildcards)
	minute := fields[0]
	hour := fields[1]

	// Minute and hour should be digits only (no *, /, -, ,)
	for _, ch := range minute {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	for _, ch := range hour {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	// Check day-of-month and month are wildcards
	if fields[2] != "*" || fields[3] != "*" {
		return false
	}

	// Check day-of-week is a specific day (0-6)
	dow := fields[4]
	for _, ch := range dow {
		if ch < '0' || ch > '6' {
			return false
		}
	}

	return true
}

// IsFuzzyCron checks if a cron expression is a fuzzy schedule placeholder
func IsFuzzyCron(cron string) bool {
	return strings.HasPrefix(cron, "FUZZY:")
}

// stableHash returns a deterministic hash value in the range [0, modulo)
// using FNV-1a hash algorithm, which is stable across platforms and Go versions.
func stableHash(s string, modulo int) int {
	h := fnv.New32a()
	// hash.Hash.Write never returns an error in practice, but check to satisfy gosec G104
	if _, err := h.Write([]byte(s)); err != nil {
		// Return 0 (safe fallback) if write somehow fails
		scheduleLog.Printf("Warning: hash write failed: %v", err)
		return 0
	}
	return int(h.Sum32() % uint32(modulo))
}

// ScatterSchedule takes a fuzzy cron expression and a workflow identifier
// and returns a deterministic scattered time for that workflow
func ScatterSchedule(fuzzyCron, workflowIdentifier string) (string, error) {
	if !IsFuzzyCron(fuzzyCron) {
		return "", fmt.Errorf("not a fuzzy schedule: %s", fuzzyCron)
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
		minute := scatteredMinutes % 60

		// Return scattered daily cron: minute hour * * *
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
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
		minute := scatteredMinutes % 60

		// Return scattered daily cron: minute hour * * *
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
	}

	// For FUZZY:DAILY * * *, we scatter across 24 hours
	if strings.HasPrefix(fuzzyCron, "FUZZY:DAILY") {
		// Use a stable hash of the workflow identifier to get a deterministic time
		hash := stableHash(workflowIdentifier, 1440) // Total minutes in a day

		hour := hash / 60
		minute := hash % 60

		// Return scattered daily cron: minute hour * * *
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
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

		// Use a stable hash to get a deterministic minute offset (0-59)
		minute := stableHash(workflowIdentifier, 60)

		// Return scattered hourly cron: minute */N * * *
		return fmt.Sprintf("%d */%d * * *", minute, interval), nil
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
		minute := scatteredMinutes % 60

		// Return scattered weekly cron: minute hour * * DOW
		return fmt.Sprintf("%d %d * * %s", minute, hour, weekday), nil
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

		// Use a stable hash of the workflow identifier to get a deterministic time
		hash := stableHash(workflowIdentifier, 1440) // Total minutes in a day

		hour := hash / 60
		minute := hash % 60

		// Return scattered weekly cron: minute hour * * DOW
		return fmt.Sprintf("%d %d * * %s", minute, hour, weekday), nil
	}

	// For FUZZY:WEEKLY * * *, we scatter across all weekdays and times
	if strings.HasPrefix(fuzzyCron, "FUZZY:WEEKLY") {
		// Use a stable hash of the workflow identifier to get a deterministic weekday and time
		// Total possibilities: 7 days * 1440 minutes = 10080 minutes in a week
		hash := stableHash(workflowIdentifier, 10080)

		// Extract weekday (0-6) and time within that day
		weekday := hash / 1440      // Which day of the week (0-6)
		minutesInDay := hash % 1440 // Which minute of that day (0-1439)
		hour := minutesInDay / 60
		minute := minutesInDay % 60

		// Return scattered weekly cron: minute hour * * DOW
		return fmt.Sprintf("%d %d * * %d", minute, hour, weekday), nil
	}

	// For FUZZY:BI_WEEKLY * * *, we scatter across 2 weeks (14 days)
	if strings.HasPrefix(fuzzyCron, "FUZZY:BI_WEEKLY") {
		// Use a stable hash of the workflow identifier to get a deterministic day and time
		// Total possibilities: 14 days * 1440 minutes = 20160 minutes in 2 weeks
		hash := stableHash(workflowIdentifier, 20160)

		// Extract time within a day (scatter across 2 weeks)
		minutesInDay := hash % 1440 // Which minute of that day (0-1439)
		hour := minutesInDay / 60
		minute := minutesInDay % 60

		// Convert to cron: We use day-of-month pattern with 14-day interval
		// Schedule every 14 days at the scattered time
		return fmt.Sprintf("%d %d */%d * *", minute, hour, 14), nil
	}

	// For FUZZY:TRI_WEEKLY * * *, we scatter across 3 weeks (21 days)
	if strings.HasPrefix(fuzzyCron, "FUZZY:TRI_WEEKLY") {
		// Use a stable hash of the workflow identifier to get a deterministic day and time
		// Total possibilities: 21 days * 1440 minutes = 30240 minutes in 3 weeks
		hash := stableHash(workflowIdentifier, 30240)

		// Extract time within a day (scatter across 3 weeks)
		minutesInDay := hash % 1440 // Which minute of that day (0-1439)
		hour := minutesInDay / 60
		minute := minutesInDay % 60

		// Convert to cron: We use day-of-month pattern with 21-day interval
		// Schedule every 21 days at the scattered time
		return fmt.Sprintf("%d %d */%d * *", minute, hour, 21), nil
	}

	return "", fmt.Errorf("unsupported fuzzy schedule type: %s", fuzzyCron)
}

// IsCronExpression checks if the input looks like a valid cron expression
// A valid cron expression has exactly 5 fields (minute, hour, day of month, month, day of week)
func IsCronExpression(input string) bool {
	// A cron expression has exactly 5 fields
	fields := strings.Fields(input)
	if len(fields) != 5 {
		return false
	}

	// Each field should match cron syntax (numbers, *, /, -, ,)
	cronFieldPattern := regexp.MustCompile(`^[\d\*\-/,]+$`)
	for _, field := range fields {
		if !cronFieldPattern.MatchString(field) {
			return false
		}
	}

	return true
}

// tokenize breaks the input into tokens
func (p *ScheduleParser) tokenize() error {
	// Normalize the input
	input := strings.ToLower(strings.TrimSpace(p.input))

	// Split on whitespace
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return fmt.Errorf("empty schedule expression")
	}

	p.tokens = tokens
	p.pos = 0
	return nil
}

// parse parses the tokens into a cron expression
func (p *ScheduleParser) parse() (string, error) {
	if len(p.tokens) == 0 {
		return "", fmt.Errorf("no tokens to parse")
	}

	// Check for interval-based schedules: "every N minutes|hours"
	if p.tokens[0] == "every" {
		return p.parseInterval()
	}

	// Otherwise, parse as base schedule (daily, weekly, monthly, yearly)
	return p.parseBase()
}

// parseInterval parses interval-based schedules like "every 10 minutes" or "every 2h"
func (p *ScheduleParser) parseInterval() (string, error) {
	if len(p.tokens) < 2 {
		return "", fmt.Errorf("invalid interval format, expected 'every N unit' or 'every Nunit'")
	}

	// Check if the second token is a duration format like "2h", "30m", "1d"
	if len(p.tokens) == 2 || (len(p.tokens) > 2 && p.tokens[2] != "minutes" && p.tokens[2] != "hours" && p.tokens[2] != "minute" && p.tokens[2] != "hour") {
		// Try to parse as short duration format: "every 2h", "every 30m", "every 1d"
		durationStr := p.tokens[1]

		// Check if it matches the pattern: number followed by unit letter (h, m, d, w, mo)
		durationPattern := regexp.MustCompile(`^(\d+)([hdwm]|mo)$`)
		matches := durationPattern.FindStringSubmatch(durationStr)

		if matches != nil {
			interval, _ := strconv.Atoi(matches[1])
			unit := matches[2]

			// Check for conflicting "at time" clause
			if len(p.tokens) > 2 {
				for i := 2; i < len(p.tokens); i++ {
					if p.tokens[i] == "at" {
						return "", fmt.Errorf("interval schedules cannot have 'at time' clause")
					}
				}
			}

			// Validate minimum duration of 5 minutes
			totalMinutes := 0
			switch unit {
			case "m":
				totalMinutes = interval
			case "h":
				totalMinutes = interval * 60
			case "d":
				totalMinutes = interval * 24 * 60
			case "w":
				totalMinutes = interval * 7 * 24 * 60
			case "mo":
				totalMinutes = interval * 30 * 24 * 60 // Approximate month as 30 days
			}

			if totalMinutes < 5 {
				return "", fmt.Errorf("minimum schedule interval is 5 minutes, got %d minute(s)", totalMinutes)
			}

			switch unit {
			case "m":
				// every Nm -> */N * * * * (minute intervals don't need scattering)
				return fmt.Sprintf("*/%d * * * *", interval), nil
			case "h":
				// every Nh -> FUZZY:HOURLY/N (fuzzy hourly interval with scattering)
				return fmt.Sprintf("FUZZY:HOURLY/%d * * *", interval), nil
			case "d":
				// every Nd -> daily at midnight, repeated N times
				// For single day, use daily. For multiple days, use interval in hours
				if interval == 1 {
					return "0 0 * * *", nil // daily
				}
				// Convert days to hours for cron expression
				return fmt.Sprintf("0 0 */%d * *", interval), nil
			case "w":
				// every Nw -> weekly interval
				// For single week, use weekly on sunday. For multiple weeks, convert to days
				if interval == 1 {
					return "0 0 * * 0", nil // weekly on sunday
				}
				// Convert weeks to days for cron expression
				days := interval * 7
				return fmt.Sprintf("0 0 */%d * *", days), nil
			case "mo":
				// every Nmo -> monthly interval
				// Cron doesn't support every N months directly, use day of month pattern
				if interval == 1 {
					return "0 0 1 * *", nil // first day of every month
				}
				// For multiple months, use month interval
				return fmt.Sprintf("0 0 1 */%d *", interval), nil
			default:
				return "", fmt.Errorf("unsupported duration unit '%s'", unit)
			}
		}
	}

	// Fall back to original parsing for "every N minutes" format
	if len(p.tokens) < 3 {
		return "", fmt.Errorf("invalid interval format, expected 'every N unit' or 'every Nunit' (e.g., 'every 2h')")
	}

	// Parse the interval number
	intervalStr := p.tokens[1]
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		return "", fmt.Errorf("invalid interval '%s', must be a positive integer", intervalStr)
	}

	// Parse the unit
	unit := p.tokens[2]
	if !strings.HasSuffix(unit, "s") {
		unit += "s" // Normalize to plural (minute -> minutes)
	}

	// Check for conflicting "at time" clause
	if len(p.tokens) > 3 {
		// Look for "at" keyword
		for i := 3; i < len(p.tokens); i++ {
			if p.tokens[i] == "at" {
				return "", fmt.Errorf("interval schedules cannot have 'at time' clause")
			}
		}
	}

	// Validate unit before checking minimum duration
	if unit != "minutes" && unit != "hours" && unit != "days" {
		return "", fmt.Errorf("unsupported interval unit '%s', use 'minutes', 'hours', or 'days'", unit)
	}

	// Validate minimum duration of 5 minutes
	totalMinutes := 0
	switch unit {
	case "minutes":
		totalMinutes = interval
	case "hours":
		totalMinutes = interval * 60
	case "days":
		totalMinutes = interval * 24 * 60
	}

	if totalMinutes < 5 {
		return "", fmt.Errorf("minimum schedule interval is 5 minutes, got %d minute(s)", totalMinutes)
	}

	switch unit {
	case "minutes":
		// every N minutes -> */N * * * * (minute intervals don't need scattering)
		return fmt.Sprintf("*/%d * * * *", interval), nil
	case "hours":
		// every N hours -> FUZZY:HOURLY/N (fuzzy hourly interval with scattering)
		return fmt.Sprintf("FUZZY:HOURLY/%d * * *", interval), nil
	case "days":
		// every N days -> daily at midnight, repeated N times
		// For single day, use daily. For multiple days, use interval in days
		if interval == 1 {
			return "0 0 * * *", nil // daily
		}
		// Convert days to day-of-month interval for cron expression
		return fmt.Sprintf("0 0 */%d * *", interval), nil
	default:
		return "", fmt.Errorf("unsupported interval unit '%s', use 'minutes', 'hours', or 'days'", unit)
	}
}

// parseBase parses base schedules like "daily", "weekly on monday", etc.
func (p *ScheduleParser) parseBase() (string, error) {
	if len(p.tokens) == 0 {
		return "", fmt.Errorf("empty schedule")
	}

	baseType := p.tokens[0]
	var minute, hour, day, month, weekday string

	// Default time is 00:00
	minute = "0"
	hour = "0"
	day = "*"
	month = "*"
	weekday = "*"

	switch baseType {
	case "daily":
		// daily -> FUZZY:DAILY (fuzzy schedule, time will be scattered)
		// daily at HH:MM -> MM HH * * *
		// daily around HH:MM -> FUZZY:DAILY_AROUND:HH:MM (fuzzy schedule with target time)
		// daily between HH:MM and HH:MM -> FUZZY:DAILY_BETWEEN:START_H:START_M:END_H:END_M (fuzzy schedule within time range)
		if len(p.tokens) == 1 {
			// Just "daily" with no time - this is a fuzzy schedule
			return "FUZZY:DAILY * * *", nil
		}
		if len(p.tokens) > 1 {
			// Check if "between" keyword is used
			if p.tokens[1] == "between" {
				// Parse: "daily between START and END"
				// We need at least: daily between TIME and TIME (5 tokens minimum)
				if len(p.tokens) < 5 {
					return "", fmt.Errorf("invalid 'between' format, expected 'daily between START and END'")
				}

				// Find the "and" keyword to split start and end times
				andIndex := -1
				for i := 2; i < len(p.tokens); i++ {
					if p.tokens[i] == "and" {
						andIndex = i
						break
					}
				}
				if andIndex == -1 {
					return "", fmt.Errorf("missing 'and' keyword in 'between' clause")
				}

				// Extract start time (tokens between "between" and "and")
				startTimeStr, err := p.extractTimeBetween(2, andIndex)
				if err != nil {
					return "", fmt.Errorf("invalid start time in 'between' clause: %w", err)
				}
				startMinute, startHour := parseTime(startTimeStr)

				// Extract end time (tokens after "and")
				endTimeStr, err := p.extractTimeAfter(andIndex + 1)
				if err != nil {
					return "", fmt.Errorf("invalid end time in 'between' clause: %w", err)
				}
				endMinute, endHour := parseTime(endTimeStr)

				// Validate that start is before end (in minutes since midnight)
				startMinutes := parseTimeToMinutes(startHour, startMinute)
				endMinutes := parseTimeToMinutes(endHour, endMinute)

				// Allow ranges that cross midnight (e.g., 22:00 to 02:00)
				// We'll handle this in the scattering logic
				if startMinutes == endMinutes {
					return "", fmt.Errorf("start and end times cannot be the same in 'between' clause")
				}

				// Return fuzzy between format: FUZZY:DAILY_BETWEEN:START_H:START_M:END_H:END_M
				return fmt.Sprintf("FUZZY:DAILY_BETWEEN:%s:%s:%s:%s * * *", startHour, startMinute, endHour, endMinute), nil
			}
			// Check if "around" keyword is used
			if p.tokens[1] == "around" {
				// Extract time after "around"
				timeStr, err := p.extractTime(2)
				if err != nil {
					return "", err
				}
				// Parse the time to validate it
				minute, hour = parseTime(timeStr)
				// Return fuzzy around format: FUZZY:DAILY_AROUND:HH:MM
				return fmt.Sprintf("FUZZY:DAILY_AROUND:%s:%s * * *", hour, minute), nil
			}
			// Reject "daily at TIME" pattern - use cron directly for fixed times
			return "", fmt.Errorf("'daily at <time>' syntax is not supported. Use fuzzy schedules like 'daily' (scattered), 'daily around <time>', or 'daily between <start> and <end>' for load distribution. For fixed times, use standard cron syntax (e.g., '0 14 * * *')")
		}

	case "hourly":
		// hourly -> FUZZY:HOURLY/1 (fuzzy hourly schedule, equivalent to "every 1h")
		if len(p.tokens) == 1 {
			return "FUZZY:HOURLY/1 * * *", nil
		}
		// hourly doesn't support time specifications
		return "", fmt.Errorf("hourly schedule does not support 'at time' clause, use 'hourly' without additional parameters")

	case "weekly":
		// weekly -> FUZZY:WEEKLY (fuzzy schedule, day and time will be scattered)
		// weekly on <weekday> -> FUZZY:WEEKLY:DOW (fuzzy schedule on specific weekday)
		// weekly on <weekday> at HH:MM -> MM HH * * DOW
		// weekly on <weekday> around HH:MM -> FUZZY:WEEKLY_AROUND:DOW:HH:MM
		if len(p.tokens) == 1 {
			// Just "weekly" with no day specified - this is a fuzzy schedule
			return "FUZZY:WEEKLY * * *", nil
		}

		if len(p.tokens) < 3 || p.tokens[1] != "on" {
			return "", fmt.Errorf("weekly schedule requires 'on <weekday>' or use 'weekly' alone for fuzzy schedule")
		}

		weekdayStr := p.tokens[2]
		weekday = mapWeekday(weekdayStr)
		if weekday == "" {
			return "", fmt.Errorf("invalid weekday '%s'", weekdayStr)
		}

		if len(p.tokens) > 3 {
			// Check if "around" keyword is used
			if p.tokens[3] == "around" {
				// Extract time after "around"
				timeStr, err := p.extractTime(4)
				if err != nil {
					return "", err
				}
				// Parse the time to validate it
				minute, hour = parseTime(timeStr)
				// Return fuzzy around format: FUZZY:WEEKLY_AROUND:DOW:HH:MM
				return fmt.Sprintf("FUZZY:WEEKLY_AROUND:%s:%s:%s * * *", weekday, hour, minute), nil
			}
			// Reject "weekly on <weekday> at TIME" pattern - use cron directly for fixed times
			return "", fmt.Errorf("'weekly on <weekday> at <time>' syntax is not supported. Use fuzzy schedules like 'weekly on %s' (scattered), 'weekly on %s around <time>', or standard cron syntax (e.g., '30 6 * * %s')", weekdayStr, weekdayStr, weekday)
		} else {
			// weekly on <weekday> with no time - this is a fuzzy schedule
			return fmt.Sprintf("FUZZY:WEEKLY:%s * * *", weekday), nil
		}

	case "bi-weekly":
		// bi-weekly -> FUZZY:BI_WEEKLY (fuzzy schedule, scattered across 2 weeks)
		if len(p.tokens) == 1 {
			// Just "bi-weekly" with no additional parameters - scatter across 2 weeks
			return "FUZZY:BI_WEEKLY * * *", nil
		}
		return "", fmt.Errorf("bi-weekly schedule does not support additional parameters, use 'bi-weekly' alone for fuzzy schedule")

	case "tri-weekly":
		// tri-weekly -> FUZZY:TRI_WEEKLY (fuzzy schedule, scattered across 3 weeks)
		if len(p.tokens) == 1 {
			// Just "tri-weekly" with no additional parameters - scatter across 3 weeks
			return "FUZZY:TRI_WEEKLY * * *", nil
		}
		return "", fmt.Errorf("tri-weekly schedule does not support additional parameters, use 'tri-weekly' alone for fuzzy schedule")

	case "monthly":
		// monthly on <day> -> rejected (use cron directly)
		// monthly on <day> at HH:MM -> rejected (use cron directly)
		if len(p.tokens) < 3 || p.tokens[1] != "on" {
			return "", fmt.Errorf("monthly schedule requires 'on <day>'")
		}

		dayNum, err := strconv.Atoi(p.tokens[2])
		if err != nil || dayNum < 1 || dayNum > 31 {
			return "", fmt.Errorf("invalid day of month '%s', must be 1-31", p.tokens[2])
		}
		day = p.tokens[2]

		// Reject monthly schedules - they always generate fixed times
		// monthly on 15 -> 0 0 15 * * (midnight on 15th)
		// monthly on 15 at 09:00 -> 0 9 15 * * (9am on 15th)
		if len(p.tokens) > 3 {
			return "", fmt.Errorf("'monthly on <day> at <time>' syntax is not supported. Use standard cron syntax for monthly schedules (e.g., '0 9 %s * *' for the %sth at 9am)", day, day)
		}
		return "", fmt.Errorf("'monthly on <day>' syntax is not supported. Use standard cron syntax for monthly schedules (e.g., '0 0 %s * *' for the %sth at midnight)", day, day)

	default:
		return "", fmt.Errorf("unsupported schedule type '%s', use 'daily', 'weekly', 'bi-weekly', 'tri-weekly', or 'monthly'", baseType)
	}

	// Build cron expression: MIN HOUR DOM MONTH DOW
	return fmt.Sprintf("%s %s %s %s %s", minute, hour, day, month, weekday), nil
}

// extractTime extracts the time specification from tokens starting at startPos
// Returns the time string (HH:MM, midnight, or noon) with optional UTC offset
func (p *ScheduleParser) extractTime(startPos int) (string, error) {
	if startPos >= len(p.tokens) {
		return "", fmt.Errorf("expected time specification")
	}

	// Check for "at" keyword
	if p.tokens[startPos] == "at" {
		startPos++
		if startPos >= len(p.tokens) {
			return "", fmt.Errorf("expected time after 'at'")
		}
	}

	timeStr := p.tokens[startPos]

	// Check if there's a UTC offset in the next token
	if startPos+1 < len(p.tokens) {
		nextToken := strings.ToLower(p.tokens[startPos+1])
		if strings.HasPrefix(nextToken, "utc") {
			// Combine time and UTC offset
			timeStr = timeStr + " " + p.tokens[startPos+1]
		}
	}

	return timeStr, nil
}

// extractTimeBetween extracts a time specification from tokens between startPos and endPos (exclusive)
// Used for parsing the start time in "between START and END" clauses
func (p *ScheduleParser) extractTimeBetween(startPos, endPos int) (string, error) {
	if startPos >= len(p.tokens) || startPos >= endPos {
		return "", fmt.Errorf("expected time specification")
	}

	// The time is in the tokens between startPos and endPos
	// It might be a single token (e.g., "9am") or multiple tokens (e.g., "14:00 utc+9")
	timeTokens := []string{}
	for i := startPos; i < endPos && i < len(p.tokens); i++ {
		timeTokens = append(timeTokens, p.tokens[i])
	}

	if len(timeTokens) == 0 {
		return "", fmt.Errorf("expected time specification")
	}

	// Check if there's a UTC offset
	if len(timeTokens) >= 2 && strings.HasPrefix(strings.ToLower(timeTokens[1]), "utc") {
		return timeTokens[0] + " " + timeTokens[1], nil
	}

	return timeTokens[0], nil
}

// extractTimeAfter extracts a time specification from tokens starting at startPos until the end
// Used for parsing the end time in "between START and END" clauses
func (p *ScheduleParser) extractTimeAfter(startPos int) (string, error) {
	if startPos >= len(p.tokens) {
		return "", fmt.Errorf("expected time specification")
	}

	// Collect remaining tokens (time and optional UTC offset)
	timeStr := p.tokens[startPos]

	// Check if there's a UTC offset in the next token
	if startPos+1 < len(p.tokens) {
		nextToken := strings.ToLower(p.tokens[startPos+1])
		if strings.HasPrefix(nextToken, "utc") {
			// Combine time and UTC offset
			timeStr = timeStr + " " + p.tokens[startPos+1]
		}
	}

	return timeStr, nil
}

// parseTimeToMinutes converts hour and minute strings to total minutes since midnight
func parseTimeToMinutes(hourStr, minuteStr string) int {
	hour, _ := strconv.Atoi(hourStr)
	minute, _ := strconv.Atoi(minuteStr)
	return hour*60 + minute
}

// parseTime converts a time string to minute and hour, with optional UTC offset
// Supports formats: HH:MM, midnight, noon, 3pm, 1am, HH:MM utc+N, HH:MM utc+HH:MM, HH:MM utc-N, 3pm utc+9
func parseTime(timeStr string) (minute string, hour string) {
	// Check for UTC offset
	parts := strings.Split(timeStr, " ")
	var utcOffset int
	var baseTime string

	if len(parts) == 2 && strings.HasPrefix(strings.ToLower(parts[1]), "utc") {
		baseTime = parts[0]
		offsetStr := strings.ToLower(parts[1])

		// Parse UTC offset (e.g., utc+9, utc-5, utc+09:00, utc-05:30)
		if len(offsetStr) > 3 {
			offsetPart := offsetStr[3:] // Skip "utc"
			sign := 1
			if strings.HasPrefix(offsetPart, "+") {
				offsetPart = offsetPart[1:]
			} else if strings.HasPrefix(offsetPart, "-") {
				sign = -1
				offsetPart = offsetPart[1:]
			}

			// Check if it's HH:MM format
			if strings.Contains(offsetPart, ":") {
				offsetParts := strings.Split(offsetPart, ":")
				if len(offsetParts) == 2 {
					hours, err1 := strconv.Atoi(offsetParts[0])
					mins, err2 := strconv.Atoi(offsetParts[1])
					if err1 == nil && err2 == nil {
						utcOffset = sign * (hours*60 + mins)
					}
				}
			} else {
				// Just hours (e.g., utc+9)
				hours, err := strconv.Atoi(offsetPart)
				if err == nil {
					utcOffset = sign * hours * 60
				}
			}
		}
	} else {
		baseTime = timeStr
	}

	var baseMinute, baseHour int

	switch baseTime {
	case "midnight":
		baseMinute, baseHour = 0, 0
	case "noon":
		baseMinute, baseHour = 0, 12
	default:
		// Check for am/pm format (e.g., "3pm", "11am")
		lowerTime := strings.ToLower(baseTime)
		if strings.HasSuffix(lowerTime, "am") || strings.HasSuffix(lowerTime, "pm") {
			isPM := strings.HasSuffix(lowerTime, "pm")
			// Remove am/pm suffix
			hourStr := lowerTime[:len(lowerTime)-2]

			hourNum, err := strconv.Atoi(hourStr)
			if err == nil && hourNum >= 1 && hourNum <= 12 {
				// Convert 12-hour to 24-hour format
				if isPM {
					if hourNum != 12 {
						hourNum += 12
					}
				} else { // AM
					if hourNum == 12 {
						hourNum = 0
					}
				}
				baseMinute, baseHour = 0, hourNum
			} else {
				// Invalid format, return defaults
				return "0", "0"
			}
		} else {
			// Parse HH:MM format
			timeParts := strings.Split(baseTime, ":")
			if len(timeParts) == 2 {
				// Validate hour
				hourNum, err := strconv.Atoi(timeParts[0])
				if err == nil && hourNum >= 0 && hourNum <= 23 {
					// Validate minute
					minNum, err := strconv.Atoi(timeParts[1])
					if err == nil && minNum >= 0 && minNum <= 59 {
						baseMinute, baseHour = minNum, hourNum
					} else {
						// Invalid format, return defaults
						return "0", "0"
					}
				} else {
					// Invalid format, return defaults
					return "0", "0"
				}
			} else {
				// Invalid format, return defaults
				return "0", "0"
			}
		}
	}

	// Apply UTC offset (convert from local time to UTC)
	// If utc+9, we subtract 9 hours to get UTC time
	totalMinutes := baseHour*60 + baseMinute - utcOffset

	// Handle wrap-around (keep within 0-1439 minutes, which is 0:00-23:59)
	for totalMinutes < 0 {
		totalMinutes += 24 * 60
	}
	for totalMinutes >= 24*60 {
		totalMinutes -= 24 * 60
	}

	finalHour := totalMinutes / 60
	finalMinute := totalMinutes % 60

	return strconv.Itoa(finalMinute), strconv.Itoa(finalHour)
}

// mapWeekday maps day names to cron day-of-week numbers (0=Sunday, 6=Saturday)
func mapWeekday(day string) string {
	day = strings.ToLower(day)
	weekdays := map[string]string{
		"sunday":    "0",
		"sun":       "0",
		"monday":    "1",
		"mon":       "1",
		"tuesday":   "2",
		"tue":       "2",
		"wednesday": "3",
		"wed":       "3",
		"thursday":  "4",
		"thu":       "4",
		"friday":    "5",
		"fri":       "5",
		"saturday":  "6",
		"sat":       "6",
	}
	return weekdays[day]
}
