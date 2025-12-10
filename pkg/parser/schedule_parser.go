package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ScheduleParser parses human-friendly schedule expressions into cron expressions
type ScheduleParser struct {
	input  string
	tokens []string
	pos    int
}

// ParseSchedule converts a human-friendly schedule expression into a cron expression
// Returns the cron expression and the original friendly format for comments
func ParseSchedule(input string) (cron string, original string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("schedule expression cannot be empty")
	}

	// If it's already a cron expression (5 fields separated by spaces), return as-is
	if isCronExpression(input) {
		return input, "", nil
	}

	parser := &ScheduleParser{
		input: input,
	}

	// Tokenize the input
	if err := parser.tokenize(); err != nil {
		return "", "", err
	}

	// Parse the tokens
	cronExpr, err := parser.parse()
	if err != nil {
		return "", "", err
	}

	return cronExpr, input, nil
}

// isCronExpression checks if the input looks like a cron expression
func isCronExpression(input string) bool {
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
				// every Nm -> */N * * * *
				return fmt.Sprintf("*/%d * * * *", interval), nil
			case "h":
				// every Nh -> 0 */N * * *
				return fmt.Sprintf("0 */%d * * *", interval), nil
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
	if unit != "minutes" && unit != "hours" {
		return "", fmt.Errorf("unsupported interval unit '%s', use 'minutes' or 'hours'", unit)
	}

	// Validate minimum duration of 5 minutes
	totalMinutes := 0
	switch unit {
	case "minutes":
		totalMinutes = interval
	case "hours":
		totalMinutes = interval * 60
	}
	
	if totalMinutes < 5 {
		return "", fmt.Errorf("minimum schedule interval is 5 minutes, got %d minute(s)", totalMinutes)
	}

	switch unit {
	case "minutes":
		// every N minutes -> */N * * * *
		return fmt.Sprintf("*/%d * * * *", interval), nil
	case "hours":
		// every N hours -> 0 */N * * *
		return fmt.Sprintf("0 */%d * * *", interval), nil
	default:
		return "", fmt.Errorf("unsupported interval unit '%s', use 'minutes' or 'hours'", unit)
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
		// daily -> 0 0 * * *
		// daily at HH:MM -> MM HH * * *
		if len(p.tokens) > 1 {
			timeStr, err := p.extractTime(1)
			if err != nil {
				return "", err
			}
			minute, hour = parseTime(timeStr)
		}

	case "weekly":
		// weekly on <weekday> -> 0 0 * * DOW
		// weekly on <weekday> at HH:MM -> MM HH * * DOW
		if len(p.tokens) < 3 || p.tokens[1] != "on" {
			return "", fmt.Errorf("weekly schedule requires 'on <weekday>'")
		}

		weekdayStr := p.tokens[2]
		weekday = mapWeekday(weekdayStr)
		if weekday == "" {
			return "", fmt.Errorf("invalid weekday '%s'", weekdayStr)
		}

		if len(p.tokens) > 3 {
			timeStr, err := p.extractTime(3)
			if err != nil {
				return "", err
			}
			minute, hour = parseTime(timeStr)
		}

	case "monthly":
		// monthly on <day> -> 0 0 <day> * *
		// monthly on <day> at HH:MM -> MM HH <day> * *
		if len(p.tokens) < 3 || p.tokens[1] != "on" {
			return "", fmt.Errorf("monthly schedule requires 'on <day>'")
		}

		dayNum, err := strconv.Atoi(p.tokens[2])
		if err != nil || dayNum < 1 || dayNum > 31 {
			return "", fmt.Errorf("invalid day of month '%s', must be 1-31", p.tokens[2])
		}
		day = p.tokens[2]

		if len(p.tokens) > 3 {
			timeStr, err := p.extractTime(3)
			if err != nil {
				return "", err
			}
			minute, hour = parseTime(timeStr)
		}

	default:
		return "", fmt.Errorf("unsupported schedule type '%s', use 'daily', 'weekly', or 'monthly'", baseType)
	}

	// Build cron expression: MIN HOUR DOM MONTH DOW
	return fmt.Sprintf("%s %s %s %s %s", minute, hour, day, month, weekday), nil
}

// extractTime extracts the time specification from tokens starting at startPos
// Returns the time string (HH:MM, midnight, or noon)
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
	return timeStr, nil
}

// parseTime converts a time string to minute and hour
func parseTime(timeStr string) (minute string, hour string) {
	switch timeStr {
	case "midnight":
		return "0", "0"
	case "noon":
		return "0", "12"
	default:
		// Parse HH:MM format
		parts := strings.Split(timeStr, ":")
		if len(parts) == 2 {
			// Validate hour
			hourNum, err := strconv.Atoi(parts[0])
			if err == nil && hourNum >= 0 && hourNum <= 23 {
				// Validate minute
				minNum, err := strconv.Atoi(parts[1])
				if err == nil && minNum >= 0 && minNum <= 59 {
					// Return as strings without leading zeros
					return strconv.Itoa(minNum), strconv.Itoa(hourNum)
				}
			}
		}
		// Invalid format, return defaults
		return "0", "0"
	}
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
