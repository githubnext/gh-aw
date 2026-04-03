package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var createCodeScanningAlertLog = logger.New("workflow:create_code_scanning_alert")

// CreateCodeScanningAlertsConfig holds configuration for creating repository security advisories (SARIF format) from agent output
type CreateCodeScanningAlertsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Driver               string   `yaml:"driver,omitempty"`        // Driver name for SARIF tool.driver.name field (default: "GitHub Agentic Workflows Security Scanner")
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository code scanning alert creation
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // List of additional repositories in format "owner/repo" that code scanning alerts can be created in
}

// parseCodeScanningAlertsConfig handles create-code-scanning-alert configuration
func (c *Compiler) parseCodeScanningAlertsConfig(outputMap map[string]any) *CreateCodeScanningAlertsConfig {
	if _, exists := outputMap["create-code-scanning-alert"]; !exists {
		return nil
	}

	createCodeScanningAlertLog.Print("Parsing create-code-scanning-alert configuration")
	configData := outputMap["create-code-scanning-alert"]
	securityReportsConfig := &CreateCodeScanningAlertsConfig{}

	if configMap, ok := configData.(map[string]any); ok {
		// Parse driver
		if driver, exists := configMap["driver"]; exists {
			if driverStr, ok := driver.(string); ok {
				securityReportsConfig.Driver = driverStr
				createCodeScanningAlertLog.Printf("Using custom SARIF driver name: %s", driverStr)
			}
		}

		// Parse target-repo
		securityReportsConfig.TargetRepoSlug = parseTargetRepoFromConfig(configMap)
		if securityReportsConfig.TargetRepoSlug != "" {
			createCodeScanningAlertLog.Printf("Target repo for code scanning alerts: %s", securityReportsConfig.TargetRepoSlug)
		}

		// Parse allowed-repos
		securityReportsConfig.AllowedRepos = parseAllowedReposFromConfig(configMap)
		if len(securityReportsConfig.AllowedRepos) > 0 {
			createCodeScanningAlertLog.Printf("Allowed repos for cross-repo alerts: %d configured", len(securityReportsConfig.AllowedRepos))
		}

		// Parse common base fields with default max of 0 (unlimited)
		c.parseBaseSafeOutputConfig(configMap, &securityReportsConfig.BaseSafeOutputConfig, 0)
	} else {
		// If configData is nil or not a map (e.g., "create-code-scanning-alert:" with no value),
		// still set the default max (nil = unlimited)
		createCodeScanningAlertLog.Print("No config map provided, using defaults (unlimited max)")
		securityReportsConfig.Max = nil
	}

	createCodeScanningAlertLog.Printf("Parsed create-code-scanning-alert config: driver=%q, target-repo=%q, allowed-repos=%d",
		securityReportsConfig.Driver, securityReportsConfig.TargetRepoSlug, len(securityReportsConfig.AllowedRepos))
	return securityReportsConfig
}

