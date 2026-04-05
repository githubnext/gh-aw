package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

// PostTransformFunc is an optional hook called after the primary field removal.
// It receives the already-modified lines, the full frontmatter, and the removed
// field's value. It returns the (potentially further modified) lines.
type PostTransformFunc func(lines []string, frontmatter map[string]any, fieldValue any) []string

// fieldRemovalCodemodConfig holds the configuration for a field-removal codemod.
type fieldRemovalCodemodConfig struct {
	ID            string
	Name          string
	Description   string
	IntroducedIn  string
	ParentKey     string            // Top-level frontmatter key that contains the field
	FieldKey      string            // Child field to remove from the parent block
	LogMsg        string            // Debug log message emitted when the codemod is applied
	Log           *logger.Logger    // Logger for the codemod
	PostTransform PostTransformFunc // Optional hook for additional transforms after field removal
}

// newFieldRemovalCodemod creates a Codemod that:
//  1. Checks that the parent key is present in the frontmatter and is a map.
//  2. Checks that the child field is present in that map.
//  3. Removes the field (and any nested content) from the YAML block.
//  4. Optionally invokes PostTransform for any additional line-level changes.
func newFieldRemovalCodemod(cfg fieldRemovalCodemodConfig) Codemod {
	return Codemod{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Description:  cfg.Description,
		IntroducedIn: cfg.IntroducedIn,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			parentValue, hasParent := frontmatter[cfg.ParentKey]
			if !hasParent {
				return content, false, nil
			}

			parentMap, ok := parentValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			fieldValue, hasField := parentMap[cfg.FieldKey]
			if !hasField {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, cfg.FieldKey, cfg.ParentKey)
				if !modified {
					return lines, false
				}

				if cfg.PostTransform != nil {
					result = cfg.PostTransform(result, frontmatter, fieldValue)
				}

				return result, true
			})
			if applied {
				cfg.Log.Print(cfg.LogMsg)
			}
			return newContent, applied, err
		},
	}
}

// moveToOnBlockConfig holds the configuration for a codemod that moves a top-level
// frontmatter key into the 'on:' block.
type moveToOnBlockConfig struct {
	ID             string
	Name           string
	Description    string
	IntroducedIn   string
	FieldKey       string            // The top-level key to move (e.g. "bots" or "roles")
	IsInlineSingle func(string) bool // Returns true when the value fits on a single line as-is
	Log            *logger.Logger
}

