---
title: AgenticImport
description: Efficiently migrate and adapt agentic workflows between repositories using AI-assisted copying and conversion
sidebar:
  order: 3
---

AgenticImport is a pattern for efficiently migrating agentic workflows between repositories. Unlike traditional workflow reuse through imports or the `gh aw add` command, AgenticImport uses an AI agent to copy and adapt workflows from one repository to another, handling necessary modifications during the migration process.

## When to Use AgenticImport

Use AgenticImport when code reuse is not the goal and you need a modified copy of a workflow:

- **Repository forking** - Create adapted versions of workflows for different projects
- **Context-specific customization** - Migrate workflows that need substantial changes for a new environment
- **Organizational boundaries** - Copy workflows across organizations where shared imports aren't practical
- **Learning and experimentation** - Fork workflows as starting points for new automation
- **One-time migrations** - Move workflows between repositories without maintaining shared dependencies

**When NOT to use AgenticImport:**

If you need synchronized updates across repositories, use [Packaging & Distribution](/gh-aw/guides/packaging-imports/) with `gh aw add` and `gh aw update` commands instead. AgenticImport creates independent copies, not linked workflows.

## How It Works

AgenticImport combines the `create-agentic-agent` custom agent with the compiler's adaptation capabilities:

```text
┌─────────────────────┐
│  Source Repository  │
│  - release.md       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  AI Agent Analysis  │
│  - Read workflow    │
│  - Understand logic │
│  - Identify deps    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Target Repository  │
│  - release.md       │
│  (adapted version)  │
└─────────────────────┘
```

The agent:
1. Reads the source workflow and understands its purpose
2. Identifies repository-specific configuration and dependencies
3. Adapts frontmatter, imports, and instructions for the target repository
4. Creates or updates the workflow file in the target repository
5. Validates compilation and suggests fixes for any issues

## Basic Example: Migrating a Release Workflow

The `release.md` workflow was migrated from `githubnext/gh-aw` to `githubnext/gh-aw-mcpg` using AgenticImport:

```yaml wrap title="Prompt for create-agentic-agent"
Migrate the release.md workflow from githubnext/gh-aw to this repository.

The workflow should:
- Keep the same release automation logic
- Adapt permissions and jobs for our repository structure
- Update any repository-specific references
- Ensure all imports and tools are compatible
```

The agent analyzes the source workflow, identifies what needs to change (repository names, paths, permissions), and creates an adapted version that compiles cleanly in the target repository.

## Using create-agentic-agent

The `create-agentic-agent` is a specialized agent for workflow creation and migration. Set it up with `gh aw init`:

```bash
gh aw init
```

This creates `.github/agents/create-agentic-workflow.md` and related agents that understand workflow structure, frontmatter configuration, and compilation requirements.

### Migration Workflow

1. **Identify source workflow**: Find the workflow you want to migrate
2. **Analyze requirements**: Understand what the workflow does and its dependencies
3. **Create task**: Use `create-agentic-agent` to request migration
4. **Review output**: Check the generated workflow file
5. **Test compilation**: Run `gh aw compile` to validate
6. **Iterate if needed**: Refine the migration based on compilation feedback

## AgenticImport vs Traditional Import

| Approach | Use Case | Maintains Link | Effort |
|----------|----------|---------------|--------|
| **gh aw add** | Shared workflows with updates | ✅ Yes | Low |
| **imports:** | Reusable components | ✅ Yes | Low |
| **AgenticImport** | One-time migration with adaptation | ❌ No | Medium |
| **Manual copy** | Simple workflows | ❌ No | High |

**AgenticImport advantages:**
- Handles complex adaptations automatically
- Understands workflow semantics and dependencies
- Can merge multiple sources or split workflows
- Validates compilation during migration
- Provides explanations of changes made

**Traditional import advantages:**
- Synchronized updates across repositories
- Lower initial setup overhead
- Explicit version management
- Clear dependency tracking

## Advanced Patterns

### Merging Multiple Workflows

Combine workflows from different sources:

```markdown
Merge the issue-triage workflow from org/repo1 and the 
label-management workflow from org/repo2 into a single 
unified workflow that handles both responsibilities.
```

