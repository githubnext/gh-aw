package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var compilerYamlLog = logger.New("workflow:compiler_yaml")

// buildJobsAndValidate builds all workflow jobs and validates their dependencies.
// It resets the job manager, builds jobs from the workflow data, and performs
// dependency and duplicate step validation.
func (c *Compiler) buildJobsAndValidate(data *WorkflowData, markdownPath string) error {
	compilerYamlLog.Printf("Building and validating jobs for workflow: %s", data.Name)

	// Reset job manager for this compilation
	c.jobManager = NewJobManager()

	// Build all jobs
	if err := c.buildJobs(data, markdownPath); err != nil {
		compilerYamlLog.Printf("Failed to build jobs: %v", err)
		return fmt.Errorf("failed to build jobs: %w", err)
	}

	compilerYamlLog.Printf("Built %d jobs successfully", len(c.jobManager.GetAllJobs()))

	// Validate job dependencies
	if err := c.jobManager.ValidateDependencies(); err != nil {
		return fmt.Errorf("job dependency validation failed: %w", err)
	}

	// Validate no duplicate steps within jobs (compiler bug detection)
	if err := c.jobManager.ValidateDuplicateSteps(); err != nil {
		return fmt.Errorf("duplicate step validation failed: %w", err)
	}

	return nil
}

// generateWorkflowBody generates the main workflow structure including name, triggers,
// permissions, concurrency, run-name, environment variables, cache comments, and jobs.
func (c *Compiler) generateWorkflowBody(yaml *strings.Builder, data *WorkflowData) {
	// Write basic workflow structure
	fmt.Fprintf(yaml, "name: \"%s\"\n", data.Name)

	// Inject on.workflow_call.outputs when workflow_call is configured and safe-outputs are present
	onSection := data.On
	if data.SafeOutputs != nil {
		onSection = c.injectWorkflowCallOutputs(onSection, data.SafeOutputs)
	}
	// Inject aw_context input into workflow_dispatch triggers so dispatched workflows
	// can receive caller metadata (repo, run_id, actor, etc.) from dispatch_workflow.
	// String-based injection preserves existing YAML comments and formatting.
	onSection = injectAwContextIntoOnYAML(onSection)
	onSection = injectNetworkAllowedIntoOnYAML(onSection, data.NetworkPermissions)
	onSection = UnquoteYAMLTopLevelKey(onSection, "on")
	yaml.WriteString(onSection)
	yaml.WriteString("\n\n")

	// Note: GitHub Actions doesn't support workflow-level if conditions
	// The workflow_run safety check is added to individual jobs instead

	// Always write empty permissions at the top level
	// Agent permissions are applied only to the agent job
	yaml.WriteString("permissions: {}\n\n")

	yaml.WriteString(data.Concurrency)
	yaml.WriteString("\n\n")
	yaml.WriteString(data.RunName)
	yaml.WriteString("\n\n")

	// Add env section if present
	if data.Env != "" {
		yaml.WriteString(data.Env)
		yaml.WriteString("\n\n")
	}

	// Add cache comment if cache configuration was provided
	if data.Cache != "" {
		yaml.WriteString("# Cache configuration from frontmatter was processed and added to the main job steps\n\n")
	}

	// Generate jobs section using JobManager — write directly to avoid an
	// intermediate string allocation.
	c.jobManager.WriteJobsYAML(yaml)
}

func (c *Compiler) generateYAML(data *WorkflowData, markdownPath string) (string, []string, []string, error) {
	compilerYamlLog.Printf("Generating YAML for workflow: %s", data.Name)
	frontmatterHash, bodyHash, err := c.computeWorkflowHashes(data, markdownPath)
	if err != nil {
		return "", nil, nil, err
	}
	data.FrontmatterHash = frontmatterHash
	if err := c.buildJobsAndValidate(data, markdownPath); err != nil {
		return "", nil, nil, fmt.Errorf("failed to build and validate jobs: %w", err)
	}
	yamlContent, secrets, actions := c.buildFinalWorkflowYAML(data, frontmatterHash, bodyHash)
	yamlContent = c.finalizeGeneratedYAML(yamlContent, data)

	compilerYamlLog.Printf("Successfully generated YAML for workflow: %s (%d bytes)", data.Name, len(yamlContent))
	return yamlContent, secrets, actions, nil
}

