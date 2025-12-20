package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var vscodeConfigLog = logger.New("cli:vscode_config")

// VSCodeSettings represents the structure of .vscode/settings.json
type VSCodeSettings struct {
	YAMLSchemas map[string]any `json:"yaml.schemas,omitempty"`
	// Include other commonly used settings as any to preserve them
	Other map[string]any `json:"-"`
}

// VSCodeExtensions represents the structure of .vscode/extensions.json
type VSCodeExtensions struct {
	Recommendations []string `json:"recommendations"`
}

// UnmarshalJSON custom unmarshaler for VSCodeSettings to preserve unknown fields
func (s *VSCodeSettings) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map to capture all fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract yaml.schemas if it exists
	if yamlSchemas, ok := raw["yaml.schemas"]; ok {
		if schemas, ok := yamlSchemas.(map[string]any); ok {
			s.YAMLSchemas = schemas
		}
		delete(raw, "yaml.schemas")
	}

	// Initialize Other if nil and store remaining fields
	if s.Other == nil {
		s.Other = make(map[string]any)
	}
	for k, v := range raw {
		s.Other[k] = v
	}
	
	return nil
}

// MarshalJSON custom marshaler for VSCodeSettings to include all fields
func (s VSCodeSettings) MarshalJSON() ([]byte, error) {
	// Create a map with all fields
	result := make(map[string]any)

	// Add all other fields first
	for k, v := range s.Other {
		result[k] = v
	}

	// Add yaml.schemas if present
	if len(s.YAMLSchemas) > 0 {
		result["yaml.schemas"] = s.YAMLSchemas
	}

	return json.Marshal(result)
}

// ensureWorkflowSchema writes the main workflow schema to .github/aw/main_workflow_schema.json
func ensureWorkflowSchema(verbose bool) error {
	vscodeConfigLog.Print("Writing main workflow schema to .github/aw/")

	// Create .github/aw directory if it doesn't exist
	awDir := filepath.Join(".github", "aw")
	if err := os.MkdirAll(awDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/aw directory: %w", err)
	}
	vscodeConfigLog.Printf("Ensured directory exists: %s", awDir)

	schemaPath := filepath.Join(awDir, "main_workflow_schema.json")

	// Get the embedded schema from parser package
	schemaContent := parser.GetMainWorkflowSchema()

	// Write schema file
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("failed to write workflow schema: %w", err)
	}
	vscodeConfigLog.Printf("Wrote schema to: %s", schemaPath)

	if verbose {
		fmt.Fprintf(os.Stderr, "Created workflow schema at %s\n", schemaPath)
	}

	return nil
}

// ensureVSCodeSettings creates or updates .vscode/settings.json with YAML schema configuration
func ensureVSCodeSettings(verbose bool) error {
	vscodeConfigLog.Print("Creating or updating .vscode/settings.json")

	// Create .vscode directory if it doesn't exist
	vscodeDir := ".vscode"
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}
	vscodeConfigLog.Printf("Ensured directory exists: %s", vscodeDir)

	settingsPath := filepath.Join(vscodeDir, "settings.json")

	// Read existing settings if they exist
	var settings VSCodeSettings
	if data, err := os.ReadFile(settingsPath); err == nil {
		vscodeConfigLog.Printf("Reading existing settings from: %s", settingsPath)
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse existing settings.json: %w", err)
		}
	} else {
		vscodeConfigLog.Print("No existing settings found, creating new one")
		settings.Other = make(map[string]any)
	}

	// Initialize yaml.schemas if it doesn't exist
	if settings.YAMLSchemas == nil {
		settings.YAMLSchemas = make(map[string]any)
	}

	// Add schema mapping for workflow files
	schemaPath := "./.github/aw/main_workflow_schema.json"
	workflowPattern := ".github/workflows/*.md"

	// Check if already configured
	if existingPattern, exists := settings.YAMLSchemas[schemaPath]; exists {
		if existingPattern == workflowPattern {
			vscodeConfigLog.Print("Schema mapping already configured, skipping update")
			if verbose {
				fmt.Fprintf(os.Stderr, "YAML schema mapping already configured in %s\n", settingsPath)
			}
			return nil
		}
	}

	settings.YAMLSchemas[schemaPath] = workflowPattern

	// Ensure yaml validation settings are enabled
	if settings.Other == nil {
		settings.Other = make(map[string]any)
	}
	if _, exists := settings.Other["yaml.validate"]; !exists {
		settings.Other["yaml.validate"] = true
	}
	if _, exists := settings.Other["yaml.hover"]; !exists {
		settings.Other["yaml.hover"] = true
	}
	if _, exists := settings.Other["yaml.completion"]; !exists {
		settings.Other["yaml.completion"] = true
	}

	// Write settings file with proper indentation
	data, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings.json: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}
	vscodeConfigLog.Printf("Wrote settings to: %s", settingsPath)

	return nil
}

// ensureVSCodeExtensions creates or updates .vscode/extensions.json with RedHat YAML extension
func ensureVSCodeExtensions(verbose bool) error {
	vscodeConfigLog.Print("Creating or updating .vscode/extensions.json")

	// Create .vscode directory if it doesn't exist
	vscodeDir := ".vscode"
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	extensionsPath := filepath.Join(vscodeDir, "extensions.json")

	// Read existing extensions if they exist
	var extensions VSCodeExtensions
	if data, err := os.ReadFile(extensionsPath); err == nil {
		vscodeConfigLog.Printf("Reading existing extensions from: %s", extensionsPath)
		if err := json.Unmarshal(data, &extensions); err != nil {
			return fmt.Errorf("failed to parse existing extensions.json: %w", err)
		}
	} else {
		vscodeConfigLog.Print("No existing extensions file found, creating new one")
		extensions.Recommendations = []string{}
	}

	// Check if RedHat YAML extension is already in recommendations
	redhatYAMLExt := "redhat.vscode-yaml"
	hasYAMLExt := false
	for _, ext := range extensions.Recommendations {
		if ext == redhatYAMLExt {
			hasYAMLExt = true
			break
		}
	}

	if hasYAMLExt {
		vscodeConfigLog.Print("RedHat YAML extension already in recommendations, skipping update")
		if verbose {
			fmt.Fprintf(os.Stderr, "RedHat YAML extension already in %s\n", extensionsPath)
		}
		return nil
	}

	// Add RedHat YAML extension to recommendations
	extensions.Recommendations = append(extensions.Recommendations, redhatYAMLExt)
	vscodeConfigLog.Printf("Added %s to recommendations", redhatYAMLExt)

	// Write extensions file with proper indentation
	data, err := json.MarshalIndent(extensions, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal extensions.json: %w", err)
	}

	if err := os.WriteFile(extensionsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write extensions.json: %w", err)
	}
	vscodeConfigLog.Printf("Wrote extensions to: %s", extensionsPath)

	return nil
}
