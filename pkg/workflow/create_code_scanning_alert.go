package workflow

import (
	"fmt"
)

// CreateCodeScanningAlertsConfig holds configuration for creating repository security advisories (SARIF format) from agent output
type CreateCodeScanningAlertsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Driver               string `yaml:"driver,omitempty"` // Driver name for SARIF tool.driver.name field (default: "GitHub Agentic Workflows Security Scanner")
}

// buildCreateOutputCodeScanningAlertJob creates the create_code_scanning_alert job
func (c *Compiler) buildCreateOutputCodeScanningAlertJob(data *WorkflowData, mainJobName string, workflowFilename string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateCodeScanningAlerts == nil {
		return nil, fmt.Errorf("safe-outputs.create-code-scanning-alert configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create Code Scanning Alert\n")
	steps = append(steps, "        id: create_code_scanning_alert\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the max configuration
	if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_SECURITY_REPORT_MAX: %d\n", data.SafeOutputs.CreateCodeScanningAlerts.Max))
	}
	// Pass the driver configuration, defaulting to frontmatter name
	driverName := data.SafeOutputs.CreateCodeScanningAlerts.Driver
	if driverName == "" {
		if data.FrontmatterName != "" {
			driverName = data.FrontmatterName
		} else {
			driverName = data.Name // fallback to H1 header name
		}
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_SECURITY_REPORT_DRIVER: %s\n", driverName))
	// Pass the workflow filename for rule ID prefix
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_FILENAME: %s\n", workflowFilename))

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		token = data.SafeOutputs.CreateCodeScanningAlerts.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createCodeScanningAlertScript)
	steps = append(steps, formattedScript...)

	// Add step to upload SARIF artifact
	steps = append(steps, "      - name: Upload SARIF artifact\n")
	steps = append(steps, "        if: steps.create_code_scanning_alert.outputs.sarif_file\n")
	steps = append(steps, "        uses: actions/upload-artifact@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: code-scanning-alert.sarif\n")
	steps = append(steps, "          path: ${{ steps.create_code_scanning_alert.outputs.sarif_file }}\n")

	// Add step to upload SARIF to GitHub Code Scanning
	steps = append(steps, "      - name: Upload SARIF to GitHub Security\n")
	steps = append(steps, "        if: steps.create_code_scanning_alert.outputs.sarif_file\n")
	steps = append(steps, "        uses: github/codeql-action/upload-sarif@v3\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          sarif_file: ${{ steps.create_code_scanning_alert.outputs.sarif_file }}\n")

	// Create outputs for the job
	outputs := map[string]string{
		"sarif_file":        "${{ steps.create_code_scanning_alert.outputs.sarif_file }}",
		"findings_count":    "${{ steps.create_code_scanning_alert.outputs.findings_count }}",
		"artifact_uploaded": "${{ steps.create_code_scanning_alert.outputs.artifact_uploaded }}",
		"codeql_uploaded":   "${{ steps.create_code_scanning_alert.outputs.codeql_uploaded }}",
	}

	// When min > 0, skip the contains check to allow the job to run even with 0 outputs
	skipContains := data.SafeOutputs.CreateCodeScanningAlerts.Min > 0
	jobCondition := BuildSafeOutputType("create-code-scanning-alert", skipContains).Render()

	job := &Job{
		Name:           "create_code_scanning_alert",
		If:             jobCondition,
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      security-events: write\n      actions: read", // Need security-events:write for SARIF upload
		TimeoutMinutes: 10,                                                                                      // 10-minute timeout
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseCodeScanningAlertsConfig handles create-code-scanning-alert configuration
func (c *Compiler) parseCodeScanningAlertsConfig(outputMap map[string]any) *CreateCodeScanningAlertsConfig {
	if _, exists := outputMap["create-code-scanning-alert"]; !exists {
		return nil
	}

	configData := outputMap["create-code-scanning-alert"]
	securityReportsConfig := &CreateCodeScanningAlertsConfig{}
	securityReportsConfig.Max = 0 // Default max is 0 (unlimited)

	if configMap, ok := configData.(map[string]any); ok {
		// Parse common base fields
		c.parseBaseSafeOutputConfig(configMap, &securityReportsConfig.BaseSafeOutputConfig)

		// Parse driver
		if driver, exists := configMap["driver"]; exists {
			if driverStr, ok := driver.(string); ok {
				securityReportsConfig.Driver = driverStr
			}
		}
	}

	return securityReportsConfig
}
