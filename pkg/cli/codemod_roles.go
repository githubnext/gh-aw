package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var rolesCodemodLog = logger.New("cli:codemod_roles")

// getRolesToOnRolesCodemod creates a codemod for moving top-level 'roles' to 'on.roles'
func getRolesToOnRolesCodemod() Codemod {
	return Codemod{
		ID:           "roles-to-on-roles",
		Name:         "Move roles to on.roles",
		Description:  "Moves the top-level 'roles' field to 'on.roles' as per the new frontmatter structure",
		IntroducedIn: "0.10.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if top-level roles exists
			_, hasTopLevelRoles := frontmatter["roles"]
			if !hasTopLevelRoles {
				return content, false, nil
			}

			// Check if on.roles already exists (shouldn't happen, but be safe)
			if onValue, hasOn := frontmatter["on"]; hasOn {
				if onMap, ok := onValue.(map[string]any); ok {
					if _, hasOnRoles := onMap["roles"]; hasOnRoles {
						rolesCodemodLog.Print("Both top-level 'roles' and 'on.roles' exist - skipping migration")
						return content, false, nil
					}
				}
			}

			// Parse frontmatter to get raw lines
			frontmatterLines, markdown, err := parseFrontmatterLines(content)
			if err != nil {
				return content, false, err
			}

			// Find roles line and on: block
			var rolesLineIdx = -1
			var rolesLineValue string
			var onBlockIdx = -1
			var onIndent string

			// First pass: find the roles line and on: block
			for i, line := range frontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Find top-level roles
				if isTopLevelKey(line) && strings.HasPrefix(trimmedLine, "roles:") {
					rolesLineIdx = i
					// Extract the value (could be on same line or on next lines)
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						rolesLineValue = strings.TrimSpace(parts[1])
					}
					rolesCodemodLog.Printf("Found top-level roles at line %d", i+1)
				}

				// Find on: block
				if isTopLevelKey(line) && strings.HasPrefix(trimmedLine, "on:") {
					onBlockIdx = i
					onIndent = getIndentation(line)
					rolesCodemodLog.Printf("Found 'on:' block at line %d", i+1)
				}
			}

			// If no roles found, nothing to do
			if rolesLineIdx == -1 {
				return content, false, nil
			}

			// Determine how roles is formatted
			var rolesLines []string
			var rolesEndIdx int

			if rolesLineValue == "all" || strings.HasPrefix(rolesLineValue, "[") {
				// roles: all or roles: [admin, write] - single line format
				rolesLines = []string{frontmatterLines[rolesLineIdx]}
				rolesEndIdx = rolesLineIdx
			} else {
				// Multi-line array format OR roles: with empty value
				// Find all lines that are part of the roles block
				rolesStartIndent := getIndentation(frontmatterLines[rolesLineIdx])
				rolesLines = append(rolesLines, frontmatterLines[rolesLineIdx])
				rolesEndIdx = rolesLineIdx

				for j := rolesLineIdx + 1; j < len(frontmatterLines); j++ {
					line := frontmatterLines[j]
					trimmed := strings.TrimSpace(line)

					// Empty lines or comments might be part of the block
					if trimmed == "" || strings.HasPrefix(trimmed, "#") {
						rolesLines = append(rolesLines, line)
						rolesEndIdx = j
						continue
					}

					// Check if still in the roles block (indented more than roles:)
					if isNestedUnder(line, rolesStartIndent) {
						rolesLines = append(rolesLines, line)
						rolesEndIdx = j
					} else {
						// Exited the block
						break
					}
				}
			}

			rolesCodemodLog.Printf("Roles spans lines %d to %d (%d lines)", rolesLineIdx+1, rolesEndIdx+1, len(rolesLines))

			// If no on: block found, we need to create one
			result := make([]string, 0, len(frontmatterLines))
			modified := false

			if onBlockIdx == -1 {
				// No on: block exists - create one with roles inside it
				rolesCodemodLog.Print("No 'on:' block found - creating new one with roles")

				for i, line := range frontmatterLines {
					if i >= rolesLineIdx && i <= rolesEndIdx {
						// Skip the original roles lines - we'll add them to the new on: block
						if i == rolesLineIdx {
							// Add new on: block with roles inside
							result = append(result, "on:")
							// Add roles lines with proper indentation
							for _, rolesLine := range rolesLines {
								trimmed := strings.TrimSpace(rolesLine)
								if trimmed == "" {
									result = append(result, rolesLine)
								} else if strings.HasPrefix(trimmed, "roles:") {
									// roles: line gets 2 spaces (nested under on:)
									result = append(result, "  "+rolesLine)
								} else {
									// Array items get 4 spaces (nested under on: and roles:)
									result = append(result, "    "+trimmed)
								}
							}
							modified = true
						}
						// Skip all other roles lines
						continue
					}
					result = append(result, line)
				}
			} else {
				// on: block exists - add roles to it
				rolesCodemodLog.Print("Found 'on:' block - adding roles to it")

				// Determine indentation for items inside on: block
				onItemIndent := onIndent + "  "

				// Track if we've inserted roles
				insertedRoles := false

				for i, line := range frontmatterLines {
					// Skip the original roles lines
					if i >= rolesLineIdx && i <= rolesEndIdx {
						modified = true
						continue
					}

					// Add the line
					result = append(result, line)

					// After the on: line, insert roles
					if i == onBlockIdx && !insertedRoles {
						// Add roles lines with proper indentation inside on: block
						for _, rolesLine := range rolesLines {
							trimmed := strings.TrimSpace(rolesLine)
							if trimmed == "" {
								result = append(result, rolesLine)
							} else {
								// Adjust indentation to be nested under on:
								// Remove "roles:" prefix and re-add with proper indentation
								if strings.HasPrefix(trimmed, "roles:") {
									// roles: value or roles:
									parts := strings.SplitN(trimmed, ":", 2)
									if len(parts) == 2 {
										result = append(result, fmt.Sprintf("%sroles:%s", onItemIndent, parts[1]))
									} else {
										result = append(result, fmt.Sprintf("%sroles:", onItemIndent))
									}
								} else {
									// Array item line (e.g., "- admin")
									// These should be indented 2 more spaces than roles: to be nested under it
									result = append(result, onItemIndent+"  "+trimmed)
								}
							}
						}
						insertedRoles = true
					}
				}
			}

			if !modified {
				return content, false, nil
			}

			// Reconstruct the content
			newContent := reconstructContent(result, markdown)
			rolesCodemodLog.Print("Successfully migrated top-level 'roles' to 'on.roles'")
			return newContent, true, nil
		},
	}
}
