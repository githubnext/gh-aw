---
"gh-aw": major
---

Share workflow run flags through an embedded `cli.WorkflowRunFlags` in `cli.RunOptions` and `cli.RunWorkflowOptions`, as requested by issue #67198.

**Breaking Go API change**: Existing keyed literals that initialize shared flags directly on either outer struct no longer compile. Go does not allow promoted fields as composite literal keys.

**Migration guide:**
- Replace `cli.RunOptions{RepoOverride: "owner/repo", Verbose: true}` with `cli.RunOptions{WorkflowRunFlags: cli.WorkflowRunFlags{RepoOverride: "owner/repo", Verbose: true}}`.
- Replace `cli.RunWorkflowOptions{WorkflowName: "my-workflow", RepoOverride: "owner/repo", Verbose: true}` with `cli.RunWorkflowOptions{WorkflowName: "my-workflow", WorkflowRunFlags: cli.WorkflowRunFlags{RepoOverride: "owner/repo", Verbose: true}}`.
- Move all shared flags (`Verbose`, `EngineOverride`, `RepoOverride`, `RefOverride`, `AutoMergePRs`, `Push`, `DryRun`, `Approve`) into the embedded value. Keep mode-specific fields on the outer struct.

Field selectors and assignments, zero-value defaults, flat JSON serialization, and CLI flags and runtime behavior remain unchanged. CLI users do not need to migrate. See the [package API migration examples](pkg/cli/README.md#migrating-workflow-run-option-literals).
