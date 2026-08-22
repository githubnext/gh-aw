package workflow

// WorkflowFile represents a GitHub Actions workflow file. It is a compatibility
// fixture used to read and structurally validate existing/generated
// copilot-setup-steps.yml content in tests; it is not used to author the
// generated scaffold, which is emitted as hand-formatted YAML text so that
// its comments and layout stay stable.
type WorkflowFile struct {
	Name string                     `yaml:"name,omitempty"`
	On   any                        `yaml:"on,omitempty"`
	Jobs map[string]WorkflowFileJob `yaml:"jobs,omitempty"`
}

// WorkflowFileJob represents a GitHub Actions workflow job in a workflow file.
type WorkflowFileJob struct {
	RunsOn      any                      `yaml:"runs-on,omitempty"`
	Permissions *WorkflowFilePermissions `yaml:"permissions,omitempty"`
	Steps       []WorkflowStep           `yaml:"steps,omitempty"`
}

// WorkflowFilePermissions represents a GitHub Actions permissions block, which may be
// written as a shorthand scalar ("read-all", "write-all", "none") or as a map of scope
// names to access levels (e.g. "contents: read").
type WorkflowFilePermissions struct {
	Shorthand string
	Scopes    map[string]string
}

// UnmarshalYAML accepts either a shorthand scalar string or a map of scope to level.
func (p *WorkflowFilePermissions) UnmarshalYAML(unmarshal func(any) error) error {
	var scalar string
	if err := unmarshal(&scalar); err == nil {
		p.Shorthand = scalar
		p.Scopes = nil
		return nil
	}
	var scopes map[string]string
	if err := unmarshal(&scopes); err != nil {
		return err
	}
	p.Shorthand = ""
	p.Scopes = scopes
	return nil
}

// MarshalYAML emits the shorthand scalar when set, otherwise the scope map.
func (p WorkflowFilePermissions) MarshalYAML() (any, error) {
	if p.Shorthand != "" {
		return p.Shorthand, nil
	}
	return p.Scopes, nil
}
