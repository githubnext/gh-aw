package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var frontmatterEditorLog = logger.New("cli:frontmatter_editor")

// UpdateFieldInFrontmatter updates a field in the frontmatter while preserving the original formatting
// when possible. It tries to preserve whitespace, comments, and formatting by working with the raw
// frontmatter lines, similar to how addSourceToWorkflow works.
func UpdateFieldInFrontmatter(content, fieldName, fieldValue string) (string, error) {
	frontmatterEditorLog.Printf("Updating frontmatter field: %s = %s", fieldName, fieldValue)

	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		frontmatterEditorLog.Printf("Failed to parse frontmatter: %v", err)
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Try to preserve original frontmatter formatting by manually updating the field
	if len(result.FrontmatterLines) > 0 {
		return updateFieldInFrontmatterRaw(result, fieldName, fieldValue), nil
	}

	// Fallback to marshal-based approach if no raw lines are available
	return updateFieldInFrontmatterFallback(result, fieldName, fieldValue)
}

func updateFieldInFrontmatterRaw(result *parser.FrontmatterResult, fieldName, fieldValue string) string {
	frontmatterEditorLog.Printf("Using raw frontmatter lines for field update (%d lines)", len(result.FrontmatterLines))
	newFrontmatterLines := make([]string, 0, len(result.FrontmatterLines))
	fieldUpdated := false
	skipChildren := false
	fieldIndentLevel := 0

	for _, line := range result.FrontmatterLines {
		if skipChildren {
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent > fieldIndentLevel {
				continue
			}
			skipChildren = false
		}
		trimmedLine := strings.TrimSpace(line)
		if !fieldUpdated && strings.HasPrefix(trimmedLine, fieldName+":") {
			updatedLine, indent := updateFieldInFrontmatterLine(line, fieldName, fieldValue)
			newFrontmatterLines = append(newFrontmatterLines, updatedLine)
			fieldUpdated = true
			fieldIndentLevel = indent
			skipChildren = true
			frontmatterEditorLog.Printf("Updated existing field %s", fieldName)
			continue
		}
		newFrontmatterLines = append(newFrontmatterLines, line)
	}

	if !fieldUpdated {
		newFrontmatterLines = append(newFrontmatterLines, fmt.Sprintf("%s: %s", fieldName, fieldValue))
		frontmatterEditorLog.Printf("Added new field %s at end of frontmatter", fieldName)
	}
	return updateFieldInFrontmatterReconstruct(newFrontmatterLines, result.Markdown)
}

func updateFieldInFrontmatterLine(line, fieldName, fieldValue string) (string, int) {
	leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	commentIndex := strings.Index(line, "#")
	var comment string
	if commentIndex > strings.Index(line, ":") && commentIndex != -1 {
		comment = line[commentIndex:]
	}
	if comment != "" {
		return fmt.Sprintf("%s%s: %s %s", leadingSpace, fieldName, fieldValue, comment), len(leadingSpace)
	}
	return fmt.Sprintf("%s%s: %s", leadingSpace, fieldName, fieldValue), len(leadingSpace)
}

func updateFieldInFrontmatterReconstruct(frontmatterLines []string, markdown string) string {
	var lines []string
	lines = append(lines, "---")
	lines = append(lines, frontmatterLines...)
	lines = append(lines, "---")
	if markdown != "" {
		// Add empty line before markdown content to match original format
		lines = append(lines, "")
		lines = append(lines, markdown)
	}
	return strings.Join(lines, "\n")
}

