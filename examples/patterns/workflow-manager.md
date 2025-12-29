---
# Pattern: Workflow Manager
# Complexity: Advanced
# Use Case: Generate or update agentic workflow files programmatically
name: Workflow Manager
description: Generates and updates agentic workflow files based on requirements
on:
  workflow_dispatch:
    inputs:
      template:
        description: "Workflow template to use"
        required: true
        type: choice
        options:
          - auto-labeler
          - daily-report
          - pr-reviewer
          - custom
permissions:
  contents: write
  pull-requests: write
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [repos, pull_requests]
  bash:
    - "gh *"
    - "git *"
safe-outputs:
  create-pull-request:
    max: 1
timeout-minutes: 15
strict: true
---

# Workflow Manager

Generate or update agentic workflow files from templates, requirements, or existing patterns.

## Use Cases

# TODO: Choose your use case

### 1. Generate from Template

Create new workflow from pattern library:

```markdown
**Input**: Template name + customization parameters
**Process**:
1. Load pattern template
2. Apply customizations (repository name, triggers, etc.)
3. Validate and compile workflow
4. Create PR with new workflow file

**Example**: Generate auto-labeler workflow for your repository
```

### 2. Update Existing Workflows

Bulk update workflows with configuration changes:

```markdown
**Input**: Update specification (e.g., "update all to use Copilot v2")
**Process**:
1. Find all workflow files
2. Apply updates (engine version, permissions, etc.)
3. Validate changes
4. Create PR with updates

**Example**: Update 20 workflows to use new MCP server configuration
```

### 3. Migrate Workflows

Convert workflows to new format or standard:

```markdown
**Input**: Migration rules
**Process**:
1. Parse existing workflows
2. Apply transformation rules
3. Validate migrated workflows
4. Create PR with changes

**Example**: Migrate workflows from old to new safe-outputs format
```

### 4. Generate from Requirements

Create custom workflow from natural language description:

```markdown
**Input**: Workflow requirements in plain English
**Process**:
1. Analyze requirements
2. Select appropriate patterns
3. Generate workflow YAML
4. Validate and compile
5. Create PR with generated workflow

**Example**: "Create a workflow that labels PRs based on file changes"
```

## Implementation Steps

### Step 1: Load Template

```bash
# TODO: Load from pattern library or custom template

TEMPLATE="${{ github.event.inputs.template }}"

if [ "$TEMPLATE" = "auto-labeler" ]; then
  cp examples/patterns/auto-labeler.md /tmp/workflow-template.md
elif [ "$TEMPLATE" = "daily-report" ]; then
  cp examples/patterns/daily-report.md /tmp/workflow-template.md
elif [ "$TEMPLATE" = "pr-reviewer" ]; then
  cp examples/patterns/pr-reviewer.md /tmp/workflow-template.md
else
  echo "Custom template specified"
fi
```

### Step 2: Customize Template

```python
#!/usr/bin/env python3
"""
Workflow Template Customizer
TODO: Customize for your needs
"""
import re

def customize_workflow(template_content, params):
    """Apply customizations to workflow template"""
    
    # Replace repository-specific values
    content = template_content.replace(
        '# TODO: Customize labels',
        f"# Labels for {params['repository']}"
    )
    
    # Customize triggers
    if params.get('trigger') == 'issues':
        content = re.sub(
            r'on:.*?permissions:',
            'on:\n  issues:\n    types: [opened, edited]\npermissions:',
            content,
            flags=re.DOTALL
        )
    
    # Customize safe-outputs
    if params.get('create_issues'):
        content = content.replace(
            '# safe-outputs:',
            'safe-outputs:\n  create-issue:\n    max: 1'
        )
    
    # Update name
    content = content.replace(
        'name: Template Name',
        f"name: {params['workflow_name']}"
    )
    
    return content

# Read template
with open('/tmp/workflow-template.md') as f:
    template = f.read()

# Customize
params = {
    'repository': '${{ github.repository }}',
    'workflow_name': 'Auto Labeler',
    'trigger': 'issues',
    'create_issues': False
}

customized = customize_workflow(template, params)

# Save customized workflow
with open('/tmp/new-workflow.md', 'w') as f:
    f.write(customized)

print("Workflow customized successfully")
```