// buildCodeScanningUploadJob creates a dedicated job that uploads the SARIF file produced by
// the create_code_scanning_alert handler to the GitHub Code Scanning API.
//
// This is a separate job (not a step inside safe_outputs) so that the checkout and SARIF
// upload do not interfere with other safe-output operations running in safe_outputs.
//
// The job:
//   - depends on safe_outputs (needs: [safe_outputs])
//   - runs only when the safe_outputs job exported a SARIF file
//     (if: needs.safe_outputs.outputs.sarif_file != ”)
//   - restores the workspace to the triggering commit via actions/checkout before upload so
//     that github/codeql-action/upload-sarif can resolve the commit reference
//   - uploads the SARIF file with explicit ref/sha to pin the result to the triggering commit
func (c *Compiler) buildCodeScanningUploadJob(data *WorkflowData) (*Job, error) {
	createCodeScanningAlertLog.Print("Building upload_code_scanning_sarif job")

	// Resolve the checkout token: consult the checkout manager for user-configured credentials,
	// then fall back to the PR checkout token chain (safe-outputs token or GITHUB_TOKEN).
	checkoutMgr := NewCheckoutManager(data.CheckoutConfigs)
	restoreToken := c.resolveRestoreCheckoutToken(checkoutMgr, data)

	var steps []string

	// Step 1: Restore workspace to the triggering commit.
	// The safe_outputs job may have checked out a different branch (e.g., the base branch for
	// a PR) which would leave HEAD pointing at a different commit. The SARIF upload action
	// requires HEAD to match the commit being scanned, otherwise it fails with "commit not found".
	steps = append(steps, "      - name: Restore checkout to triggering commit\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          ref: ${{ github.sha }}\n")
	steps = append(steps, fmt.Sprintf("          token: %s\n", restoreToken))
	steps = append(steps, "          persist-credentials: false\n")
	steps = append(steps, "          fetch-depth: 1\n")

	// Step 2: Upload SARIF file to GitHub Code Scanning.
	// The sarif_file path is passed from the safe_outputs job via job outputs.
	steps = append(steps, "      - name: Upload SARIF to GitHub Code Scanning\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", constants.UploadCodeScanningJobName))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("github/codeql-action/upload-sarif")))
	steps = append(steps, "        with:\n")
	// NOTE: github/codeql-action/upload-sarif uses 'token' as the input name, not 'github-token'
	c.addUploadSARIFToken(&steps, data, data.SafeOutputs.CreateCodeScanningAlerts.GitHubToken)
	// sarif_file is passed from safe_outputs job output (set by create_code_scanning_alert.cjs handler)
	steps = append(steps, fmt.Sprintf("          sarif_file: ${{ needs.%s.outputs.sarif_file }}\n", constants.SafeOutputsJobName))
	// ref and sha pin the upload to the exact triggering commit regardless of local git state
	steps = append(steps, "          ref: ${{ github.ref }}\n")
	steps = append(steps, "          sha: ${{ github.sha }}\n")
	steps = append(steps, "          wait-for-processing: true\n")

	// The job only runs when the safe_outputs job exported a non-empty SARIF file path.
	jobCondition := fmt.Sprintf("needs.%s.outputs.sarif_file != ''", constants.SafeOutputsJobName)

	// Permissions: contents:read to checkout, security-events:write to upload SARIF
	permissions := NewPermissionsContentsReadSecurityEventsWrite()

	job := &Job{
		Name:           string(constants.UploadCodeScanningJobName),
		If:             jobCondition,
		RunsOn:         c.formatFrameworkJobRunsOn(data),
		Environment:    c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions:    permissions.RenderToYAML(),
		TimeoutMinutes: 10,
		Steps:          steps,
		Needs:          []string{string(constants.SafeOutputsJobName)},
	}

	createCodeScanningAlertLog.Print("Built upload_code_scanning_sarif job")
	return job, nil
}

// addUploadSARIFToken adds the 'token' input for github/codeql-action/upload-sarif.
// This action uses 'token' as the input name (not 'github-token' like other GitHub Actions).
// Uses precedence: config token > safe-outputs global github-token > GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
func (c *Compiler) addUploadSARIFToken(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// If app is configured, use app token
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubApp != nil {
		*steps = append(*steps, "          token: ${{ steps.safe-outputs-app-token.outputs.token }}\n")
		return
	}

	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	effectiveToken := getEffectiveSafeOutputGitHubToken(effectiveCustomToken)
	// Log which token source is being used for debugging
	tokenSource := "default (GH_AW_GITHUB_TOKEN || GITHUB_TOKEN)"
	if configToken != "" {
		tokenSource = "per-config github-token"
	} else if safeOutputsToken != "" {
		tokenSource = "safe-outputs github-token"
	}
	createCodeScanningAlertLog.Printf("Using token for SARIF upload from source: %s (upload-sarif uses 'token' not 'github-token')", tokenSource)
	*steps = append(*steps, fmt.Sprintf("          token: %s\n", effectiveToken))
}

// resolveRestoreCheckoutToken returns the GitHub token to use for the pre-SARIF
// restore checkout step. It consults the checkout manager first (user-configured
// checkout frontmatter), then falls back to the PR checkout token chain.
//
// Token precedence:
//  1. User-configured checkout.github-token from the default workspace checkout
//  2. PR checkout token (create-pull-request / push-to-pull-request-branch / safe-outputs token)
//  3. Default: GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
func (c *Compiler) resolveRestoreCheckoutToken(checkoutMgr *CheckoutManager, data *WorkflowData) string {
	// User-configured checkout token from frontmatter (checkout: github-token: ...)
	override := checkoutMgr.GetDefaultCheckoutOverride()
	if override != nil && override.token != "" {
		createCodeScanningAlertLog.Printf("Using checkout manager override token for SARIF restore checkout")
		return getEffectiveSafeOutputGitHubToken(override.token)
	}

	// Fall back to PR checkout token (which itself falls back to safe-outputs token or GITHUB_TOKEN)
	prToken, _ := computeEffectivePRCheckoutToken(data.SafeOutputs)
	return prToken
}
