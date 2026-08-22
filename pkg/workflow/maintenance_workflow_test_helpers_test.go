//go:build !integration

package workflow

type maintenanceWorkflowCallOutput struct {
	Value string `yaml:"value"`
}

type maintenanceWorkflowCall struct {
	Outputs struct {
		AppliedRunURL maintenanceWorkflowCallOutput `yaml:"applied_run_url"`
	} `yaml:"outputs"`
}

type maintenanceWorkflowDocument struct {
	On struct {
		WorkflowCall maintenanceWorkflowCall `yaml:"workflow_call"`
	} `yaml:"on"`
}
