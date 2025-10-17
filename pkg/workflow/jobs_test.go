package workflow

import (
	"strings"
	"testing"
)

func TestJobManager_AddJob(t *testing.T) {
	jm := NewJobManager()

	tests := []struct {
		name    string
		job     *Job
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid job",
			job: &Job{
				Name:   "test-job",
				RunsOn: "ubuntu-latest",
			},
			wantErr: false,
		},
		{
			name: "empty job name",
			job: &Job{
				Name:   "",
				RunsOn: "ubuntu-latest",
			},
			wantErr: true,
			errMsg:  "job name cannot be empty",
		},
		{
			name: "duplicate job name",
			job: &Job{
				Name:   "test-job", // Same name as first test
				RunsOn: "windows-latest",
			},
			wantErr: true,
			errMsg:  "job 'test-job' already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := jm.AddJob(tt.job)
			if tt.wantErr {
				if err == nil {
					t.Errorf("AddJob() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("AddJob() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("AddJob() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestJobManager_ValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		jobs    []*Job
		wantErr bool
		errMsg  string
	}{
		{
			name: "no dependencies",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest"},
				{Name: "job2", RunsOn: "ubuntu-latest"},
			},
			wantErr: false,
		},
		{
			name: "valid dependencies",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest"},
				{Name: "job2", RunsOn: "ubuntu-latest", Needs: []string{"job1"}},
				{Name: "job3", RunsOn: "ubuntu-latest", Needs: []string{"job1", "job2"}},
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest"},
				{Name: "job2", RunsOn: "ubuntu-latest", Needs: []string{"nonexistent"}},
			},
			wantErr: true,
			errMsg:  "depends on non-existent job 'nonexistent'",
		},
		{
			name: "simple cycle",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest", Needs: []string{"job2"}},
				{Name: "job2", RunsOn: "ubuntu-latest", Needs: []string{"job1"}},
			},
			wantErr: true,
			errMsg:  "cycle detected",
		},
		{
			name: "complex cycle",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest", Needs: []string{"job2"}},
				{Name: "job2", RunsOn: "ubuntu-latest", Needs: []string{"job3"}},
				{Name: "job3", RunsOn: "ubuntu-latest", Needs: []string{"job1"}},
			},
			wantErr: true,
			errMsg:  "cycle detected",
		},
		{
			name: "self-dependency cycle",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest", Needs: []string{"job1"}},
			},
			wantErr: true,
			errMsg:  "cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			for _, job := range tt.jobs {
				if err := jm.AddJob(job); err != nil {
					t.Fatalf("Failed to add job %s: %v", job.Name, err)
				}
			}

			err := jm.ValidateDependencies()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateDependencies() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateDependencies() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDependencies() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestJobManager_GetTopologicalOrder(t *testing.T) {
	tests := []struct {
		name     string
		jobs     []*Job
		expected []string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "no dependencies",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest"},
				{Name: "job2", RunsOn: "ubuntu-latest"},
			},
			expected: []string{"job1", "job2"}, // Alphabetical order
			wantErr:  false,
		},
		{
			name: "linear dependencies",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest"},
				{Name: "job2", RunsOn: "ubuntu-latest", Needs: []string{"job1"}},
				{Name: "job3", RunsOn: "ubuntu-latest", Needs: []string{"job2"}},
			},
			expected: []string{"job1", "job2", "job3"},
			wantErr:  false,
		},
		{
			name: "complex dependencies",
			jobs: []*Job{
				{Name: "build", RunsOn: "ubuntu-latest"},
				{Name: "test", RunsOn: "ubuntu-latest", Needs: []string{"build"}},
				{Name: "lint", RunsOn: "ubuntu-latest", Needs: []string{"build"}},
				{Name: "deploy", RunsOn: "ubuntu-latest", Needs: []string{"test", "lint"}},
			},
			expected: []string{"build", "lint", "test", "deploy"}, // build first, then lint/test (alphabetical), then deploy
			wantErr:  false,
		},
		{
			name: "cycle should error",
			jobs: []*Job{
				{Name: "job1", RunsOn: "ubuntu-latest", Needs: []string{"job2"}},
				{Name: "job2", RunsOn: "ubuntu-latest", Needs: []string{"job1"}},
			},
			wantErr: true,
			errMsg:  "cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			for _, job := range tt.jobs {
				if err := jm.AddJob(job); err != nil {
					t.Fatalf("Failed to add job %s: %v", job.Name, err)
				}
			}

			result, err := jm.GetTopologicalOrder()
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTopologicalOrder() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("GetTopologicalOrder() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetTopologicalOrder() unexpected error = %v", err)
					return
				}
				if len(result) != len(tt.expected) {
					t.Errorf("GetTopologicalOrder() length = %d, want %d", len(result), len(tt.expected))
					return
				}
				for i, expected := range tt.expected {
					if result[i] != expected {
						t.Errorf("GetTopologicalOrder()[%d] = %s, want %s", i, result[i], expected)
					}
				}
			}
		})
	}
}