// updateFieldInFrontmatterFallback implements the original behavior as a fallback
func updateFieldInFrontmatterFallback(result *parser.FrontmatterResult, fieldName, fieldValue string) (string, error) {
	// Initialize frontmatter if it doesn't exist
	if result.Frontmatter == nil {
		result.Frontmatter = make(map[string]any)
	}

	// Update the field
	result.Frontmatter[fieldName] = fieldValue

	// Convert back to YAML with proper field ordering
	updatedFrontmatter, err := workflow.MarshalWithFieldOrder(result.Frontmatter, constants.PriorityWorkflowFields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}

	// Clean up quoted keys - replace "on": with on: at the start of a line
	frontmatterStr := strings.TrimSuffix(string(updatedFrontmatter), "\n")
	frontmatterStr = workflow.UnquoteYAMLKey(frontmatterStr, "on")

	// Reconstruct the file
	var lines []string
	lines = append(lines, "---")
	if frontmatterStr != "" {
		lines = append(lines, strings.Split(frontmatterStr, "\n")...)
	}
	lines = append(lines, "---")
	if result.Markdown != "" {
		lines = append(lines, result.Markdown)
	}

	return strings.Join(lines, "\n"), nil
}

// addFieldToFrontmatter adds a new field to the frontmatter while preserving formatting.
// This is used when we know the field doesn't exist yet.
// When trailingBlankLine is true, a blank line is appended after the new field to provide
// visual separation from fields added subsequently (e.g., separating engine from source).
func addFieldToFrontmatter(content, fieldName, fieldValue string, trailingBlankLine bool) (string, error) {
	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Try to preserve original frontmatter formatting by manually inserting the field
	if len(result.FrontmatterLines) > 0 {
		// Check if field already exists
		if result.Frontmatter != nil {
			if _, exists := result.Frontmatter[fieldName]; exists {
				// Field exists, update it instead
				return UpdateFieldInFrontmatter(content, fieldName, fieldValue)
			}
		}

		// Field doesn't exist, add it manually to preserve formatting
		frontmatterLines := append([]string(nil), result.FrontmatterLines...)

		// Add field at the end of the frontmatter, preserving original formatting
		newField := fmt.Sprintf("%s: %s", fieldName, fieldValue)
		frontmatterLines = append(frontmatterLines, newField)
		if trailingBlankLine {
			frontmatterLines = append(frontmatterLines, "")
		}

		// Reconstruct the file with preserved formatting
		var lines []string
		lines = append(lines, "---")
		lines = append(lines, frontmatterLines...)
		lines = append(lines, "---")
		if result.Markdown != "" {
			// Add empty line before markdown content to match original format
			lines = append(lines, "")
			lines = append(lines, result.Markdown)
		}

		return strings.Join(lines, "\n"), nil
	}

	// Fallback to original behavior if no frontmatter lines are available
	return updateFieldInFrontmatterFallback(result, fieldName, fieldValue)
}

// RemoveFieldFromOnTrigger removes a field from the 'on' trigger object in the frontmatter.
// This handles nested fields like "stop-after" which are located under the "on" key.
// It preserves the original formatting of the frontmatter including comments and blank lines.
func RemoveFieldFromOnTrigger(content, fieldName string) (string, error) {
	frontmatterEditorLog.Printf("Removing field from 'on' trigger: %s", fieldName)

	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		frontmatterEditorLog.Printf("Failed to parse frontmatter: %v", err)
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Check if frontmatter exists
	if result.Frontmatter == nil {
		// No frontmatter, return content unchanged
		return content, nil
	}

	// Check if 'on' field exists
	onValue, exists := result.Frontmatter["on"]
	if !exists {
		// No 'on' field, return content unchanged
		return content, nil
	}

	// Check if 'on' is an object (map)
	onMap, isMap := onValue.(map[string]any)
	if !isMap {
		// 'on' is not a map (might be a string), return content unchanged
		return content, nil
	}

	// Check if the field to remove exists in the 'on' map
	if _, fieldExists := onMap[fieldName]; !fieldExists {
		// Field doesn't exist, return content unchanged
		return content, nil
	}

	// Work with raw frontmatter lines to preserve formatting
	if len(result.FrontmatterLines) > 0 {
		frontmatterLines := removeFieldFromOnTriggerRaw(result.FrontmatterLines, fieldName)
		frontmatterEditorLog.Printf("Successfully removed field %s from 'on' trigger", fieldName)
		return removeFieldFromOnTriggerReconstruct(frontmatterLines, result.Markdown), nil
	}

	// This should rarely happen since we already checked for frontmatter existence
	frontmatterEditorLog.Printf("No raw frontmatter lines available, returning content unchanged")
	return content, nil
}