func (c *Compiler) computeWorkflowHashes(data *WorkflowData, markdownPath string) (string, string, error) {
	if markdownPath == "" {
		return "", "", nil
	}
	baseDir := filepath.Dir(markdownPath)
	cache := parser.NewImportCache(baseDir)
	hash, err := c.computeWorkflowHashValue(data, markdownPath, func() (string, error) {
		return parser.ComputeFrontmatterHashFromParsedContent(data.FrontmatterYAML, data.RawMarkdown, data.RawFrontmatter, baseDir, cache, parser.DefaultFileReader)
	}, func() (string, error) {
		return parser.ComputeFrontmatterHashFromFileWithParsedFrontmatter(markdownPath, data.RawFrontmatter, cache, parser.DefaultFileReader)
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate workflow YAML: could not compute stable frontmatter hash for %q: %w", markdownPath, err)
	}
	bodyHash, bodyErr := c.computeWorkflowHashValue(data, markdownPath, func() (string, error) {
		return parser.ComputeBodyHashFromParsedContent(data.RawMarkdown, data.FrontmatterYAML, baseDir, parser.DefaultFileReader)
	}, func() (string, error) {
		return parser.ComputeBodyHashFromFile(markdownPath)
	})
	if bodyErr != nil {
		compilerYamlLog.Printf("Warning: could not compute body hash for %q: %v", markdownPath, bodyErr)
		bodyHash = ""
	}
	return hash, bodyHash, nil
}

func (c *Compiler) computeWorkflowHashValue(data *WorkflowData, markdownPath string, fromParsed func() (string, error), fromFile func() (string, error)) (string, error) {
	if data.RawMarkdown != "" {
		return fromParsed()
	}
	compilerYamlLog.Printf("RawMarkdown not set; falling back to reading file from disk: %s", markdownPath)
	return fromFile()
}

func (c *Compiler) buildFinalWorkflowYAML(data *WorkflowData, frontmatterHash, bodyHash string) (string, []string, []string) {
	const initialBuilderCapacity = 96 * 1024
	bodyContent, secrets, actions := c.renderWorkflowBodyWithMetadata(data, initialBuilderCapacity)
	var yaml strings.Builder
	yaml.Grow(initialBuilderCapacity)
	c.generateWorkflowHeader(&yaml, data, frontmatterHash, bodyHash, secrets, actions)
	yaml.WriteString(bodyContent)
	return yaml.String(), secrets, actions
}

func (c *Compiler) renderWorkflowBodyWithMetadata(data *WorkflowData, capacity int) (string, []string, []string) {
	bodyContent := c.renderWorkflowBody(data, capacity)
	secrets := CollectSecretReferences(bodyContent)
	actions := CollectActionReferences(bodyContent)
	if hasWorkflowCallTrigger(data.On) && len(secrets) > 0 {
		updatedOn := injectWorkflowCallSecretsSection(data.On, secrets)
		if updatedOn != data.On {
			data.On = updatedOn
			bodyContent = c.renderWorkflowBody(data, capacity)
			compilerYamlLog.Printf("Regenerated workflow body with on.workflow_call.secrets declarations")
		}
	}
	return bodyContent, secrets, actions
}

func (c *Compiler) renderWorkflowBody(data *WorkflowData, capacity int) string {
	var body strings.Builder
	body.Grow(capacity)
	c.generateWorkflowBody(&body, data)
	return body.String()
}

func (c *Compiler) finalizeGeneratedYAML(yamlContent string, data *WorkflowData) string {
	if c.trialMode && c.hasIssueTrigger(data.On) {
		compilerYamlLog.Print("Trial mode enabled, replacing issue number references")
		yamlContent = c.replaceIssueNumberReferences(yamlContent)
	}
	return normalizeBlankLines(yamlContent)
}