func TestJobManager_RenderToYAML(t *testing.T) {
	tests := []struct {
		name     string
		jobs     []*Job
		expected []string // Strings that should be present in the output
	}{
		{
			name: "empty job manager",
			jobs: []*Job{},
			expected: []string{
				"jobs:",
			},
		},
		{
			name: "single simple job",
			jobs: []*Job{
				{
					Name:   "test-job",
					RunsOn: "runs-on: ubuntu-latest",
					Steps:  []string{"      - name: Test\n        run: echo hello\n"},
				},
			},
			expected: []string{
				"jobs:",
				"  test-job:",
				"    runs-on: ubuntu-latest",
				"    steps:",
				"      - name: Test",
				"        run: echo hello",
			},
		},
		{
			name: "job with dependencies",
			jobs: []*Job{
				{
					Name:   "job1",
					RunsOn: "runs-on: ubuntu-latest",
					Needs:  []string{"job2"},
					Steps:  []string{"      - name: Step1\n        run: echo step1\n"},
				},
				{
					Name:   "job2",
					RunsOn: "runs-on: ubuntu-latest",
					Steps:  []string{"      - name: Step2\n        run: echo step2\n"},
				},
			},
			expected: []string{
				"jobs:",
				"  job1:",
				"    needs: job2",
				"    runs-on: ubuntu-latest",
				"  job2:",
				"    runs-on: ubuntu-latest",
			},
		},
		{
			name: "job with multiple dependencies",
			jobs: []*Job{
				{
					Name:   "deploy",
					RunsOn: "runs-on: ubuntu-latest",
					Needs:  []string{"build", "test"},
					Steps:  []string{"      - name: Deploy\n        run: echo deploy\n"},
				},
			},
			expected: []string{
				"jobs:",
				"  deploy:",
				"    needs:",
				"      - build",
				"      - test",
				"    runs-on: ubuntu-latest",
			},
		},
		{
			name: "job with if condition",
			jobs: []*Job{
				{
					Name:   "conditional-job",
					RunsOn: "runs-on: ubuntu-latest",
					If:     "github.event_name == 'push'",
					Steps:  []string{"      - name: Conditional Step\n        run: echo conditional\n"},
				},
			},
			expected: []string{
				"jobs:",
				"  conditional-job:",
				"    if: github.event_name == 'push'",
				"    runs-on: ubuntu-latest",
			},
		},
		{
			name: "job with outputs",
			jobs: []*Job{
				{
					Name:   "output-job",
					RunsOn: "runs-on: ubuntu-latest",
					Outputs: map[string]string{
						"result":  "${{ steps.test.outputs.result }}",
						"version": "${{ steps.version.outputs.version }}",
					},
					Steps: []string{"      - name: Generate Output\n        run: echo output\n"},
				},
			},
			expected: []string{
				"jobs:",
				"  output-job:",
				"    runs-on: ubuntu-latest",
				"    outputs:",
				"      result: ${{ steps.test.outputs.result }}",
				"      version: ${{ steps.version.outputs.version }}",
			},
		},
		{
			name: "jobs sorted alphabetically regardless of insertion order",
			jobs: []*Job{
				{
					Name:   "zebra-job",
					RunsOn: "runs-on: ubuntu-latest",
					Steps:  []string{"      - name: Zebra\n        run: echo zebra\n"},
				},
				{
					Name:   "alpha-job",
					RunsOn: "runs-on: ubuntu-latest",
					Steps:  []string{"      - name: Alpha\n        run: echo alpha\n"},
				},
				{
					Name:   "charlie-job",
					RunsOn: "runs-on: ubuntu-latest",
					Steps:  []string{"      - name: Charlie\n        run: echo charlie\n"},
				},
				{
					Name:   "beta-job",
					RunsOn: "runs-on: ubuntu-latest",
					Steps:  []string{"      - name: Beta\n        run: echo beta\n"},
				},
			},
			expected: []string{
				"jobs:",
				"  alpha-job:",
				"  beta-job:",
				"  charlie-job:",
				"  zebra-job:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			for _, job := range tt.jobs {
				if err := jm.AddJob(job); err != nil {
					t.Fatalf("Failed to add job %s: %v", job.Name, err)
				}
			}

			result := jm.RenderToYAML()

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("RenderToYAML() result does not contain expected string: %s\nFull result:\n%s", expected, result)
				}
			}
		})
	}
}

