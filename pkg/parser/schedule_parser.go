package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
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

	// Check if "on weekdays" suffix is present at the end
	hasWeekdaysSuffix := p.hasWeekdaysSuffix()

	// Check if the second token is a duration format like "2h", "30m", "1d"
	if len(p.tokens) == 2 || (len(p.tokens) == 4 && hasWeekdaysSuffix) || (len(p.tokens) > 2 && !hasWeekdaysSuffix && p.tokens[2] != "minutes" && p.tokens[2] != "hours" && p.tokens[2] != "minute" && p.tokens[2] != "hour") {
		// Try to parse as short duration format: "every 2h", "every 30m", "every 1d"
		durationStr := p.tokens[1]

		// Check if it matches the pattern: number followed by unit letter (h, m, d, w, mo)
		durationPattern := regexp.MustCompile(`^(\d+)([hdwm]|mo)$`)
		matches := durationPattern.FindStringSubmatch(durationStr)

		if matches != nil {
			interval, _ := strconv.Atoi(matches[1])
			unit := matches[2]

			// Check for conflicting "at time" clause (but allow "on weekdays")
			endPos := len(p.tokens)
			if hasWeekdaysSuffix {
				endPos -= 2
			}
			if len(p.tokens) > 2 {
				for i := 2; i < endPos; i++ {
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
				// Minute intervals with weekdays not supported (would run every N minutes only on weekdays)
				if hasWeekdaysSuffix {
					return "", fmt.Errorf("minute intervals with 'on weekdays' are not supported")
				}
				return fmt.Sprintf("*/%d * * * *", interval), nil
			case "h":
				// every Nh -> FUZZY:HOURLY/N or FUZZY:HOURLY_WEEKDAYS/N (fuzzy hourly interval with scattering)
				if hasWeekdaysSuffix {
					return fmt.Sprintf("FUZZY:HOURLY_WEEKDAYS/%d * * *", interval), nil
				}
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
	minTokens := 3
	if hasWeekdaysSuffix {
		minTokens = 5
	}
	if len(p.tokens) < minTokens {
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

	// Check for conflicting "at time" clause (but allow "on weekdays")
	endPos := len(p.tokens)
	if hasWeekdaysSuffix {
		endPos -= 2
	}
	if len(p.tokens) > 3 {
		// Look for "at" keyword
		for i := 3; i < endPos; i++ {
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
		// Minute intervals with weekdays not supported
		if hasWeekdaysSuffix {
			return "", fmt.Errorf("minute intervals with 'on weekdays' are not supported")
		}
		return fmt.Sprintf("*/%d * * * *", interval), nil
	case "hours":
		// every N hours -> FUZZY:HOURLY/N or FUZZY:HOURLY_WEEKDAYS/N (fuzzy hourly interval with scattering)
		if hasWeekdaysSuffix {
			return fmt.Sprintf("FUZZY:HOURLY_WEEKDAYS/%d * * *", interval), nil
		}
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

	// Check if "on weekdays" suffix is present at the end
	hasWeekdaysSuffix := p.hasWeekdaysSuffix()

	switch baseType {
	case "daily":
		// daily -> FUZZY:DAILY (fuzzy schedule, time will be scattered)
		// daily on weekdays -> FUZZY:DAILY_WEEKDAYS (fuzzy schedule, Mon-Fri only)
		// daily at HH:MM -> MM HH * * *
		// daily around HH:MM -> FUZZY:DAILY_AROUND:HH:MM (fuzzy schedule with target time)
		// daily around HH:MM on weekdays -> FUZZY:DAILY_AROUND_WEEKDAYS:HH:MM
		// daily between HH:MM and HH:MM -> FUZZY:DAILY_BETWEEN:START_H:START_M:END_H:END_M (fuzzy schedule within time range)
		// daily between HH:MM and HH:MM on weekdays -> FUZZY:DAILY_BETWEEN_WEEKDAYS:START_H:START_M:END_H:END_M
		if len(p.tokens) == 1 || (len(p.tokens) == 3 && hasWeekdaysSuffix) {
			// Just "daily" or "daily on weekdays" with no time - this is a fuzzy schedule
			if hasWeekdaysSuffix {
				return "FUZZY:DAILY_WEEKDAYS * * *", nil
			}
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
				endPos := len(p.tokens)
				// If "on weekdays" suffix, exclude the last 2 tokens from search
				if hasWeekdaysSuffix {
					endPos -= 2
				}
				for i := 2; i < endPos; i++ {
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

				// Extract end time (tokens after "and", possibly excluding "on weekdays")
				endTimeStr, err := p.extractTimeAfter(andIndex+1, hasWeekdaysSuffix)
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

				// Return fuzzy between format with optional weekdays suffix
				if hasWeekdaysSuffix {
					return fmt.Sprintf("FUZZY:DAILY_BETWEEN_WEEKDAYS:%s:%s:%s:%s * * *", startHour, startMinute, endHour, endMinute), nil
				}
				return fmt.Sprintf("FUZZY:DAILY_BETWEEN:%s:%s:%s:%s * * *", startHour, startMinute, endHour, endMinute), nil
			}
			// Check if "around" keyword is used
			if p.tokens[1] == "around" {
				// Extract time after "around", possibly excluding "on weekdays"
				timeStr, err := p.extractTimeWithWeekdays(2, hasWeekdaysSuffix)
				if err != nil {
					return "", err
				}
				// Parse the time to validate it
				minute, hour = parseTime(timeStr)
				// Return fuzzy around format with optional weekdays suffix
				if hasWeekdaysSuffix {
					return fmt.Sprintf("FUZZY:DAILY_AROUND_WEEKDAYS:%s:%s * * *", hour, minute), nil
				}
				return fmt.Sprintf("FUZZY:DAILY_AROUND:%s:%s * * *", hour, minute), nil
			}
			// Reject "daily at TIME" pattern - use cron directly for fixed times
			return "", fmt.Errorf("'daily at <time>' syntax is not supported. Use fuzzy schedules like 'daily' (scattered), 'daily around <time>', or 'daily between <start> and <end>' for load distribution. For fixed times, use standard cron syntax (e.g., '0 14 * * *')")
		}

	case "hourly":
		// hourly -> FUZZY:HOURLY/1 (fuzzy hourly schedule, equivalent to "every 1h")
		// hourly on weekdays -> FUZZY:HOURLY_WEEKDAYS/1 (fuzzy hourly schedule, Mon-Fri only)
		if len(p.tokens) == 1 || (len(p.tokens) == 3 && hasWeekdaysSuffix) {
			if hasWeekdaysSuffix {
				return "FUZZY:HOURLY_WEEKDAYS/1 * * *", nil
			}
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

	timeTokens := []string{p.tokens[startPos]}
	nextIndex := startPos + 1
	if nextIndex < len(p.tokens) && isAMPMToken(p.tokens[nextIndex]) {
		timeTokens = append(timeTokens, p.tokens[nextIndex])
		nextIndex++
	}
	if nextIndex < len(p.tokens) {
		timezoneToken := strings.ToLower(p.tokens[nextIndex])
		if strings.HasPrefix(timezoneToken, "utc") {
			timeTokens = append(timeTokens, timezoneToken)
		} else if normalized, ok := normalizeTimezoneAbbreviation(timezoneToken); ok {
			timeTokens = append(timeTokens, normalized)
		}
	}

	return normalizeTimeTokens(timeTokens), nil
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

	return normalizeTimeTokens(timeTokens), nil
}

// extractTimeAfter extracts a time specification from tokens starting at startPos until the end
// Used for parsing the end time in "between START and END" clauses
// If hasWeekdaysSuffix is true, excludes the last 2 tokens ("on weekdays")
func (p *ScheduleParser) extractTimeAfter(startPos int, hasWeekdaysSuffix bool) (string, error) {
	if startPos >= len(p.tokens) {
		return "", fmt.Errorf("expected time specification")
	}

	endPos := len(p.tokens)
	if hasWeekdaysSuffix {
		endPos -= 2
	}

	if startPos >= endPos {
		return "", fmt.Errorf("expected time specification")
	}

	// Collect tokens until endPos (time and optional UTC offset)
	timeStr := p.tokens[startPos]

	timeTokens := []string{timeStr}
	nextIndex := startPos + 1
	if nextIndex < endPos && isAMPMToken(p.tokens[nextIndex]) {
		timeTokens = append(timeTokens, p.tokens[nextIndex])
		nextIndex++
	}
	if nextIndex < endPos {
		timezoneToken := strings.ToLower(p.tokens[nextIndex])
		if strings.HasPrefix(timezoneToken, "utc") {
			timeTokens = append(timeTokens, timezoneToken)
		} else if normalized, ok := normalizeTimezoneAbbreviation(timezoneToken); ok {
			timeTokens = append(timeTokens, normalized)
		}
	}

	return normalizeTimeTokens(timeTokens), nil
}

// hasWeekdaysSuffix checks if "on weekdays" is present at the end of tokens
func (p *ScheduleParser) hasWeekdaysSuffix() bool {
	if len(p.tokens) < 2 {
		return false
	}
	// Check if the last two tokens are "on" and "weekdays"
	return p.tokens[len(p.tokens)-2] == "on" && p.tokens[len(p.tokens)-1] == "weekdays"
}

// extractTimeWithWeekdays extracts time specification, handling optional "on weekdays" suffix
func (p *ScheduleParser) extractTimeWithWeekdays(startPos int, hasWeekdaysSuffix bool) (string, error) {
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

	endPos := len(p.tokens)
	if hasWeekdaysSuffix {
		endPos -= 2
	}

	timeTokens := []string{p.tokens[startPos]}
	nextIndex := startPos + 1
	if nextIndex < endPos && isAMPMToken(p.tokens[nextIndex]) {
		timeTokens = append(timeTokens, p.tokens[nextIndex])
		nextIndex++
	}
	if nextIndex < endPos {
		timezoneToken := strings.ToLower(p.tokens[nextIndex])
		if strings.HasPrefix(timezoneToken, "utc") {
			timeTokens = append(timeTokens, timezoneToken)
		} else if normalized, ok := normalizeTimezoneAbbreviation(timezoneToken); ok {
			timeTokens = append(timeTokens, normalized)
		}
	}

	return normalizeTimeTokens(timeTokens), nil
}
