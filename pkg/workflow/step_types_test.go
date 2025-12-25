package workflow

import (
	"testing"
)

func TestWorkflowStep_IsUsesStep(t *testing.T) {
	tests := []struct {
		name string
		step *WorkflowStep
		want bool
	}{
		{
			name: "step with uses field",
			step: &WorkflowStep{Uses: "actions/checkout@v4"},
			want: true,
		},
		{
			name: "step with run field only",
			step: &WorkflowStep{Run: "echo hello"},
			want: false,
		},
		{
			name: "empty step",
			step: &WorkflowStep{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.IsUsesStep(); got != tt.want {
				t.Errorf("WorkflowStep.IsUsesStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkflowStep_IsRunStep(t *testing.T) {
	tests := []struct {
		name string
		step *WorkflowStep
		want bool
	}{
		{
			name: "step with run field",
			step: &WorkflowStep{Run: "echo hello"},
			want: true,
		},
		{
			name: "step with uses field only",
			step: &WorkflowStep{Uses: "actions/checkout@v4"},
			want: false,
		},
		{
			name: "empty step",
			step: &WorkflowStep{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.IsRunStep(); got != tt.want {
				t.Errorf("WorkflowStep.IsRunStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkflowStep_ToMap(t *testing.T) {
	tests := []struct {
		name string
		step *WorkflowStep
		want map[string]any
	}{
		{
			name: "complete step with uses",
			step: &WorkflowStep{
				Name: "Checkout code",
				ID:   "checkout",
				Uses: "actions/checkout@v4",
				With: map[string]any{"fetch-depth": "0"},
			},
			want: map[string]any{
				"name": "Checkout code",
				"id":   "checkout",
				"uses": "actions/checkout@v4",
				"with": map[string]any{"fetch-depth": "0"},
			},
		},
		{
			name: "step with run",
			step: &WorkflowStep{
				Name:  "Run tests",
				Run:   "npm test",
				Shell: "bash",
				Env:   map[string]string{"NODE_ENV": "test"},
			},
			want: map[string]any{
				"name":  "Run tests",
				"run":   "npm test",
				"shell": "bash",
				"env":   map[string]string{"NODE_ENV": "test"},
			},
		},
		{
			name: "step with all fields",
			step: &WorkflowStep{
				Name:             "Complex step",
				ID:               "complex",
				If:               "success()",
				Uses:             "some/action@v1",
				WorkingDirectory: "/path/to/dir",
				With:             map[string]any{"key": "value"},
				Env:              map[string]string{"VAR": "val"},
				ContinueOnError:  true,
				TimeoutMinutes:   10,
			},
			want: map[string]any{
				"name":              "Complex step",
				"id":                "complex",
				"if":                "success()",
				"uses":              "some/action@v1",
				"working-directory": "/path/to/dir",
				"with":              map[string]any{"key": "value"},
				"env":               map[string]string{"VAR": "val"},
				"continue-on-error": true,
				"timeout-minutes":   10,
			},
		},
		{
			name: "minimal step",
			step: &WorkflowStep{
				Uses: "actions/checkout@v4",
			},
			want: map[string]any{
				"uses": "actions/checkout@v4",
			},
		},
		{
			name: "step with string continue-on-error",
			step: &WorkflowStep{
				Name:            "Test step",
				Run:             "npm test",
				ContinueOnError: "false",
			},
			want: map[string]any{
				"name":              "Test step",
				"run":               "npm test",
				"continue-on-error": "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.step.ToMap()
			if len(got) != len(tt.want) {
				t.Errorf("WorkflowStep.ToMap() map length = %d, want %d", len(got), len(tt.want))
			}
			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("WorkflowStep.ToMap() missing key %q", key)
					continue
				}
				// Compare values - for maps, do a deep comparison
				if !compareStepValues(gotVal, wantVal) {
					t.Errorf("WorkflowStep.ToMap()[%q] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestMapToStep(t *testing.T) {
	tests := []struct {
		name    string
		stepMap map[string]any
		want    *WorkflowStep
		wantErr bool
	}{
		{
			name: "complete step with uses",
			stepMap: map[string]any{
				"name": "Checkout code",
				"id":   "checkout",
				"uses": "actions/checkout@v4",
				"with": map[string]any{"fetch-depth": "0"},
			},
			want: &WorkflowStep{
				Name: "Checkout code",
				ID:   "checkout",
				Uses: "actions/checkout@v4",
				With: map[string]any{"fetch-depth": "0"},
			},
			wantErr: false,
		},
		{
			name: "step with run",
			stepMap: map[string]any{
				"name":  "Run tests",
				"run":   "npm test",
				"shell": "bash",
				"env":   map[string]any{"NODE_ENV": "test"},
			},
			want: &WorkflowStep{
				Name:  "Run tests",
				Run:   "npm test",
				Shell: "bash",
				Env:   map[string]string{"NODE_ENV": "test"},
			},
			wantErr: false,
		},
		{
			name: "step with all fields",
			stepMap: map[string]any{
				"name":              "Complex step",
				"id":                "complex",
				"if":                "success()",
				"uses":              "some/action@v1",
				"working-directory": "/path/to/dir",
				"with":              map[string]any{"key": "value"},
				"env":               map[string]any{"VAR": "val"},
				"continue-on-error": true,
				"timeout-minutes":   10,
			},
			want: &WorkflowStep{
				Name:             "Complex step",
				ID:               "complex",
				If:               "success()",
				Uses:             "some/action@v1",
				WorkingDirectory: "/path/to/dir",
				With:             map[string]any{"key": "value"},
				Env:              map[string]string{"VAR": "val"},
				ContinueOnError:  true,
				TimeoutMinutes:   10,
			},
			wantErr: false,
		},
		{
			name:    "nil step map",
			stepMap: nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty step map",
			stepMap: map[string]any{},
			want:    &WorkflowStep{},
			wantErr: false,
		},
		{
			name: "step with string continue-on-error",
			stepMap: map[string]any{
				"name":              "Test step",
				"run":               "npm test",
				"continue-on-error": "false",
			},
			want: &WorkflowStep{
				Name:            "Test step",
				Run:             "npm test",
				ContinueOnError: "false",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapToStep(tt.stepMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapToStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !compareSteps(got, tt.want) {
				t.Errorf("MapToStep() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWorkflowStep_Clone(t *testing.T) {
	original := &WorkflowStep{
		Name:             "Original step",
		ID:               "original",
		If:               "success()",
		Uses:             "some/action@v1",
		Run:              "echo test",
		WorkingDirectory: "/test",
		Shell:            "bash",
		With:             map[string]any{"key": "value", "nested": map[string]any{"inner": "val"}},
		Env:              map[string]string{"VAR1": "val1", "VAR2": "val2"},
		ContinueOnError:  true,
		TimeoutMinutes:   15,
	}

	clone := original.Clone()

	// Verify clone is equal to original
	if !compareSteps(clone, original) {
		t.Errorf("Clone() created unequal step")
	}

	// Verify clone is independent (modifying clone doesn't affect original)
	clone.Name = "Modified"
	if original.Name == "Modified" {
		t.Errorf("Clone() is not independent - modifying clone affected original")
	}

	clone.With["new-key"] = "new-value"
	if _, exists := original.With["new-key"]; exists {
		t.Errorf("Clone() did not deep copy With map")
	}

	clone.Env["NEW_VAR"] = "new-val"
	if _, exists := original.Env["NEW_VAR"]; exists {
		t.Errorf("Clone() did not deep copy Env map")
	}
}

func TestWorkflowStep_ToYAML(t *testing.T) {
	tests := []struct {
		name    string
		step    *WorkflowStep
		wantErr bool
	}{
		{
			name: "simple step",
			step: &WorkflowStep{
				Name: "Test step",
				Uses: "actions/checkout@v4",
			},
			wantErr: false,
		},
		{
			name: "step with complex fields",
			step: &WorkflowStep{
				Name: "Complex step",
				Uses: "some/action@v1",
				With: map[string]any{
					"string-field": "value",
					"int-field":    42,
					"bool-field":   true,
				},
				Env: map[string]string{
					"VAR": "value",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.step.ToYAML()
			if (err != nil) != tt.wantErr {
				t.Errorf("WorkflowStep.ToYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("WorkflowStep.ToYAML() returned empty string")
			}
		})
	}
}

func TestMapToStep_RoundTrip(t *testing.T) {
	// Test that converting map -> step -> map produces the same result
	// Note: env field converts from map[string]any to map[string]string
	originalMap := map[string]any{
		"name": "Test step",
		"id":   "test",
		"uses": "actions/checkout@v4",
		"with": map[string]any{"fetch-depth": "0"},
		"env":  map[string]any{"NODE_ENV": "test"},
	}

	step, err := MapToStep(originalMap)
	if err != nil {
		t.Fatalf("MapToStep() failed: %v", err)
	}

	resultMap := step.ToMap()

	// Compare maps
	if len(resultMap) != len(originalMap) {
		t.Errorf("Round trip changed map size: got %d, want %d", len(resultMap), len(originalMap))
	}

	for key, originalVal := range originalMap {
		resultVal, ok := resultMap[key]
		if !ok {
			t.Errorf("Round trip lost key %q", key)
			continue
		}
		// Special handling for env field which converts from map[string]any to map[string]string
		if key == "env" {
			origEnv, origOK := originalVal.(map[string]any)
			resultEnv, resultOK := resultVal.(map[string]string)
			if origOK && resultOK {
				if len(origEnv) != len(resultEnv) {
					t.Errorf("Round trip changed env map size: got %d, want %d", len(resultEnv), len(origEnv))
				}
				for k, v := range origEnv {
					if vStr, ok := v.(string); ok {
						if resultEnv[k] != vStr {
							t.Errorf("Round trip changed env[%q]: got %q, want %q", k, resultEnv[k], vStr)
						}
					}
				}
				continue
			}
		}
		if !compareStepValues(resultVal, originalVal) {
			t.Errorf("Round trip changed value for key %q: got %v, want %v", key, resultVal, originalVal)
		}
	}
}

// Helper function to compare two values (handles nested maps)
func compareStepValues(a, b any) bool {
	switch aVal := a.(type) {
	case map[string]any:
		bMap, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if len(aVal) != len(bMap) {
			return false
		}
		for k, v := range aVal {
			bv, ok := bMap[k]
			if !ok || !compareStepValues(v, bv) {
				return false
			}
		}
		return true
	case map[string]string:
		bMap, ok := b.(map[string]string)
		if !ok {
			return false
		}
		if len(aVal) != len(bMap) {
			return false
		}
		for k, v := range aVal {
			if bMap[k] != v {
				return false
			}
		}
		return true
	case []string:
		bSlice, ok := b.([]string)
		if !ok {
			return false
		}
		if len(aVal) != len(bSlice) {
			return false
		}
		for i := range aVal {
			if aVal[i] != bSlice[i] {
				return false
			}
		}
		return true
	case []map[string]any:
		bSlice, ok := b.([]map[string]any)
		if !ok {
			return false
		}
		if len(aVal) != len(bSlice) {
			return false
		}
		for i := range aVal {
			if !compareStepValues(aVal[i], bSlice[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// Helper function to compare two WorkflowStep structs
func compareSteps(a, b *WorkflowStep) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Name != b.Name || a.ID != b.ID || a.If != b.If ||
		a.Uses != b.Uses || a.Run != b.Run ||
		a.WorkingDirectory != b.WorkingDirectory || a.Shell != b.Shell ||
		a.TimeoutMinutes != b.TimeoutMinutes {
		return false
	}

	// Compare ContinueOnError (can be any type)
	if !compareStepValues(a.ContinueOnError, b.ContinueOnError) {
		return false
	}

	// Compare With maps
	if !compareStepValues(a.With, b.With) {
		return false
	}

	// Compare Env maps
	if !compareStepValues(a.Env, b.Env) {
		return false
	}

	return true
}

// Tests for WorkflowJob type

func TestWorkflowJob_ToMap(t *testing.T) {
	tests := []struct {
		name string
		job  *WorkflowJob
		want map[string]any
	}{
		{
			name: "simple job with steps",
			job: &WorkflowJob{
				Name:   "Test Job",
				RunsOn: "ubuntu-latest",
				Steps: []WorkflowStep{
					{Name: "Checkout", Uses: "actions/checkout@v4"},
					{Name: "Run tests", Run: "npm test"},
				},
			},
			want: map[string]any{
				"name":    "Test Job",
				"runs-on": "ubuntu-latest",
				"steps": []map[string]any{
					{"name": "Checkout", "uses": "actions/checkout@v4"},
					{"name": "Run tests", "run": "npm test"},
				},
			},
		},
		{
			name: "job with all fields",
			job: &WorkflowJob{
				Name:           "Complex Job",
				RunsOn:         "ubuntu-latest",
				Needs:          []string{"build"},
				If:             "success()",
				TimeoutMinutes: 30,
				Permissions:    map[string]string{"contents": "read"},
				Env:            map[string]string{"NODE_ENV": "test"},
				Steps: []WorkflowStep{
					{Name: "Test", Run: "npm test"},
				},
			},
			want: map[string]any{
				"name":            "Complex Job",
				"runs-on":         "ubuntu-latest",
				"needs":           []string{"build"},
				"if":              "success()",
				"timeout-minutes": 30,
				"permissions":     map[string]string{"contents": "read"},
				"env":             map[string]string{"NODE_ENV": "test"},
				"steps": []map[string]any{
					{"name": "Test", "run": "npm test"},
				},
			},
		},
		{
			name: "reusable workflow job",
			job: &WorkflowJob{
				Uses: "./.github/workflows/reusable.yml",
				With: map[string]any{"config": "test"},
			},
			want: map[string]any{
				"uses": "./.github/workflows/reusable.yml",
				"with": map[string]any{"config": "test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.job.ToMap()
			if !compareJobMaps(got, tt.want) {
				t.Errorf("WorkflowJob.ToMap() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMapToJob(t *testing.T) {
	tests := []struct {
		name    string
		jobMap  map[string]any
		want    *WorkflowJob
		wantErr bool
	}{
		{
			name: "simple job with steps",
			jobMap: map[string]any{
				"name":    "Test Job",
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"name": "Checkout", "uses": "actions/checkout@v4"},
					map[string]any{"name": "Run tests", "run": "npm test"},
				},
			},
			want: &WorkflowJob{
				Name:   "Test Job",
				RunsOn: "ubuntu-latest",
				Steps: []WorkflowStep{
					{Name: "Checkout", Uses: "actions/checkout@v4"},
					{Name: "Run tests", Run: "npm test"},
				},
			},
			wantErr: false,
		},
		{
			name: "job with needs as string",
			jobMap: map[string]any{
				"name":    "Deploy",
				"runs-on": "ubuntu-latest",
				"needs":   "build",
			},
			want: &WorkflowJob{
				Name:   "Deploy",
				RunsOn: "ubuntu-latest",
				Needs:  []string{"build"},
			},
			wantErr: false,
		},
		{
			name: "job with needs as array",
			jobMap: map[string]any{
				"name":    "Deploy",
				"runs-on": "ubuntu-latest",
				"needs":   []any{"build", "test"},
			},
			want: &WorkflowJob{
				Name:   "Deploy",
				RunsOn: "ubuntu-latest",
				Needs:  []string{"build", "test"},
			},
			wantErr: false,
		},
		{
			name:    "nil job map",
			jobMap:  nil,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapToJob(tt.jobMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapToJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !compareJobs(got, tt.want) {
				t.Errorf("MapToJob() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestStepsToAny(t *testing.T) {
	steps := []WorkflowStep{
		{Name: "Step 1", Uses: "actions/checkout@v4"},
		{Name: "Step 2", Run: "echo hello"},
	}

	result := StepsToAny(steps)

	if len(result) != 2 {
		t.Errorf("StepsToAny() length = %d, want 2", len(result))
	}

	// Check first step
	if stepMap, ok := result[0].(map[string]any); ok {
		if stepMap["name"] != "Step 1" {
			t.Errorf("StepsToAny()[0][name] = %v, want Step 1", stepMap["name"])
		}
	} else {
		t.Error("StepsToAny()[0] is not a map[string]any")
	}
}

func TestStepsFromAny(t *testing.T) {
	stepsAny := []any{
		map[string]any{"name": "Step 1", "uses": "actions/checkout@v4"},
		map[string]any{"name": "Step 2", "run": "echo hello"},
	}

	steps, err := StepsFromAny(stepsAny)
	if err != nil {
		t.Fatalf("StepsFromAny() error = %v", err)
	}

	if len(steps) != 2 {
		t.Errorf("StepsFromAny() length = %d, want 2", len(steps))
	}

	if steps[0].Name != "Step 1" {
		t.Errorf("StepsFromAny()[0].Name = %s, want Step 1", steps[0].Name)
	}
	if steps[1].Run != "echo hello" {
		t.Errorf("StepsFromAny()[1].Run = %s, want echo hello", steps[1].Run)
	}
}

// Helper functions for comparing jobs

func compareJobs(a, b *WorkflowJob) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Name != b.Name || a.If != b.If || a.TimeoutMinutes != b.TimeoutMinutes {
		return false
	}

	// Compare RunsOn
	if !compareStepValues(a.RunsOn, b.RunsOn) {
		return false
	}

	// Compare Needs
	if len(a.Needs) != len(b.Needs) {
		return false
	}
	for i := range a.Needs {
		if a.Needs[i] != b.Needs[i] {
			return false
		}
	}

	// Compare Steps
	if len(a.Steps) != len(b.Steps) {
		return false
	}
	for i := range a.Steps {
		if !compareSteps(&a.Steps[i], &b.Steps[i]) {
			return false
		}
	}

	return true
}

func compareJobMaps(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for key, aVal := range a {
		bVal, ok := b[key]
		if !ok {
			return false
		}

		// Special handling for steps (array of maps)
		if key == "steps" {
			aSteps, aOK := aVal.([]map[string]any)
			bSteps, bOK := bVal.([]map[string]any)
			if aOK && bOK {
				if len(aSteps) != len(bSteps) {
					return false
				}
				for i := range aSteps {
					if !compareStepValues(aSteps[i], bSteps[i]) {
						return false
					}
				}
				continue
			}
		}

		if !compareStepValues(aVal, bVal) {
			return false
		}
	}

	return true
}