func removeFieldFromOnTriggerRaw(lines []string, fieldName string) []string {
	frontmatterEditorLog.Printf("Using raw frontmatter lines to remove field (%d lines)", len(lines))
	frontmatterLines := make([]string, 0, len(lines))
	inOnBlock := false
	onIndentLevel := 0
	skipNextLine := false
	fieldIndentLevel := 0
	for i := range len(lines) {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)
		if skipNextLine {
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent > fieldIndentLevel {
				continue
			}
			skipNextLine = false
		}
		if !inOnBlock && removeFieldFromOnTriggerIsOnLine(trimmedLine) {
			inOnBlock = true
			onIndentLevel = len(line) - len(strings.TrimLeft(line, " \t"))
			frontmatterLines = append(frontmatterLines, line)
			continue
		}
		if inOnBlock {
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && currentIndent <= onIndentLevel {
				inOnBlock = false
				frontmatterLines = append(frontmatterLines, line)
				continue
			}
			if removeFieldFromOnTriggerIsField(trimmedLine, fieldName) {
				frontmatterEditorLog.Printf("Found field %s to remove at line %d", fieldName, i+1)
				fieldIndentLevel = currentIndent
				skipNextLine = true
				continue
			}
		}
		frontmatterLines = append(frontmatterLines, line)
	}
	return frontmatterLines
}

func removeFieldFromOnTriggerIsOnLine(trimmedLine string) bool {
	return trimmedLine == "on:" || trimmedLine == `"on":` ||
		strings.HasPrefix(trimmedLine, "on: #") || strings.HasPrefix(trimmedLine, `"on": #`)
}

func removeFieldFromOnTriggerIsField(trimmedLine, fieldName string) bool {
	return trimmedLine == fieldName+":" ||
		strings.HasPrefix(trimmedLine, fieldName+": ") ||
		strings.HasPrefix(trimmedLine, fieldName+":\t")
}

func removeFieldFromOnTriggerReconstruct(frontmatterLines []string, markdown string) string {
	var lines []string
	lines = append(lines, "---")
	lines = append(lines, frontmatterLines...)
	lines = append(lines, "---")
	if markdown != "" {
		// Add empty line before markdown content to match original format
		lines = append(lines, "")
		lines = append(lines, markdown)
	}
	return strings.Join(lines, "\n")
}

// SetFieldInOnTrigger sets a field value in the 'on' trigger object in the frontmatter.
// This handles nested fields like "stop-after" which are located under the "on" key.
// It preserves the original formatting of the frontmatter including comments and blank lines.
func SetFieldInOnTrigger(content, fieldName, fieldValue string) (string, error) {
	frontmatterEditorLog.Printf("Setting field in 'on' trigger: %s = %s", fieldName, fieldValue)

	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		frontmatterEditorLog.Printf("Failed to parse frontmatter: %v", err)
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Check if frontmatter exists
	if result.Frontmatter == nil {
		// No frontmatter, cannot set nested field without 'on' block
		return "", errors.New("no frontmatter found, cannot set field in 'on' trigger")
	}

	// Check if 'on' field exists
	onValue, exists := result.Frontmatter["on"]
	if !exists {
		// No 'on' field exists, need to create it
		// Add the 'on:' block with the field at the beginning of frontmatter
		if len(result.FrontmatterLines) > 0 {
			return setFieldInOnTriggerCreateBlock(result, fieldName, fieldValue), nil
		}

		// No frontmatter lines, cannot create 'on' block
		return "", errors.New("no frontmatter found, cannot set field in 'on' trigger")
	}

	// Check if 'on' is an object (map)
	_, isMap := onValue.(map[string]any)
	if !isMap {
		// 'on' is not a map (might be a string), cannot set field
		return "", errors.New("'on' field is not an object, cannot set nested field")
	}

	// Work with raw frontmatter lines to preserve formatting
	if len(result.FrontmatterLines) > 0 {
		frontmatterLines, fieldUpdated := setFieldInOnTriggerRaw(result.FrontmatterLines, fieldName, fieldValue)
		if !fieldUpdated {
			return "", fmt.Errorf("failed to set field %s in 'on' trigger", fieldName)
		}

		frontmatterEditorLog.Printf("Successfully set field %s in 'on' trigger", fieldName)
		return setFieldInOnTriggerReconstruct(frontmatterLines, result.Markdown), nil
	}

	// This should rarely happen since we already checked for frontmatter existence
	frontmatterEditorLog.Printf("No raw frontmatter lines available")
	return "", errors.New("no frontmatter lines available to modify")
}

