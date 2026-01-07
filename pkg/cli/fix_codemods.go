package cli

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var codemodsLog = logger.New("cli:codemods")

const (
	// Migration comment for network.firewall to sandbox.agent
	sandboxAgentComment = "# Firewall disabled (migrated from network.firewall)"
)

// getSandboxAgentFalseLines returns the standard lines for adding sandbox.agent: false
func getSandboxAgentFalseLines() []string {
	return []string{
		"sandbox:",
		"  agent: false  " + sandboxAgentComment,
	}
}

// Codemod represents a single code transformation that can be applied to workflow files
type Codemod struct {
	ID           string // Unique identifier for the codemod
	Name         string // Human-readable name
	Description  string // Description of what the codemod does
	IntroducedIn string // Version where this codemod was introduced
	Apply        func(content string, frontmatter map[string]any) (string, bool, error)
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
		getCommandToSlashCommandCodemod(),
		getSafeInputsModeCodemod(),
		getUploadAssetsCodemod(),
		getWritePermissionsCodemod(),
	}
}

// getTimeoutMinutesCodemod creates a codemod for migrating timeout_minutes to timeout-minutes
func getTimeoutMinutesCodemod() Codemod {
	return Codemod{
		ID:           "timeout-minutes-migration",
		Name:         "Migrate timeout_minutes to timeout-minutes",
		Description:  "Replaces deprecated 'timeout_minutes' field with 'timeout-minutes'",
		IntroducedIn: "0.1.0",
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

// getNetworkFirewallCodemod creates a codemod for migrating network.firewall to sandbox.agent
func getNetworkFirewallCodemod() Codemod {
	return Codemod{
		ID:           "network-firewall-migration",
		Name:         "Migrate network.firewall to sandbox.agent",
		Description:  "Replaces deprecated 'network.firewall' field with 'sandbox.agent' (false for disabled firewall, awf for enabled)",
		IntroducedIn: "0.1.0",
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
			firewallValue, hasFirewall := networkMap["firewall"]
			if !hasFirewall {
				return content, false, nil
			}

			// Determine the sandbox.agent value based on firewall value
			// firewall: true -> sandbox.agent: awf
			// firewall: false or null -> sandbox.agent: false
			var sandboxAgentValue string
			if firewallValue == true {
				sandboxAgentValue = "awf"
			} else {
				sandboxAgentValue = "false"
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Find and remove the firewall line (and all nested properties), add sandbox.agent if needed
			var modified bool
			var inNetworkBlock bool
			var networkIndent string
			var firewallLineIndex = -1
			var inFirewallBlock bool
			var firewallIndent string

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
					firewallLineIndex = i
					modified = true
					inFirewallBlock = true
					firewallIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					codemodsLog.Printf("Removed network.firewall on line %d (value was: %v)", i+1, firewallValue)
					continue
				}

				// Skip nested properties under firewall (lines with greater indentation)
				if inFirewallBlock {
					// Empty lines within the firewall block should be removed
					if len(trimmedLine) == 0 {
						continue
					}

					currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

					// Comments need to check indentation
					if strings.HasPrefix(trimmedLine, "#") {
						if len(currentIndent) > len(firewallIndent) {
							// Comment is nested under firewall, remove it
							codemodsLog.Printf("Removed nested firewall comment on line %d: %s", i+1, trimmedLine)
							continue
						}
						// Comment is at same or less indentation, exit firewall block and keep it
						inFirewallBlock = false
						frontmatterLines = append(frontmatterLines, line)
						continue
					}

					// If this line has more indentation than firewall, it's a nested property
					if len(currentIndent) > len(firewallIndent) {
						codemodsLog.Printf("Removed nested firewall property on line %d: %s", i+1, trimmedLine)
						continue
					}
					// We've exited the firewall block (found a line at same or less indentation)
					inFirewallBlock = false
				}

				frontmatterLines = append(frontmatterLines, line)
			}

			if !modified {
				return content, false, nil
			}

			// Add sandbox.agent if not already present
			_, hasSandbox := frontmatter["sandbox"]
			if !hasSandbox {
				// Create the appropriate sandbox lines based on firewall value
				var sandboxLines []string
				if sandboxAgentValue == "awf" {
					sandboxLines = []string{
						"sandbox:",
						"  agent: awf  # Firewall enabled (migrated from network.firewall)",
					}
				} else {
					sandboxLines = getSandboxAgentFalseLines()
				}

				// Try to place it after network block if we found firewall
				if firewallLineIndex >= 0 {
					// Find where to insert (after network block)
					insertIndex := -1
					inNet := false
					for i, line := range frontmatterLines {
						trimmed := strings.TrimSpace(line)
						if strings.HasPrefix(trimmed, "network:") {
							inNet = true
						} else if inNet && len(trimmed) > 0 {
							// Check if this is a top-level key (no leading whitespace)
							currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
							if len(currentIndent) == 0 && !strings.HasPrefix(trimmed, "#") {
								// Found next top-level key
								insertIndex = i
								break
							}
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

					codemodsLog.Printf("Added sandbox.agent: %s", sandboxAgentValue)
				} else {
					// Just append at the end
					frontmatterLines = append(frontmatterLines, sandboxLines...)
					codemodsLog.Printf("Added sandbox.agent: %s at end", sandboxAgentValue)
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
			codemodsLog.Printf("Applied network.firewall migration (firewall: %v -> sandbox.agent: %s)", firewallValue, sandboxAgentValue)
			return newContent, true, nil
		},
	}
}

// getCommandToSlashCommandCodemod creates a codemod for migrating on.command to on.slash_command
func getCommandToSlashCommandCodemod() Codemod {
	return Codemod{
		ID:           "command-to-slash-command-migration",
		Name:         "Migrate on.command to on.slash_command",
		Description:  "Replaces deprecated 'on.command' field with 'on.slash_command'",
		IntroducedIn: "0.2.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if on.command exists
			onValue, hasOn := frontmatter["on"]
			if !hasOn {
				return content, false, nil
			}

			onMap, ok := onValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if command field exists in on
			_, hasCommand := onMap["command"]
			if !hasCommand {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Find and replace the command line within the on: block
			var modified bool
			var inOnBlock bool
			var onIndent string

			frontmatterLines := make([]string, len(result.FrontmatterLines))

			for i, line := range result.FrontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the on block
				if strings.HasPrefix(trimmedLine, "on:") {
					inOnBlock = true
					onIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					frontmatterLines[i] = line
					continue
				}

				// Check if we've left the on block (new top-level key with same or less indentation)
				if inOnBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					if len(currentIndent) <= len(onIndent) && strings.Contains(line, ":") {
						inOnBlock = false
					}
				}

				// Replace command with slash_command if in on block
				if inOnBlock && strings.HasPrefix(trimmedLine, "command:") {
					// Preserve indentation
					leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

					// Extract the value and any trailing comment
					parts := strings.SplitN(line, ":", 2)
					if len(parts) >= 2 {
						valueAndComment := parts[1]
						frontmatterLines[i] = fmt.Sprintf("%sslash_command:%s", leadingSpace, valueAndComment)
						modified = true
						codemodsLog.Printf("Replaced on.command with on.slash_command on line %d", i+1)
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
			codemodsLog.Print("Applied on.command to on.slash_command migration")
			return newContent, true, nil
		},
	}
}

// getSafeInputsModeCodemod creates a codemod for removing the deprecated safe-inputs.mode field
func getSafeInputsModeCodemod() Codemod {
	return Codemod{
		ID:           "safe-inputs-mode-removal",
		Name:         "Remove deprecated safe-inputs.mode field",
		Description:  "Removes the deprecated 'safe-inputs.mode' field (HTTP is now the only supported mode)",
		IntroducedIn: "0.2.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if safe-inputs.mode exists
			safeInputsValue, hasSafeInputs := frontmatter["safe-inputs"]
			if !hasSafeInputs {
				return content, false, nil
			}

			safeInputsMap, ok := safeInputsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if mode field exists in safe-inputs
			_, hasMode := safeInputsMap["mode"]
			if !hasMode {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Find and remove the mode line within the safe-inputs block
			var modified bool
			var inSafeInputsBlock bool
			var safeInputsIndent string

			frontmatterLines := make([]string, 0, len(result.FrontmatterLines))

			for i, line := range result.FrontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the safe-inputs block
				if strings.HasPrefix(trimmedLine, "safe-inputs:") {
					inSafeInputsBlock = true
					safeInputsIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					frontmatterLines = append(frontmatterLines, line)
					continue
				}

				// Check if we've left the safe-inputs block (new top-level key with same or less indentation)
				if inSafeInputsBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					if len(currentIndent) <= len(safeInputsIndent) && strings.Contains(line, ":") {
						inSafeInputsBlock = false
					}
				}

				// Remove mode line if in safe-inputs block
				if inSafeInputsBlock && strings.HasPrefix(trimmedLine, "mode:") {
					modified = true
					codemodsLog.Printf("Removed safe-inputs.mode on line %d", i+1)
					continue
				}

				frontmatterLines = append(frontmatterLines, line)
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
			codemodsLog.Print("Applied safe-inputs.mode removal")
			return newContent, true, nil
		},
	}
}

// getUploadAssetsCodemod creates a codemod for migrating upload-assets to upload-asset (plural to singular)
func getUploadAssetsCodemod() Codemod {
	return Codemod{
		ID:           "upload-assets-to-upload-asset-migration",
		Name:         "Migrate upload-assets to upload-asset",
		Description:  "Replaces deprecated 'safe-outputs.upload-assets' field with 'safe-outputs.upload-asset' (plural to singular)",
		IntroducedIn: "0.3.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if safe-outputs.upload-assets exists
			safeOutputsValue, hasSafeOutputs := frontmatter["safe-outputs"]
			if !hasSafeOutputs {
				return content, false, nil
			}

			safeOutputsMap, ok := safeOutputsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if upload-assets field exists in safe-outputs (plural is deprecated)
			_, hasUploadAssets := safeOutputsMap["upload-assets"]
			if !hasUploadAssets {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Find and replace upload-assets with upload-asset within the safe-outputs block
			var modified bool
			var inSafeOutputsBlock bool
			var safeOutputsIndent string

			frontmatterLines := make([]string, len(result.FrontmatterLines))

			for i, line := range result.FrontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the safe-outputs block
				if strings.HasPrefix(trimmedLine, "safe-outputs:") {
					inSafeOutputsBlock = true
					safeOutputsIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					frontmatterLines[i] = line
					continue
				}

				// Check if we've left the safe-outputs block (new top-level key with same or less indentation)
				if inSafeOutputsBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					if len(currentIndent) <= len(safeOutputsIndent) && strings.Contains(line, ":") {
						inSafeOutputsBlock = false
					}
				}

				// Replace upload-assets with upload-asset if in safe-outputs block
				if inSafeOutputsBlock && strings.HasPrefix(trimmedLine, "upload-assets:") {
					// Preserve indentation
					leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

					// Extract the value and any trailing comment
					parts := strings.SplitN(line, ":", 2)
					if len(parts) >= 2 {
						valueAndComment := parts[1]
						frontmatterLines[i] = fmt.Sprintf("%supload-asset:%s", leadingSpace, valueAndComment)
						modified = true
						codemodsLog.Printf("Replaced safe-outputs.upload-assets with safe-outputs.upload-asset on line %d", i+1)
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
			codemodsLog.Print("Applied upload-assets to upload-asset migration")
			return newContent, true, nil
		},
	}
}

// getWritePermissionsCodemod creates a codemod for converting write permissions to read
func getWritePermissionsCodemod() Codemod {
	return Codemod{
		ID:           "write-permissions-to-read-migration",
		Name:         "Convert write permissions to read",
		Description:  "Converts all write permissions to read permissions to comply with the new security policy",
		IntroducedIn: "0.4.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if permissions exist
			permissionsValue, hasPermissions := frontmatter["permissions"]
			if !hasPermissions {
				return content, false, nil
			}

			// Check if any write permissions exist
			hasWritePermissions := false

			// Handle string shorthand (write-all, write)
			if strValue, ok := permissionsValue.(string); ok {
				if strValue == "write-all" || strValue == "write" {
					hasWritePermissions = true
				}
			}

			// Handle map format
			if mapValue, ok := permissionsValue.(map[string]any); ok {
				for _, value := range mapValue {
					if strValue, ok := value.(string); ok && strValue == "write" {
						hasWritePermissions = true
						break
					}
				}
			}

			if !hasWritePermissions {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			result, err := parser.ExtractFrontmatterFromContent(content)
			if err != nil {
				return content, false, fmt.Errorf("failed to parse frontmatter: %w", err)
			}

			// Find and replace write permissions
			var modified bool
			var inPermissionsBlock bool
			var permissionsIndent string

			frontmatterLines := make([]string, len(result.FrontmatterLines))

			for i, line := range result.FrontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the permissions block
				if strings.HasPrefix(trimmedLine, "permissions:") {
					inPermissionsBlock = true
					permissionsIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]

					// Handle shorthand on same line: "permissions: write-all" or "permissions: write"
					if strings.Contains(trimmedLine, ": write-all") {
						frontmatterLines[i] = strings.Replace(line, ": write-all", ": read-all", 1)
						modified = true
						codemodsLog.Printf("Replaced permissions: write-all with permissions: read-all on line %d", i+1)
						continue
					} else if strings.Contains(trimmedLine, ": write") && !strings.Contains(trimmedLine, "write-all") {
						frontmatterLines[i] = strings.Replace(line, ": write", ": read", 1)
						modified = true
						codemodsLog.Printf("Replaced permissions: write with permissions: read on line %d", i+1)
						continue
					}

					frontmatterLines[i] = line
					continue
				}

				// Check if we've left the permissions block (new top-level key with same or less indentation)
				if inPermissionsBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					currentIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					if len(currentIndent) <= len(permissionsIndent) && strings.Contains(line, ":") {
						inPermissionsBlock = false
					}
				}

				// Replace write with read if in permissions block
				if inPermissionsBlock && strings.Contains(trimmedLine, ": write") {
					// Preserve indentation and everything else
					// Extract the key, value, and any trailing comment
					parts := strings.SplitN(line, ":", 2)
					if len(parts) >= 2 {
						key := parts[0]
						valueAndComment := parts[1]

						// Replace "write" with "read" in the value part
						newValueAndComment := strings.Replace(valueAndComment, " write", " read", 1)
						frontmatterLines[i] = fmt.Sprintf("%s:%s", key, newValueAndComment)
						modified = true
						codemodsLog.Printf("Replaced write with read on line %d", i+1)
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
			codemodsLog.Print("Applied write permissions to read migration")
			return newContent, true, nil
		},
	}
}