The agent analyzes both workflows, identifies overlapping configuration, and creates a merged workflow with compatible frontmatter and combined instructions.

### Splitting Complex Workflows

Break down monolithic workflows:

```markdown
Split the multi-responsibility workflow.md into separate 
workflows for:
- Issue triage (issue-triage.md)
- PR review (pr-review.md)
- Security scanning (security-scan.md)

Extract shared configuration into common imports.
```

### Cross-Organization Migration

Migrate workflows across organizational boundaries:

```markdown
Migrate the ci-doctor workflow from public-org/workflows
to our private enterprise repository, adapting for our
internal MCP servers and security policies.
```

## Real-World Example: release.md Migration

The `release.md` workflow migration from `githubnext/gh-aw` to `githubnext/gh-aw-mcpg` demonstrates AgenticImport in practice:

**Source workflow** (gh-aw):
- Multi-job release pipeline
- GitHub Actions-based binary building
- SBOM generation
- Release note automation
- Repository-specific permissions

**Adapted workflow** (gh-aw-mcpg):
- Same release automation logic
- Adjusted permissions for different repository
- Updated repository references
- Compatible with target repository structure
- Preserved core functionality

The migration was completed through agent-assisted copying and adaptation, handling necessary modifications while maintaining the workflow's purpose.

## Best Practices

### Before Migration

- **Read the source workflow** - Understand what it does and why
- **Check dependencies** - Identify required secrets, tools, and permissions
- **Review imports** - Understand what shared files are needed
- **Plan adaptations** - Know what needs to change for target repository

### During Migration

- **Provide context** - Give the agent information about target repository requirements
- **Specify constraints** - Mention security policies, tool restrictions, or style preferences
- **Be specific** - Clear instructions produce better adaptations
- **Test incrementally** - Compile and validate after each migration step

### After Migration

- **Validate compilation** - Run `gh aw compile` to ensure workflow is valid
- **Review changes** - Understand what the agent modified and why
- **Test execution** - Run the workflow in a safe environment first
- **Document adaptations** - Note what differs from the source for future reference
- **Remove source link** - Delete or update `source:` frontmatter field if present

## Testing Migrated Workflows

Use [TrialOps](/gh-aw/guides/trialops/) to test migrated workflows safely:

```yaml wrap title=".github/workflows/migrated-workflow.md"
---
on:
  workflow_dispatch:
engine: copilot
trial:
  target-repo: "my-org/trial-testing"
safe-outputs:
  create-issue:
---

# Migrated Workflow Test

Test the migrated workflow in a trial repository before 
deploying to production.
```

## Workflow Authoring Assistance

The `create-agentic-agent` provides conversational workflow authoring:

```bash
# Create or update workflows interactively
gh copilot agent --agent create-agentic-workflow

# Debug compilation issues
gh copilot agent --agent debug-agentic-workflow
```

These agents understand:
- Frontmatter schema and validation rules
- Tool and MCP server configuration
- Safe output types and parameters
- GitHub Actions compatibility
- Common workflow patterns

## Limitations

**Not for continuous synchronization**: AgenticImport creates point-in-time copies. For ongoing updates, use `gh aw add` with version tracking.

**Requires understanding**: The agent needs clear instructions about target repository context and requirements.

**Manual validation needed**: Always review and test migrated workflows before production deployment.

**Compilation errors possible**: Complex workflows may need iteration to adapt successfully.

## Related Patterns

- **[Packaging & Distribution](/gh-aw/guides/packaging-imports/)** - Shared workflows with version management
- **[Deterministic & Agentic Patterns](/gh-aw/guides/deterministic-agentic-patterns/)** - Combining AI and deterministic steps
- **[Custom Agents](/gh-aw/reference/custom-agents/)** - Creating specialized workflow authoring agents
- **[TrialOps](/gh-aw/guides/trialops/)** - Safe testing of workflows

## Related Documentation

- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Understanding workflow format
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options
- [Compilation Process](/gh-aw/reference/compilation-process/) - How workflows are validated
- [CLI Commands](/gh-aw/setup/cli/) - Available workflow management commands