func TestJobManager_GetJob(t *testing.T) {
	jm := NewJobManager()

	testJob := &Job{
		Name:   "test-job",
		RunsOn: "ubuntu-latest",
	}

	// Add a job
	err := jm.AddJob(testJob)
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Test retrieving existing job
	retrievedJob, exists := jm.GetJob("test-job")
	if !exists {
		t.Error("Expected job to exist but it doesn't")
	}
	if retrievedJob.Name != testJob.Name {
		t.Errorf("Retrieved job name = %s, want %s", retrievedJob.Name, testJob.Name)
	}

	// Test retrieving non-existent job
	_, exists = jm.GetJob("nonexistent")
	if exists {
		t.Error("Expected job to not exist but it does")
	}
}

func TestJobManager_GetAllJobs(t *testing.T) {
	jm := NewJobManager()

	jobs := []*Job{
		{Name: "job1", RunsOn: "ubuntu-latest"},
		{Name: "job2", RunsOn: "windows-latest"},
	}

	for _, job := range jobs {
		if err := jm.AddJob(job); err != nil {
			t.Fatalf("Failed to add job %s: %v", job.Name, err)
		}
	}

	allJobs := jm.GetAllJobs()

	if len(allJobs) != len(jobs) {
		t.Errorf("GetAllJobs() returned %d jobs, want %d", len(allJobs), len(jobs))
	}

	for _, originalJob := range jobs {
		retrievedJob, exists := allJobs[originalJob.Name]
		if !exists {
			t.Errorf("Job %s not found in GetAllJobs() result", originalJob.Name)
		}
		if retrievedJob.Name != originalJob.Name {
			t.Errorf("Job name mismatch: got %s, want %s", retrievedJob.Name, originalJob.Name)
		}
	}

	// Test that modifying returned map doesn't affect internal state
	allJobs["new-job"] = &Job{Name: "new-job"}

	// Original manager should not be affected
	if _, exists := jm.GetJob("new-job"); exists {
		t.Error("Internal state was modified by external change to GetAllJobs() result")
	}
}

