package cli

// Shared help text constants for CLI commands

// WorkflowIDExplanation describes what a workflow-id is and accepted formats.
// This text is shared across multiple commands to ensure consistency.
const WorkflowIDExplanation = `The workflow-id is the basename of the Markdown file without the .md extension.
You can provide either the workflow-id (e.g., 'ci-doctor') or the full filename (e.g., 'ci-doctor.md').`
