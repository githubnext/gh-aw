package workflow

// WorkflowFile represents a GitHub Actions workflow file.
type WorkflowFile struct {
	Name string                     `yaml:"name,omitempty"`
	On   any                        `yaml:"on,omitempty"`
	Jobs map[string]WorkflowFileJob `yaml:"jobs,omitempty"`
}

// WorkflowFileJob represents a GitHub Actions workflow job in a workflow file.
type WorkflowFileJob struct {
	RunsOn      any            `yaml:"runs-on,omitempty"`
	Permissions map[string]any `yaml:"permissions,omitempty"`
	Steps       []WorkflowStep `yaml:"steps,omitempty"`
}