func setFieldInOnTriggerCreateBlock(result *parser.FrontmatterResult, fieldName, fieldValue string) string {
	frontmatterEditorLog.Printf("Creating 'on' block with field %s", fieldName)
	frontmatterLines := make([]string, 0, len(result.FrontmatterLines)+2)
	frontmatterLines = append(frontmatterLines, "on:")
	frontmatterLines = append(frontmatterLines, fmt.Sprintf("    %s: %s", fieldName, fieldValue))
	frontmatterLines = append(frontmatterLines, result.FrontmatterLines...)
	frontmatterEditorLog.Printf("Successfully created 'on' block with field %s", fieldName)
	return setFieldInOnTriggerReconstruct(frontmatterLines, result.Markdown)
}

func setFieldInOnTriggerRaw(lines []string, fieldName, fieldValue string) ([]string, bool) {
	frontmatterEditorLog.Printf("Using raw frontmatter lines to set field (%d lines)", len(lines))
	frontmatterLines := make([]string, 0, len(lines))
	inOnBlock := false
	onIndentLevel := 0
	fieldUpdated := false
	for i := range len(lines) {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)
		if !inOnBlock && removeFieldFromOnTriggerIsOnLine(trimmedLine) {
			inOnBlock = true
			onIndentLevel = len(line) - len(strings.TrimLeft(line, " \t"))
			frontmatterLines = append(frontmatterLines, line)
			continue
		}
		if inOnBlock {
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && currentIndent <= onIndentLevel {
				inOnBlock = false
				frontmatterLines, fieldUpdated = setFieldInOnTriggerAppendNew(frontmatterLines, fieldUpdated, onIndentLevel, fieldName, fieldValue, false)
				frontmatterLines = append(frontmatterLines, line)
				continue
			}
			if removeFieldFromOnTriggerIsField(trimmedLine, fieldName) {
				frontmatterLines = append(frontmatterLines, setFieldInOnTriggerUpdatedLine(line, fieldName, fieldValue))
				fieldUpdated = true
				frontmatterEditorLog.Printf("Updated existing field %s in 'on' block", fieldName)
				continue
			}
		}
		frontmatterLines = append(frontmatterLines, line)
	}
	if inOnBlock && !fieldUpdated {
		frontmatterLines, fieldUpdated = setFieldInOnTriggerAppendNew(frontmatterLines, fieldUpdated, onIndentLevel, fieldName, fieldValue, true)
	}
	return frontmatterLines, fieldUpdated
}

func setFieldInOnTriggerAppendNew(lines []string, fieldUpdated bool, onIndentLevel int, fieldName, fieldValue string, atEnd bool) ([]string, bool) {
	if fieldUpdated {
		return lines, fieldUpdated
	}
	indent := strings.Repeat(" ", onIndentLevel+4)
	lines = append(lines, fmt.Sprintf("%s%s: %s", indent, fieldName, fieldValue))
	if atEnd {
		frontmatterEditorLog.Printf("Added new field %s at end of 'on' block", fieldName)
	} else {
		frontmatterEditorLog.Printf("Added new field %s to 'on' block", fieldName)
	}
	return lines, true
}

