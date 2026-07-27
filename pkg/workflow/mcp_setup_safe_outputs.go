package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var safeOutputsSetupLog = logger.New("workflow:mcp_setup_safe_outputs")

// safeOutputsSecretEnvPrefix is prepended to secret names when generating step env var names for
// safe-outputs config placeholders. The prefix avoids accidental collisions between a workflow
// secret name and a pre-existing step env var (e.g. a secret named DEBUG or
// GH_AW_SAFE_OUTPUTS_CONFIG_PATH would silently override those step vars without the prefix).
// The prefixed env vars are written into the step env: block and resolved in memory at runtime
// by the JavaScript safe-outputs loader (resolveEnvPlaceholders in safe_outputs_config.cjs).
const safeOutputsSecretEnvPrefix = "GH_AW_SECRET_"

func generateSafeOutputsSetup(c *Compiler, yaml *strings.Builder, safeOutputConfig string, workflowData *WorkflowData) {
	if !HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		return
	}
	safeOutputsSetupLog.Printf("Generating safe outputs setup: configLen=%d", len(safeOutputConfig))
	yaml.WriteString("      - name: Generate Safe Outputs Config\n")
	sanitizedConfig, envKeys, envValues := buildSafeOutputsConfigRuntimeData(safeOutputConfig)
	if len(envKeys) > 0 {
		safeOutputsSetupLog.Printf("Safe outputs config: envVars=%d", len(envKeys))
		yaml.WriteString("        env:\n")
		writeStepEnvVars(yaml, envKeys, envValues)
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs\"\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/safeoutputs\n")
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UploadArtifact != nil {
		yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts\"\n")
	}
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UploadAssets != nil {
		yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs/assets\"\n")
	}

	delimiter := GenerateHeredocDelimiterFromContent("SAFE_OUTPUTS_CONFIG", sanitizedConfig)
	if safeOutputConfig != "" {
		yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/safeoutputs/config.json\" << '" + delimiter + "'\n")
		yaml.WriteString("          " + sanitizedConfig + "\n")
		yaml.WriteString("          " + delimiter + "\n")
	}

	toolsMetaJSON, err := generateToolsMetaJSON(workflowData, c.markdownPath)
	if err != nil {
		mcpSetupGeneratorLog.Printf("Error generating tools meta JSON: %v", err)
		toolsMetaJSON = `{"description_suffixes":{},"repo_params":{},"dynamic_tools":[]}`
	}

	var enabledTypes []string
	if safeOutputConfig != "" {
		var configMap map[string]any
		if err := json.Unmarshal([]byte(safeOutputConfig), &configMap); err == nil {
			for typeName := range configMap {
				enabledTypes = append(enabledTypes, typeName)
			}
		}
	}
	// Propagate mentions config to the collection pass so that allowed @-mentions
	// (e.g. "@copilot") are not backtick-escaped before publish-side handlers run.
	var mentionsBlock map[string]any
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.Mentions != nil {
		mentionsBlock = buildMentionsHandlerConfig(workflowData.SafeOutputs.Mentions)
	}
	validationConfigJSON, err := GetValidationConfigJSON(enabledTypes, mentionsBlock)
	if err != nil {
		mcpSetupGeneratorLog.Printf("CRITICAL: Error generating validation config JSON: %v - validation will not work correctly", err)
		validationConfigJSON = "{}"
	}

	yaml.WriteString("      - name: Generate Safe Outputs Tools\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_TOOLS_META_JSON: |\n")
	for line := range strings.SplitSeq(toolsMetaJSON, "\n") {
		yaml.WriteString("            " + line + "\n")
	}
	yaml.WriteString("          GH_AW_VALIDATION_JSON: |\n")
	for line := range strings.SplitSeq(validationConfigJSON, "\n") {
		yaml.WriteString("            " + line + "\n")
	}
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", workflowData))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString(generateGitHubScriptWithRequire("generate_safe_outputs_tools.cjs"))
}

func buildSafeOutputsConfigRuntimeEnvVars(safeOutputConfig string) ([]string, map[string]string) {
	configSecrets := ExtractSecretsFromValue(safeOutputConfig)
	configContextVars := ExtractGitHubContextExpressionsFromValue(safeOutputConfig)
	configWorkflowInputs := ExtractWorkflowInputExpressionsFromValue(safeOutputConfig)
	envValues := make(map[string]string, safeAllocationCapacity(len(configSecrets), len(configContextVars), len(configWorkflowInputs)))
	addEnvValue := func(key, value string) {
		envValues[key] = value
	}
	for k, v := range configWorkflowInputs {
		addEnvValue(k, v)
	}
	for k, v := range configContextVars {
		addEnvValue(k, v)
	}
	for k, v := range configSecrets {
		// Prefix secret env vars to avoid colliding with reserved/known step env var names.
		addEnvValue(safeOutputsSecretEnvPrefix+k, v)
	}
	return sliceutil.SortedKeys(envValues), envValues
}

func buildSafeOutputsConfigRuntimeData(safeOutputConfig string) (string, []string, map[string]string) {
	sanitizedConfig := safeOutputConfig
	envKeys, envValues := buildSafeOutputsConfigRuntimeEnvVars(safeOutputConfig)
	safeOutputsSetupLog.Printf("Building safe outputs config runtime data: envKeys=%d", len(envKeys))
	for _, varName := range envKeys {
		value := envValues[varName]
		sanitizedConfig = strings.ReplaceAll(sanitizedConfig, value, "${"+varName+"}")
	}
	return sanitizedConfig, envKeys, envValues
}

func writeStepEnvVars(yaml *strings.Builder, envKeys []string, envValues map[string]string) {
	for _, varName := range envKeys {
		yaml.WriteString("          " + varName + ": " + envValues[varName] + "\n")
	}
}