### Step 3: Validate Workflow

```bash
# Compile to check for errors
gh aw compile /tmp/new-workflow.md

if [ $? -eq 0 ]; then
  echo "✓ Workflow compiled successfully"
else
  echo "✗ Workflow compilation failed"
  exit 1
fi

# Check for common issues
echo "Running validation checks..."

# Check 1: Has required fields
grep -q "^name:" /tmp/new-workflow.md || echo "⚠️  Missing name field"
grep -q "^on:" /tmp/new-workflow.md || echo "⚠️  Missing on field"
grep -q "^engine:" /tmp/new-workflow.md || echo "⚠️  Missing engine field"

# Check 2: Safe-outputs are properly configured
if grep -q "safe-outputs:" /tmp/new-workflow.md; then
  echo "✓ Has safe-outputs configured"
fi

# Check 3: Strict mode for security
if grep -q "^strict: true" /tmp/new-workflow.md; then
  echo "✓ Strict mode enabled"
else
  echo "⚠️  Consider enabling strict mode"
fi
```

### Step 4: Create Branch and Commit

```bash
# Create feature branch
BRANCH_NAME="workflow-manager/add-$(date +%Y%m%d-%H%M%S)"
git checkout -b "$BRANCH_NAME"

# Copy workflow to .github/workflows/
WORKFLOW_NAME="auto-labeler-$(date +%Y%m%d).md"
cp /tmp/new-workflow.md ".github/workflows/$WORKFLOW_NAME"

# Compile the lock file
gh aw compile ".github/workflows/$WORKFLOW_NAME"

# Stage files
git add ".github/workflows/$WORKFLOW_NAME"
git add ".github/workflows/$WORKFLOW_NAME.lock.yml"

# Commit
git commit -m "Add generated workflow: $WORKFLOW_NAME

Generated by Workflow Manager
Template: ${{ github.event.inputs.template }}
"
```

### Step 5: Create Pull Request

```markdown
Use create-pull-request safe-output:

**Title**: Add generated workflow: [workflow-name]

**Description**:
## Generated Workflow

This PR adds a new agentic workflow generated from the **[template-name]** pattern.

### Workflow Details
- **Name**: [workflow-name]
- **Trigger**: [trigger-type]
- **Engine**: [engine-type]
- **Safe Outputs**: [outputs-list]

### What It Does

[Brief description of workflow functionality]

### Customizations Applied

- [Customization 1]
- [Customization 2]
- [Customization 3]

### Testing

- [x] Workflow compiled successfully
- [x] Lock file generated
- [x] Validation checks passed
- [ ] Review configuration
- [ ] Test with workflow_dispatch

### Next Steps

1. Review the workflow configuration
2. Test using workflow_dispatch trigger
3. Monitor first few runs
4. Adjust as needed

---
*Generated by [Workflow Manager]({run_url})*
```

## Customization Guide

### Add More Templates

```bash
# TODO: Add your custom templates

TEMPLATES_DIR="examples/patterns"
CUSTOM_TEMPLATES_DIR=".github/workflow-templates"

# List available templates
list_templates() {
  echo "Available templates:"
  ls -1 "$TEMPLATES_DIR"/*.md
  ls -1 "$CUSTOM_TEMPLATES_DIR"/*.md 2>/dev/null
}
```

### Batch Updates

