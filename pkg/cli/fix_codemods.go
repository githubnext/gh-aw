package cli

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var codemodsLog = logger.New("cli:codemods")

// Codemod represents a single code transformation that can be applied to workflow files
type Codemod struct {
	ID          string // Unique identifier for the codemod
	Name        string // Human-readable name
	Description string // Description of what the codemod does
	Apply       func(content string, frontmatter map[string]any) (string, bool, error)
}

// CodemodResult represents the result of applying a codemod
type CodemodResult struct {
	Applied bool   // Whether the codemod was applied
	Message string // Description of what changed
}

// GetAllCodemods returns all available codemods in the registry
func GetAllCodemods() []Codemod {
	return []Codemod{
		getTimeoutMinutesCodemod(),
		getNetworkFirewallCodemod(),
	}
}

// getTimeoutMinutesCodemod creates a codemod for migrating timeout_minutes to timeout-minutes
func getTimeoutMinutesCodemod() Codemod {
	return Codemod{
		ID:          "timeout-minutes-migration",
		Name:        "Migrate timeout_minutes to timeout-minutes",
		Description: "Replaces deprecated 'timeout_minutes' field with 'timeout-minutes'",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if the deprecated field exists
			value, exists := frontmatter["timeout_minutes"]
			if !exists {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Replace the field in raw lines while preserving formatting
			var modified bool
			frontmatterLines := make([]string, len(result.FrontmatterLines))
			for i, line := range result.FrontmatterLines {
				// Check if this line contains the deprecated field
				trimmedLine := strings.TrimSpace(line)
				if strings.HasPrefix(trimmedLine, "timeout_minutes:") {
					// Preserve indentation
					leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

					// Extract the value and any trailing comment
					parts := strings.SplitN(line, ":", 2)
					if len(parts) >= 2 {
						valueAndComment := parts[1]
						frontmatterLines[i] = fmt.Sprintf("%stimeout-minutes:%s", leadingSpace, valueAndComment)
						modified = true
						codemodsLog.Printf("Replaced timeout_minutes with timeout-minutes on line %d", i+1)
					} else {
						frontmatterLines[i] = line
					}
				} else {
					frontmatterLines[i] = line
				}
			}

			if !modified {
				return content, false, nil
			}

			// Reconstruct the content
			var lines []string
			lines = append(lines, "---")
			lines = append(lines, frontmatterLines...)
			lines = append(lines, "---")
			if result.Markdown != "" {
				lines = append(lines, "")
				lines = append(lines, result.Markdown)
			}

			newContent := strings.Join(lines, "\n")
			codemodsLog.Printf("Applied timeout_minutes migration (value: %v)", value)
			return newContent, true, nil
		},
	}
}

// getNetworkFirewallCodemod creates a codemod for migrating network.firewall to sandbox.agent: false
func getNetworkFirewallCodemod() Codemod {
	return Codemod{
		ID:          "network-firewall-migration",
		Name:        "Migrate network.firewall to sandbox.agent",
		Description: "Replaces deprecated 'network.firewall' field with 'sandbox.agent: false'",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if network.firewall exists
			networkValue, hasNetwork := frontmatter["network"]
			if !hasNetwork {
				return content, false, nil
			}

			networkMap, ok := networkValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if firewall field exists in network
			_, hasFirewall := networkMap["firewall"]
			if !hasFirewall {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Find and remove the firewall line, add sandbox.agent if needed
			var modified bool
			var firewallIndent string
			var inNetworkBlock bool
			var networkIndent string
			var firewallLineIndex = -1

			frontmatterLines := make([]string, 0, len(result.FrontmatterLines))

			for i, line := range result.FrontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the network block
				if strings.HasPrefix(trimmedLine, "network:") {
					inNetworkBlock = true
					networkIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					frontmatterLines = append(frontmatterLines, line)
					continue
				}

				// Check if we've left the network block (new top-level key with same or less indentation)
				if inNetworkBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					if len(currentIndent) <= len(networkIndent) && strings.Contains(line, ":") {
						inNetworkBlock = false
					}
				}

				// Remove firewall line if in network block
				if inNetworkBlock && strings.HasPrefix(trimmedLine, "firewall:") {
					firewallIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					firewallLineIndex = i
					modified = true
					codemodsLog.Printf("Removed network.firewall on line %d", i+1)
					continue
				}

				frontmatterLines = append(frontmatterLines, line)
			}

			if !modified {
				return content, false, nil
			}

			// Add sandbox.agent: false if not already present
			_, hasSandbox := frontmatter["sandbox"]
			if !hasSandbox {
				// Add sandbox.agent: false at the top level
				// Try to place it after network block if we found firewall
				if firewallLineIndex >= 0 && len(firewallIndent) > 0 {
					// Use network-level indentation (typically no indentation for top-level)
					sandboxLines := []string{
						"sandbox:",
						"  agent: false  # Firewall disabled (migrated from network.firewall)",
					}

					// Find where to insert (after network block)
					insertIndex := -1
					inNet := false
					for i, line := range frontmatterLines {
						trimmed := strings.TrimSpace(line)
						if strings.HasPrefix(trimmed, "network:") {
							inNet = true
						} else if inNet && len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(trimmed, "#") {
							// Found next top-level key
							insertIndex = i
							break
						}
					}

					if insertIndex >= 0 {
						// Insert after network block
						newLines := make([]string, 0, len(frontmatterLines)+len(sandboxLines))
						newLines = append(newLines, frontmatterLines[:insertIndex]...)
						newLines = append(newLines, sandboxLines...)
						newLines = append(newLines, frontmatterLines[insertIndex:]...)
						frontmatterLines = newLines
					} else {
						// Append at the end
						frontmatterLines = append(frontmatterLines, sandboxLines...)
					}

					codemodsLog.Print("Added sandbox.agent: false")
				} else {
					// Just append at the end
					frontmatterLines = append(frontmatterLines, "sandbox:")
					frontmatterLines = append(frontmatterLines, "  agent: false  # Firewall disabled (migrated from network.firewall)")
					codemodsLog.Print("Added sandbox.agent: false at end")
				}
			}

			// Reconstruct the content
			var lines []string
			lines = append(lines, "---")
			lines = append(lines, frontmatterLines...)
			lines = append(lines, "---")
			if result.Markdown != "" {
				lines = append(lines, "")
				lines = append(lines, result.Markdown)
			}

			newContent := strings.Join(lines, "\n")
			codemodsLog.Print("Applied network.firewall migration")
			return newContent, true, nil
		},
	}
}
