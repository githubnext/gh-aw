package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var scheduleCodemodLog = logger.New("cli:codemod_schedule")

// getScheduleAtToAroundCodemod creates a codemod for converting "daily at TIME" to "daily around TIME"
func getScheduleAtToAroundCodemod() Codemod {
	return Codemod{
		ID:           "schedule-at-to-around-migration",
		Name:         "Migrate schedule 'at' syntax to 'around' syntax",
		Description:  "Converts deprecated 'daily at TIME', 'weekly on DAY at TIME', and 'monthly on N at TIME' to fuzzy schedules or standard cron",
		IntroducedIn: "0.5.0",
		Apply:        getScheduleAtToAroundCodemodApply,
	}
}

func getScheduleAtToAroundCodemodApply(content string, _ map[string]any) (string, bool, error) {
	return applyFrontmatterLineTransform(content, getScheduleAtToAroundCodemodTransform)
}

func getScheduleAtToAroundCodemodTransform(lines []string) ([]string, bool) {
	var modified bool
	result := make([]string, len(lines))

	for i, line := range lines {
		newLine, changed := getScheduleAtToAroundCodemodTransformLine(line, i)
		result[i] = newLine
		modified = modified || changed
	}

	if modified {
		scheduleCodemodLog.Print("Applied schedule 'at' to 'around' migration")
	}
	return result, modified
}

func getScheduleAtToAroundCodemodTransformLine(line string, lineIndex int) (string, bool) {
	trimmedLine := strings.TrimSpace(line)
	// Skip if not a cron or schedule line
	if !strings.Contains(trimmedLine, "cron:") && !strings.Contains(trimmedLine, "schedule:") {
		return line, false
	}

	leadingSpace, listMarker, fieldName, scheduleValue := getScheduleAtToAroundCodemodLineParts(line, trimmedLine)
	if scheduleValue == "" {
		return line, false
	}

	// Remove quotes if present
	scheduleValue = strings.Trim(scheduleValue, "\"'")
	if newLine, changed := getScheduleAtToAroundCodemodDailyLine(leadingSpace, listMarker, fieldName, scheduleValue, lineIndex); changed {
		return newLine, true
	}
	if newLine, changed := getScheduleAtToAroundCodemodWeeklyLine(leadingSpace, listMarker, fieldName, scheduleValue, lineIndex); changed {
		return newLine, true
	}
	if newLine, changed := getScheduleAtToAroundCodemodMonthlyLine(leadingSpace, listMarker, fieldName, scheduleValue, lineIndex); changed {
		return newLine, true
	}
	return line, false
}

func getScheduleAtToAroundCodemodLineParts(line, trimmedLine string) (string, string, string, string) {
	// Extract leading whitespace to preserve indentation
	leadingSpace := getIndentation(line)

	// Check if this is a list item (starts with - after whitespace)
	restAfterSpace := strings.TrimLeft(line, " \t")
	listMarker := ""
	if strings.HasPrefix(restAfterSpace, "-") {
		// This is a list item, preserve the dash
		listMarker = "- "
	}

	// Extract the schedule value (after "cron:" or "schedule:")
	if strings.Contains(trimmedLine, "cron:") {
		parts := strings.SplitN(trimmedLine, "cron:", 2)
		if len(parts) == 2 {
			return leadingSpace, listMarker, "cron", strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(trimmedLine, "schedule:") {
		parts := strings.SplitN(trimmedLine, "schedule:", 2)
		if len(parts) == 2 {
			return leadingSpace, listMarker, "schedule", strings.TrimSpace(parts[1])
		}
	}
	return leadingSpace, listMarker, "", ""
}

func getScheduleAtToAroundCodemodDailyLine(leadingSpace, listMarker, fieldName, scheduleValue string, lineIndex int) (string, bool) {
	// Pattern 1: daily at TIME (not "daily around" or "daily between")
	if !strings.HasPrefix(scheduleValue, "daily at") || strings.Contains(scheduleValue, "around") || strings.Contains(scheduleValue, "between") {
		return "", false
	}
	newSchedule := strings.Replace(scheduleValue, "daily at", "daily around", 1)
	scheduleCodemodLog.Printf("Converted 'daily at' to 'daily around' on line %d: %s -> %s", lineIndex+1, scheduleValue, newSchedule)
	return fmt.Sprintf("%s%s%s: %s", leadingSpace, listMarker, fieldName, newSchedule), true
}

func getScheduleAtToAroundCodemodWeeklyLine(leadingSpace, listMarker, fieldName, scheduleValue string, lineIndex int) (string, bool) {
	// Pattern 2: weekly on DAY at TIME
	if !strings.Contains(scheduleValue, "weekly on") || !strings.Contains(scheduleValue, " at ") || strings.Contains(scheduleValue, "around") {
		return "", false
	}
	newSchedule := strings.Replace(scheduleValue, " at ", " around ", 1)
	scheduleCodemodLog.Printf("Converted 'weekly on DAY at' to 'weekly on DAY around' on line %d: %s -> %s", lineIndex+1, scheduleValue, newSchedule)
	return fmt.Sprintf("%s%s%s: %s", leadingSpace, listMarker, fieldName, newSchedule), true
}

func getScheduleAtToAroundCodemodMonthlyLine(leadingSpace, listMarker, fieldName, scheduleValue string, lineIndex int) (string, bool) {
	// Pattern 3: monthly on N [at TIME] - convert to cron
	if !strings.HasPrefix(scheduleValue, "monthly on") {
		return "", false
	}
	day := getScheduleAtToAroundCodemodMonthlyDay(scheduleValue)
	if day == "" {
		return "", false
	}
	cronExpr := fmt.Sprintf("0 0 %s * *", day)
	if strings.Contains(scheduleValue, " at ") {
		// Has time - default to 09:00 as example since we can't parse arbitrary times in codemod
		// The user should manually adjust the hour/minute if needed
		cronExpr = fmt.Sprintf("0 9 %s * *", day)
	}

	// Replace with cron and add explanatory comment
	scheduleCodemodLog.Printf("Converted 'monthly on' to cron on line %d: %s -> %s", lineIndex+1, scheduleValue, cronExpr)
	return fmt.Sprintf("%s%s%s: \"%s\"  # Converted from '%s' (adjust time as needed)", leadingSpace, listMarker, fieldName, cronExpr, scheduleValue), true
}

func getScheduleAtToAroundCodemodMonthlyDay(scheduleValue string) string {
	// Extract day number
	monthlyParts := strings.Fields(scheduleValue)
	for idx, part := range monthlyParts {
		if part == "on" && idx+1 < len(monthlyParts) {
			return monthlyParts[idx+1]
		}
	}
	return ""
}