func setFieldInOnTriggerUpdatedLine(line, fieldName, fieldValue string) string {
	leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	fieldSep := fieldName + ":"
	fieldSepIndex := strings.Index(line, fieldSep)
	commentIndex := strings.Index(line, "#")
	var comment string
	if fieldSepIndex != -1 && commentIndex > fieldSepIndex {
		comment = line[commentIndex:]
	}
	if comment != "" {
		return fmt.Sprintf("%s%s: %s %s", leadingSpace, fieldName, fieldValue, comment)
	}
	return fmt.Sprintf("%s%s: %s", leadingSpace, fieldName, fieldValue)
}

func setFieldInOnTriggerReconstruct(frontmatterLines []string, markdown string) string {
	var lines []string
	lines = append(lines, "---")
	lines = append(lines, frontmatterLines...)
	lines = append(lines, "---")
	if markdown != "" {
		// Add empty line before markdown content to match original format
		lines = append(lines, "")
		lines = append(lines, markdown)
	}
	return strings.Join(lines, "\n")
}

// UpdateScheduleInOnBlock updates the "schedule" sub-key inside the "on:" block mapping in
// the workflow frontmatter. It replaces the existing schedule value—whether a scalar
// (schedule: daily) or a list (schedule:\n  - cron: "0 9 * * *")—with a new scalar
// expression, while preserving all sibling trigger keys (e.g., workflow_dispatch, push).
func UpdateScheduleInOnBlock(content, scheduleExpr string) (string, error) {
	frontmatterEditorLog.Printf("Updating schedule in on: block to %q", scheduleExpr)

	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if len(result.FrontmatterLines) == 0 {
		return "", errors.New("no frontmatter lines available to modify")
	}

	frontmatterLines, scheduleFound := updateScheduleInOnBlockRaw(result.FrontmatterLines, scheduleExpr)
	if !scheduleFound {
		return "", errors.New("schedule key not found inside on: block")
	}

	return updateScheduleInOnBlockReconstruct(frontmatterLines, result.Markdown), nil
}

func updateScheduleInOnBlockRaw(lines []string, scheduleExpr string) ([]string, bool) {
	frontmatterLines := make([]string, 0, len(lines))
	inOnBlock := false
	onIndentLevel := 0
	scheduleFound := false
	skipScheduleChildren := false
	scheduleIndentLevel := 0
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
		if !inOnBlock && removeFieldFromOnTriggerIsOnLine(trimmedLine) {
			inOnBlock = true
			onIndentLevel = currentIndent
			frontmatterLines = append(frontmatterLines, line)
			continue
		}
		if inOnBlock {
			if skipScheduleChildren && currentIndent > scheduleIndentLevel {
				continue
			}
			skipScheduleChildren = false
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && currentIndent <= onIndentLevel {
				inOnBlock = false
				frontmatterLines = append(frontmatterLines, line)
				continue
			}
			if !scheduleFound && updateScheduleInOnBlockIsSchedule(trimmedLine) {
				frontmatterLines = append(frontmatterLines, fmt.Sprintf("%sschedule: %s", line[:currentIndent], scheduleExpr))
				scheduleFound = true
				scheduleIndentLevel = currentIndent
				skipScheduleChildren = true
				frontmatterEditorLog.Printf("Updated schedule in on: block to %q", scheduleExpr)
				continue
			}
		}
		frontmatterLines = append(frontmatterLines, line)
	}
	return frontmatterLines, scheduleFound
}

func updateScheduleInOnBlockIsSchedule(trimmedLine string) bool {
	return trimmedLine == "schedule:" ||
		strings.HasPrefix(trimmedLine, "schedule: ") ||
		strings.HasPrefix(trimmedLine, "schedule:\t")
}

func updateScheduleInOnBlockReconstruct(frontmatterLines []string, markdown string) string {
	var lines []string
	lines = append(lines, "---")
	lines = append(lines, frontmatterLines...)
	lines = append(lines, "---")
	if markdown != "" {
		lines = append(lines, "")
		lines = append(lines, markdown)
	}
	return strings.Join(lines, "\n")
}