func TestJobManager_GenerateMermaidGraph(t *testing.T) {
	tests := []struct {
		name     string
		jobs     []*Job
		expected []string // Strings that should be present in the Mermaid graph
	}{
		{
			name:     "empty job manager",
			jobs:     []*Job{},
			expected: []string{},
		},
		{
			name: "single job without dependencies",
			jobs: []*Job{
				{
					Name:        "build",
					DisplayName: "Build Project",
					RunsOn:      "ubuntu-latest",
				},
			},
			expected: []string{
				"```mermaid",
				"graph LR",
				"  build[\"Build Project\"]",
				"```",
			},
		},
		{
			name: "single job without display name",
			jobs: []*Job{
				{
					Name:   "test",
					RunsOn: "ubuntu-latest",
				},
			},
			expected: []string{
				"```mermaid",
				"graph LR",
				"  test[\"test\"]",
				"```",
			},
		},
		{
			name: "linear dependency chain",
			jobs: []*Job{
				{Name: "build", DisplayName: "Build", RunsOn: "ubuntu-latest"},
				{Name: "test", DisplayName: "Test", RunsOn: "ubuntu-latest", Needs: []string{"build"}},
				{Name: "deploy", DisplayName: "Deploy", RunsOn: "ubuntu-latest", Needs: []string{"test"}},
			},
			expected: []string{
				"```mermaid",
				"graph LR",
				"  build[\"Build\"]",
				"  test[\"Test\"]",
				"  deploy[\"Deploy\"]",
				"  build --> test",
				"  test --> deploy",
				"```",
			},
		},
		{
			name: "multiple dependencies",
			jobs: []*Job{
				{Name: "build", DisplayName: "Build", RunsOn: "ubuntu-latest"},
				{Name: "test", DisplayName: "Test", RunsOn: "ubuntu-latest", Needs: []string{"build"}},
				{Name: "lint", DisplayName: "Lint", RunsOn: "ubuntu-latest", Needs: []string{"build"}},
				{Name: "deploy", DisplayName: "Deploy", RunsOn: "ubuntu-latest", Needs: []string{"test", "lint"}},
			},
			expected: []string{
				"```mermaid",
				"graph LR",
				"  build[\"Build\"]",
				"  test[\"Test\"]",
				"  lint[\"Lint\"]",
				"  deploy[\"Deploy\"]",
				"  build --> test",
				"  build --> lint",
				"  test --> deploy",
				"  lint --> deploy",
				"```",
			},
		},
		{
			name: "job with quotes in display name",
			jobs: []*Job{
				{Name: "build", DisplayName: "Build \"Production\"", RunsOn: "ubuntu-latest"},
			},
			expected: []string{
				"```mermaid",
				"graph LR",
				"  build[\"Build \\\"Production\\\"\"]",
				"```",
			},
		},
		{
			name: "nodes sorted alphabetically regardless of insertion order",
			jobs: []*Job{
				{Name: "zebra", DisplayName: "Zebra Job", RunsOn: "ubuntu-latest"},
				{Name: "apple", DisplayName: "Apple Job", RunsOn: "ubuntu-latest"},
				{Name: "banana", DisplayName: "Banana Job", RunsOn: "ubuntu-latest", Needs: []string{"apple"}},
			},
			expected: []string{
				"```mermaid",
				"graph LR",
				"  apple[\"Apple Job\"]",
				"  banana[\"Banana Job\"]",
				"  zebra[\"Zebra Job\"]",
				"  apple --> banana",
				"```",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			for _, job := range tt.jobs {
				if err := jm.AddJob(job); err != nil {
					t.Fatalf("Failed to add job %s: %v", job.Name, err)
				}
			}

			result := jm.GenerateMermaidGraph()

			// For empty job manager, expect empty string
			if len(tt.jobs) == 0 {
				if result != "" {
					t.Errorf("GenerateMermaidGraph() for empty manager = %q, want empty string", result)
				}
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("GenerateMermaidGraph() result does not contain expected string: %s\nFull result:\n%s", expected, result)
				}
			}
		})
	}
}