```python
# TODO: Implement batch update logic

def batch_update_workflows(directory, update_fn):
    """Apply updates to all workflows"""
    import os
    from pathlib import Path
    
    workflow_dir = Path(directory)
    updated = []
    
    for workflow_file in workflow_dir.glob('*.md'):
        # Skip lock files
        if workflow_file.name.endswith('.lock.yml'):
            continue
        
        # Read workflow
        content = workflow_file.read_text()
        
        # Apply update function
        new_content = update_fn(content)
        
        # Only save if changed
        if new_content != content:
            workflow_file.write_text(new_content)
            updated.append(workflow_file.name)
            print(f"Updated: {workflow_file.name}")
    
    return updated

# Example: Update all workflows to use new engine version
def update_engine_version(content):
    return content.replace(
        'engine: copilot',
        'engine:\n  id: copilot\n  model: gpt-5-mini'
    )

updated_files = batch_update_workflows('.github/workflows', update_engine_version)
print(f"Updated {len(updated_files)} workflows")
```

### Validation Rules

```yaml
# TODO: Define validation rules

validation-rules:
  required-fields:
    - name
    - on
    - engine
    - safe-outputs
  
  recommended-settings:
    - strict: true
    - timeout-minutes: 30
  
  security-checks:
    - no-hardcoded-secrets
    - safe-outputs-configured
    - minimal-permissions
  
  style-checks:
    - consistent-naming
    - proper-documentation
    - clear-description
```

## Example Scenarios

### Scenario 1: Generate Auto-Labeler

```markdown
**Input**:
- Template: auto-labeler
- Repository: myorg/myrepo
- Labels: ["bug", "enhancement", "documentation"]

**Generated Workflow**:
```yaml
---
name: Auto Labeler
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
safe-outputs:
  add-labels:
    max: 3
---

Automatically label issues and PRs...
```

**Result**: PR created with new workflow
```

### Scenario 2: Bulk Update

```markdown
**Input**: Update all workflows to use MCP remote mode

**Process**:
1. Find 15 workflows using local GitHub tools
2. Update to use remote mode
3. Recompile all workflows
4. Create single PR with all changes

**Result**: PR updating 15 workflows
```

### Scenario 3: Migration

```markdown
**Input**: Migrate old safe-outputs format to new format

**Before**:
```yaml
safe-outputs:
  create_issue: true
```

**After**:
```yaml
safe-outputs:
  create-issue:
    max: 1
```

**Result**: PR migrating 20 workflows to new format
```

## Advanced Features

### AI-Assisted Generation

```markdown
Use the AI to analyze requirements and generate workflows:

**Input**: "I need a workflow that monitors PR reviews and comments when reviews are too slow"

**Process**:
1. AI analyzes requirements
2. Selects relevant patterns (PR reviewer + CI monitor)
3. Generates custom workflow combining patterns
4. Validates and creates PR
```

### Template Variables

```yaml
# TODO: Define template variables

template-variables:
  repository: ${{ github.repository }}
  owner: ${{ github.repository_owner }}
  default_branch: main
  workflow_name: Generated Workflow
  trigger_event: workflow_dispatch
  engine: copilot
```

### Workflow Testing

```bash
# TODO: Add automated testing

# Test workflow locally
test_workflow() {
  local workflow_file=$1
  
  # Compile
  gh aw compile "$workflow_file" || return 1
  
  # Validate
  validate_workflow "$workflow_file" || return 1
  
  # Dry-run (if possible)
  # gh aw run --dry-run "$workflow_file"
  
  return 0
}
```

## Related Examples

- **Production examples**:
  - `.github/workflows/workflow-generator.md` - Workflow generation from requirements

## Tips

- **Start with patterns**: Use proven patterns as templates
- **Validate thoroughly**: Always compile and validate generated workflows
- **Test incrementally**: Test generated workflows before rolling out
- **Document customizations**: Keep track of what was customized
- **Version templates**: Track template versions for reproducibility
- **Review generated code**: Always review generated workflows before merging

## Security Considerations

- Generated workflows should use `strict: true`
- Validate all inputs before generating workflows
- Review generated workflows for security issues
- Use minimal permissions in generated workflows
- Never include hardcoded secrets

---

**Pattern Info**:
- Complexity: Advanced
- Trigger: workflow_dispatch (manual)
- Safe Outputs: create_pull_request
- Tools: GitHub (repos, pull_requests), bash (git, gh)
