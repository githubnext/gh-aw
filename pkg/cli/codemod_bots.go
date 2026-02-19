package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var botsCodemodLog = logger.New("cli:codemod_bots")

// getBotsToOnBotsCodemod creates a codemod for moving top-level 'bots' to 'on.bots'
func getBotsToOnBotsCodemod() Codemod {
	return Codemod{
		ID:           "bots-to-on-bots",
		Name:         "Move bots to on.bots",
		Description:  "Moves the top-level 'bots' field to 'on.bots' as per the new frontmatter structure",
		IntroducedIn: "0.10.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if top-level bots exists
			_, hasTopLevelBots := frontmatter["bots"]
			if !hasTopLevelBots {
				return content, false, nil
			}

			// Check if on.bots already exists (shouldn't happen, but be safe)
			if onValue, hasOn := frontmatter["on"]; hasOn {
				if onMap, ok := onValue.(map[string]any); ok {
					if _, hasOnBots := onMap["bots"]; hasOnBots {
						botsCodemodLog.Print("Both top-level 'bots' and 'on.bots' exist - skipping migration")
						return content, false, nil
					}
				}
			}

			// Parse frontmatter to get raw lines
			frontmatterLines, markdown, err := parseFrontmatterLines(content)
			if err != nil {
				return content, false, err
			}

			// Find bots line and on: block
			var botsLineIdx = -1
			var botsLineValue string
			var onBlockIdx = -1
			var onIndent string

			// First pass: find the bots line and on: block
			for i, line := range frontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Find top-level bots
				if isTopLevelKey(line) && strings.HasPrefix(trimmedLine, "bots:") {
					botsLineIdx = i
					// Extract the value (could be on same line or on next lines)
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						botsLineValue = strings.TrimSpace(parts[1])
					}
					botsCodemodLog.Printf("Found top-level bots at line %d", i+1)
				}

				// Find on: block
				if isTopLevelKey(line) && strings.HasPrefix(trimmedLine, "on:") {
					onBlockIdx = i
					onIndent = getIndentation(line)
					botsCodemodLog.Printf("Found 'on:' block at line %d", i+1)
				}
			}

			// If no bots found, nothing to do
			if botsLineIdx == -1 {
				return content, false, nil
			}

			// Determine how bots is formatted
			var botsLines []string
			var botsEndIdx int

			if strings.HasPrefix(botsLineValue, "[") {
				// bots: [dependabot, renovate] - single line format
				botsLines = []string{frontmatterLines[botsLineIdx]}
				botsEndIdx = botsLineIdx
			} else {
				// Multi-line array format OR bots: with empty value
				// Find all lines that are part of the bots block
				botsStartIndent := getIndentation(frontmatterLines[botsLineIdx])
				botsLines = append(botsLines, frontmatterLines[botsLineIdx])
				botsEndIdx = botsLineIdx

				for j := botsLineIdx + 1; j < len(frontmatterLines); j++ {
					line := frontmatterLines[j]
					trimmed := strings.TrimSpace(line)

					// Empty lines or comments might be part of the block
					if trimmed == "" || strings.HasPrefix(trimmed, "#") {
						botsLines = append(botsLines, line)
						botsEndIdx = j
						continue
					}

					// Check if still in the bots block (indented more than bots:)
					if isNestedUnder(line, botsStartIndent) {
						botsLines = append(botsLines, line)
						botsEndIdx = j
					} else {
						// Exited the block
						break
					}
				}
			}

			botsCodemodLog.Printf("Bots spans lines %d to %d (%d lines)", botsLineIdx+1, botsEndIdx+1, len(botsLines))

			// If no on: block found, we need to create one
			result := make([]string, 0, len(frontmatterLines))
			modified := false

			if onBlockIdx == -1 {
				// No on: block exists - create one with bots inside it
				botsCodemodLog.Print("No 'on:' block found - creating new one with bots")

				for i, line := range frontmatterLines {
					if i >= botsLineIdx && i <= botsEndIdx {
						// Skip the original bots lines - we'll add them to the new on: block
						if i == botsLineIdx {
							// Add new on: block with bots inside
							result = append(result, "on:")
							// Add bots lines with proper indentation
							for _, botsLine := range botsLines {
								trimmed := strings.TrimSpace(botsLine)
								if trimmed == "" {
									result = append(result, botsLine)
								} else if strings.HasPrefix(trimmed, "bots:") {
									// bots: line gets 2 spaces (nested under on:)
									result = append(result, "  "+botsLine)
								} else {
									// Array items get 4 spaces (nested under on: and bots:)
									result = append(result, "    "+trimmed)
								}
							}
							modified = true
						}
						// Skip all other bots lines
						continue
					}
					result = append(result, line)
				}
			} else {
				// on: block exists - add bots to it
				botsCodemodLog.Print("Found 'on:' block - adding bots to it")

				// Determine indentation for items inside on: block
				onItemIndent := onIndent + "  "

				// Track if we've inserted bots
				insertedBots := false

				for i, line := range frontmatterLines {
					// Skip the original bots lines
					if i >= botsLineIdx && i <= botsEndIdx {
						modified = true
						continue
					}

					// Add the line
					result = append(result, line)

					// After the on: line, insert bots
					if i == onBlockIdx && !insertedBots {
						// Add bots lines with proper indentation inside on: block
						for _, botsLine := range botsLines {
							trimmed := strings.TrimSpace(botsLine)
							if trimmed == "" {
								result = append(result, botsLine)
							} else {
								// Adjust indentation to be nested under on:
								// Remove "bots:" prefix and re-add with proper indentation
								if strings.HasPrefix(trimmed, "bots:") {
									// bots: value or bots:
									parts := strings.SplitN(trimmed, ":", 2)
									if len(parts) == 2 {
										result = append(result, fmt.Sprintf("%sbots:%s", onItemIndent, parts[1]))
									} else {
										result = append(result, fmt.Sprintf("%sbots:", onItemIndent))
									}
								} else {
									// Array item line (e.g., "- dependabot")
									// These should be indented 2 more spaces than bots: to be nested under it
									result = append(result, onItemIndent+"  "+trimmed)
								}
							}
						}
						insertedBots = true
					}
				}
			}

			if !modified {
				return content, false, nil
			}

			// Reconstruct the content
			newContent := reconstructContent(result, markdown)
			botsCodemodLog.Print("Successfully migrated top-level 'bots' to 'on.bots'")
			return newContent, true, nil
		},
	}
}
