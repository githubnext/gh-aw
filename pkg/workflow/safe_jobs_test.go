package workflow

import (
	"strings"
	"testing"
)

func TestParseSafeJobsConfig(t *testing.T) {
	c := NewCompiler(false, "", "test")
	
	// Test basic safe-jobs configuration
	outputMap := map[string]any{
		"safe-jobs": map[string]any{
			"deploy": map[string]any{
				"runs-on": "ubuntu-latest",
				"if":      "github.event.issue.number",
				"needs":   []any{"task"},
				"env": map[string]any{
					"DEPLOY_ENV": "production",
				},
				"permissions": map[string]any{
					"contents": "write",
					"issues":   "read",
				},
				"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				"inputs": map[string]any{
					"environment": map[string]any{
						"description": "Target deployment environment",
						"required":    true,
						"type":        "choice",
						"options":     []any{"staging", "production"},
					},
					"force": map[string]any{
						"description": "Force deployment even if tests fail",
						"required":    false,
						"type":        "boolean",
						"default":     "false",
					},
				},
				"steps": []any{
					map[string]any{
						"name": "Deploy application",
						"run":  "echo 'Deploying to ${{ inputs.environment }}'",
					},
				},
			},
		},
	}
	
	result := c.parseSafeJobsConfig(outputMap)
	
	if result == nil {
		t.Fatal("Expected safe-jobs config to be parsed, got nil")
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 safe job, got %d", len(result))
	}
	
	deployJob, exists := result["deploy"]
	if !exists {
		t.Fatal("Expected 'deploy' job to exist")
	}
	
	// Test runs-on
	if deployJob.RunsOn != "ubuntu-latest" {
		t.Errorf("Expected runs-on to be 'ubuntu-latest', got %v", deployJob.RunsOn)
	}
	
	// Test if condition
	if deployJob.If != "github.event.issue.number" {
		t.Errorf("Expected if condition to be 'github.event.issue.number', got %s", deployJob.If)
	}
	
	// Test needs
	if len(deployJob.Needs) != 1 || deployJob.Needs[0] != "task" {
		t.Errorf("Expected needs to be ['task'], got %v", deployJob.Needs)
	}
	
	// Test env
	if len(deployJob.Env) != 1 || deployJob.Env["DEPLOY_ENV"] != "production" {
		t.Errorf("Expected env to contain DEPLOY_ENV=production, got %v", deployJob.Env)
	}
	
	// Test permissions
	if len(deployJob.Permissions) != 2 || deployJob.Permissions["contents"] != "write" || deployJob.Permissions["issues"] != "read" {
		t.Errorf("Expected specific permissions, got %v", deployJob.Permissions)
	}
	
	// Test github-token
	if deployJob.GitHubToken != "${{ secrets.CUSTOM_TOKEN }}" {
		t.Errorf("Expected github-token to be '${{ secrets.CUSTOM_TOKEN }}', got %s", deployJob.GitHubToken)
	}
	
	// Test inputs
	if len(deployJob.Inputs) != 2 {
		t.Fatalf("Expected 2 inputs, got %d", len(deployJob.Inputs))
	}
	
	envInput, exists := deployJob.Inputs["environment"]
	if !exists {
		t.Fatal("Expected 'environment' input to exist")
	}
	
	if envInput.Description != "Target deployment environment" {
		t.Errorf("Expected environment input description, got %s", envInput.Description)
	}
	
	if !envInput.Required {
		t.Error("Expected environment input to be required")
	}
	
	if envInput.Type != "choice" {
		t.Errorf("Expected environment input type to be 'choice', got %s", envInput.Type)
	}
	
	if len(envInput.Options) != 2 || envInput.Options[0] != "staging" || envInput.Options[1] != "production" {
		t.Errorf("Expected environment input options to be ['staging', 'production'], got %v", envInput.Options)
	}
	
	forceInput, exists := deployJob.Inputs["force"]
	if !exists {
		t.Fatal("Expected 'force' input to exist")
	}
	
	if forceInput.Required {
		t.Error("Expected force input to not be required")
	}
	
	if forceInput.Type != "boolean" {
		t.Errorf("Expected force input type to be 'boolean', got %s", forceInput.Type)
	}
	
	if forceInput.Default != "false" {
		t.Errorf("Expected force input default to be 'false', got %s", forceInput.Default)
	}
	
	// Test steps
	if len(deployJob.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(deployJob.Steps))
	}
}

func TestHasSafeOutputsEnabledWithSafeJobs(t *testing.T) {
	// Test that safe-jobs are detected by HasSafeOutputsEnabled
	config := &SafeOutputsConfig{
		SafeJobs: map[string]*SafeJobConfig{
			"deploy": &SafeJobConfig{
				RunsOn: "ubuntu-latest",
			},
		},
	}
	
	if !HasSafeOutputsEnabled(config) {
		t.Error("Expected HasSafeOutputsEnabled to return true when safe-jobs are configured")
	}
	
	// Test combined with other safe-outputs
	config.CreateIssues = &CreateIssuesConfig{}
	
	if !HasSafeOutputsEnabled(config) {
		t.Error("Expected HasSafeOutputsEnabled to return true with both safe-jobs and create-issues")
	}
}