// newMoveTopLevelKeyToOnBlockCodemod creates a Codemod that moves a top-level frontmatter
// key (e.g. "bots" or "roles") into the 'on:' block, creating the block if necessary.
//
// The algorithm:
//  1. Bail out if the top-level key is absent.
//  2. Bail out if on.<fieldKey> already exists (avoid double-migration).
//  3. Locate the field block and the existing 'on:' block in the YAML lines.
//  4. If no 'on:' block exists, replace the field lines with a new 'on:' block
//     that contains the field nested inside it.
//  5. If an 'on:' block exists, remove the original field lines and insert them
//     immediately after the 'on:' line with adjusted indentation.
func newMoveTopLevelKeyToOnBlockCodemod(cfg moveToOnBlockConfig) Codemod {
	fieldKey := cfg.FieldKey
	fieldKeyPrefix := fieldKey + ":"

	return Codemod{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Description:  cfg.Description,
		IntroducedIn: cfg.IntroducedIn,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Bail out if the top-level field does not exist.
			if _, hasField := frontmatter[fieldKey]; !hasField {
				return content, false, nil
			}

			// Bail out if on.<fieldKey> already exists.
			if onValue, hasOn := frontmatter["on"]; hasOn {
				if onMap, ok := onValue.(map[string]any); ok {
					if _, hasOnField := onMap[fieldKey]; hasOnField {
						cfg.Log.Printf("Both top-level '%s' and 'on.%s' exist - skipping migration", fieldKey, fieldKey)
						return content, false, nil
					}
				}
			}

			return applyFrontmatterLineTransform(content, func(frontmatterLines []string) ([]string, bool) {
				var fieldLineIdx = -1
				var fieldLineValue string
				var onBlockIdx = -1
				var onIndent string

				for i, line := range frontmatterLines {
					trimmedLine := strings.TrimSpace(line)

					if isTopLevelKey(line) && strings.HasPrefix(trimmedLine, fieldKeyPrefix) {
						fieldLineIdx = i
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							fieldLineValue = strings.TrimSpace(parts[1])
						}
						cfg.Log.Printf("Found top-level %s at line %d", fieldKey, i+1)
					}

					if isTopLevelKey(line) && strings.HasPrefix(trimmedLine, "on:") {
						onBlockIdx = i
						onIndent = getIndentation(line)
						cfg.Log.Printf("Found 'on:' block at line %d", i+1)
					}
				}

				if fieldLineIdx == -1 {
					return frontmatterLines, false
				}

				// Collect all lines that belong to the field block.
				var fieldLines []string
				var fieldEndIdx int

				if cfg.IsInlineSingle != nil && cfg.IsInlineSingle(fieldLineValue) {
					// Single-line format (e.g. "bots: [a, b]" or "roles: all")
					fieldLines = []string{frontmatterLines[fieldLineIdx]}
					fieldEndIdx = fieldLineIdx
				} else {
					// Multi-line or empty-value format — collect nested lines.
					fieldStartIndent := getIndentation(frontmatterLines[fieldLineIdx])
					fieldLines = append(fieldLines, frontmatterLines[fieldLineIdx])
					fieldEndIdx = fieldLineIdx

					for j := fieldLineIdx + 1; j < len(frontmatterLines); j++ {
						line := frontmatterLines[j]
						trimmed := strings.TrimSpace(line)

						if trimmed == "" || strings.HasPrefix(trimmed, "#") {
							fieldLines = append(fieldLines, line)
							fieldEndIdx = j
							continue
						}

						if isNestedUnder(line, fieldStartIndent) {
							fieldLines = append(fieldLines, line)
							fieldEndIdx = j
						} else {
							break
						}
					}
				}

				cfg.Log.Printf("%s spans lines %d to %d (%d lines)", fieldKey, fieldLineIdx+1, fieldEndIdx+1, len(fieldLines))

				result := make([]string, 0, len(frontmatterLines))

				if onBlockIdx == -1 {
					// No 'on:' block — create one and place the field inside it.
					cfg.Log.Printf("No 'on:' block found - creating new one with %s", fieldKey)

					for i, line := range frontmatterLines {
						if i >= fieldLineIdx && i <= fieldEndIdx {
							if i == fieldLineIdx {
								result = append(result, "on:")
								for _, fl := range fieldLines {
									trimmed := strings.TrimSpace(fl)
									if trimmed == "" {
										result = append(result, fl)
									} else if strings.HasPrefix(trimmed, fieldKeyPrefix) {
										result = append(result, "  "+fl)
									} else {
										result = append(result, "    "+trimmed)
									}
								}
							}
							continue
						}
						result = append(result, line)
					}
				} else {
					// 'on:' block exists — insert the field right after the 'on:' line.
					cfg.Log.Printf("Found 'on:' block - adding %s to it", fieldKey)

					onItemIndent := onIndent + "  "
					insertedField := false

					for i, line := range frontmatterLines {
						if i >= fieldLineIdx && i <= fieldEndIdx {
							continue
						}

						result = append(result, line)

						if i == onBlockIdx && !insertedField {
							for _, fl := range fieldLines {
								trimmed := strings.TrimSpace(fl)
								if trimmed == "" {
									result = append(result, fl)
								} else if strings.HasPrefix(trimmed, fieldKeyPrefix) {
									parts := strings.SplitN(trimmed, ":", 2)
									if len(parts) == 2 {
										result = append(result, fmt.Sprintf("%s%s:%s", onItemIndent, fieldKey, parts[1]))
									} else {
										result = append(result, onItemIndent+fieldKey+":")
									}
								} else {
									result = append(result, onItemIndent+"  "+trimmed)
								}
							}
							insertedField = true
						}
					}
				}

				cfg.Log.Printf("Successfully migrated top-level '%s' to 'on.%s'", fieldKey, fieldKey)
				return result, true
			})
		},
	}
}