func TestBuildSafeJobs(t *testing.T) {
	c := NewCompiler(false, "", "test")
	
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			SafeJobs: map[string]*SafeJobConfig{
				"deploy": &SafeJobConfig{
					RunsOn: "ubuntu-latest",
					If:     "github.event.issue.number",
					Env: map[string]string{
						"DEPLOY_ENV": "production",
					},
					Inputs: map[string]*SafeJobInput{
						"environment": &SafeJobInput{
							Description: "Target deployment environment",
							Required:    true,
							Type:        "choice",
							Options:     []string{"staging", "production"},
						},
					},
					Steps: []any{
						map[string]any{
							"name": "Deploy",
							"run":  "echo 'Deploying'",
						},
					},
				},
			},
			Env: map[string]string{
				"GLOBAL_VAR": "global_value",
			},
		},
	}
	
	err := c.buildSafeJobs(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building safe jobs: %v", err)
	}
	
	jobs := c.jobManager.GetAllJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job to be created, got %d", len(jobs))
	}
	
	var job *Job
	for _, j := range jobs {
		job = j
		break
	}
	
	// Check job name
	if job.Name != "safe_job_deploy" {
		t.Errorf("Expected job name to be 'safe_job_deploy', got %s", job.Name)
	}
	
	// Check dependencies - should include main job and any additional needs
	expectedNeeds := []string{"main_job"}
	if len(job.Needs) != len(expectedNeeds) {
		t.Errorf("Expected needs %v, got %v", expectedNeeds, job.Needs)
	}
	
	// Check if condition
	if job.If != "github.event.issue.number" {
		t.Errorf("Expected if condition to be 'github.event.issue.number', got %s", job.If)
	}
	
	// Check runs-on
	if job.RunsOn != "runs-on: ubuntu-latest" {
		t.Errorf("Expected runs-on to be 'runs-on: ubuntu-latest', got %s", job.RunsOn)
	}
	
	// Check that steps were generated
	if len(job.Steps) == 0 {
		t.Error("Expected steps to be generated")
	}
	
	// Check that environment setup step includes input variables
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "SAFE_JOB_ENVIRONMENT") {
		t.Error("Expected environment variable for input 'environment' to be set")
	}
	
	if !strings.Contains(stepsContent, "GITHUB_AW_AGENT_OUTPUT") {
		t.Error("Expected main job output to be available as environment variable")
	}
	
	if !strings.Contains(stepsContent, "GLOBAL_VAR=global_value") {
		t.Error("Expected global environment variables from safe-outputs.env to be set")
	}
}

func TestBuildSafeJobsWithNoConfiguration(t *testing.T) {
	c := NewCompiler(false, "", "test")
	
	// Test with no SafeOutputs
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	
	err := c.buildSafeJobs(workflowData, "main_job")
	if err != nil {
		t.Errorf("Expected no error with no safe-outputs, got %v", err)
	}
	
	// Test with SafeOutputs but no SafeJobs
	workflowData.SafeOutputs = &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{},
	}
	
	err = c.buildSafeJobs(workflowData, "main_job")
	if err != nil {
		t.Errorf("Expected no error with safe-outputs but no safe-jobs, got %v", err)
	}
	
	jobs := c.jobManager.GetAllJobs()
	if len(jobs) != 0 {
		t.Errorf("Expected no jobs to be created, got %d", len(jobs))
	}
}

func TestSafeJobsInSafeOutputsConfig(t *testing.T) {
	c := NewCompiler(false, "", "test")
	
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			SafeJobs: map[string]*SafeJobConfig{
				"deploy": &SafeJobConfig{
					Inputs: map[string]*SafeJobInput{
						"environment": &SafeJobInput{
							Description: "Target deployment environment",
							Required:    true,
							Type:        "choice",
							Options:     []string{"staging", "production"},
						},
					},
				},
				"notify": &SafeJobConfig{
					Inputs: map[string]*SafeJobInput{
						"message": &SafeJobInput{
							Description: "Notification message",
							Required:    false,
							Type:        "string",
							Default:     "Deployment completed",
						},
					},
				},
			},
		},
	}
	
	configJSON := c.generateSafeOutputsConfig(workflowData)
	
	if configJSON == "" {
		t.Fatal("Expected safe-outputs config JSON to be generated")
	}
	
	// Should contain both safe jobs
	if !strings.Contains(configJSON, "deploy") {
		t.Error("Expected config to contain 'deploy' job")
	}
	
	if !strings.Contains(configJSON, "notify") {
		t.Error("Expected config to contain 'notify' job")
	}
	
	// Should contain input definitions
	if !strings.Contains(configJSON, "environment") {
		t.Error("Expected config to contain 'environment' input")
	}
	
	if !strings.Contains(configJSON, "message") {
		t.Error("Expected config to contain 'message' input")
	}
}