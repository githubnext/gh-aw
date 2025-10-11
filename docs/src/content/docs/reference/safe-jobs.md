---
title: Safe Jobs
description: Learn about safe-jobs feature that enables defining custom post-processing workflows with GitHub Actions job properties and artifact access.
sidebar:
  order: 600
---

The `safe-outputs.jobs:` element of your workflow's frontmatter enables you to define custom post-processing jobs that execute after the main agentic workflow completes. Safe-jobs provide a powerful way to create sophisticated automation workflows while maintaining security through controlled job execution.

**How It Works:**
1. Safe-jobs are defined under the `safe-outputs.jobs` section of the frontmatter
2. Each safe-job **must have an "inputs" section with at least one input** - these inputs become the arguments of the MCP tool
3. Each safe-job has access to all standard GitHub Actions job properties (runs-on, if, needs, env, permissions, steps)
4. Safe-jobs automatically download the agent output artifact and can process it using jq
5. Safe-jobs become available as callable tools in the safe-outputs MCP server
6. Safe-jobs can be imported from included workflows with automatic conflict detection

**Important Requirement**: Every safe-job definition must include an `inputs` section with at least one input parameter. These inputs define the MCP tool's arguments and enable the agentic workflow to call the safe-job with appropriate parameters.

## Basic Usage

**Note**: All safe-jobs must have an `inputs` section with at least one input parameter.

```aw wrap
---
on: issues
engine: claude
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      inputs:
        environment:
          description: "Target deployment environment"
          required: true
          type: choice
          options: ["staging", "production"]
      steps:
        - name: Deploy application
          run: |
            if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
              ENV=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "deploy") | .environment // "staging"')
              echo "Deploying to $ENV based on agent analysis"
            fi
---

# Deployment Workflow

Analyze the issue and potentially trigger deployment using the safe-job.
```

## Safe-Jobs vs Safe-Outputs

| Feature | Safe-Outputs | Safe-Jobs |
|---------|--------------|-----------|
| **Purpose** | Predefined GitHub API actions | Custom workflow jobs |
| **Configuration** | Simple configuration options | Full GitHub Actions job properties |
| **Permissions** | Automatically granted | User-defined permissions |
| **Flexibility** | Limited to built-in actions | Complete control over job execution |
| **Agent Output** | Processed automatically | Manual processing via jq |
| **Include Support** | ✓ | ✓ |

## Core Features

### GitHub Actions Job Properties

Safe-jobs support all standard GitHub Actions job properties:

```yaml
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      if: github.event.issue.number
      timeout-minutes: 30
      permissions:
        contents: write
        deployments: write
      env:
        DEPLOY_ENV: production
      inputs:
        confirm:
          description: "Confirm deployment"
          required: true
          type: boolean
          default: "false"
    steps:
      - name: Deploy
        run: echo "Deploying..."
```

### Workflow Dispatch Inputs (Required)

**Every safe-job must define inputs** using workflow_dispatch syntax for parameterization. The inputs become the MCP tool arguments:

```yaml
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      inputs:
        environment:
          description: "Target deployment environment"
          required: true
          type: choice
          options: ["staging", "production"]
        force:
          description: "Force deployment even if tests fail"
          required: false
          type: boolean
          default: "false"
    steps:
      - name: Deploy application
        run: |
          if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
            ENV=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "deploy") | .environment // "staging"')
            echo "Deploying to $ENV"
          fi
```

### Custom Output Messages

Safe-jobs can return custom response messages via the MCP server:

```yaml
safe-outputs:
  jobs:
    notify:
      runs-on: ubuntu-latest
      output: "Notification sent successfully!"
      inputs:
        message:
          description: "Notification message"
          required: true
          type: string
      steps:
        - name: Send notification
          run: echo "Sending notification..."
```

### Agent Output Processing

Safe-jobs automatically receive access to the agent output artifact:

```yaml
safe-outputs:
  jobs:
    analyze:
      runs-on: ubuntu-latest
      inputs:
        data_type:
          description: "Type of data to analyze"
          required: true
          type: string
      steps:
        - name: Process agent output
          run: |
            if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
              # Extract specific data from agent output
              RESULT=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "analyze") | .result')
              echo "Agent analysis result: $RESULT"
            else
              echo "No agent output available"
            fi
```

## Include Support

Safe-jobs can be imported from included workflows with automatic conflict detection:

**Main workflow:**
```aw wrap
---
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      inputs:
        target:
          description: "Deployment target"
          required: true
          type: string
      steps:
        - name: Deploy
          run: echo "Deploying..."
---

@import shared/common-jobs.md
```

**Imported file (shared/common-jobs.md):**
```aw wrap
---
safe-outputs:
  jobs:
    test:
      runs-on: ubuntu-latest
      inputs:
        suite:
          description: "Test suite to run"
          required: true
          type: string
      steps:
        - name: Test
          run: echo "Testing..."
---
```

**Result:** Both `deploy` and `test` safe-jobs are available.

**Conflict Detection:** If both files define a safe-job with the same name, compilation fails with:
```
failed to merge safe-jobs: safe-job name conflict: 'deploy' is defined in both main workflow and included files
```

## MCP Server Integration

Safe-jobs are automatically registered as tools in the safe-outputs MCP server, allowing the agentic workflow to call them:

```yaml
safe-outputs:
  jobs:
    database-backup:
      runs-on: ubuntu-latest
      inputs:
        database:
          description: "Database to backup"
          required: true
          type: string
      steps:
        - name: Backup database
          run: echo "Backing up database..."
```

The agent can then call this safe-job:
```
Please backup the user database using the database-backup safe-job.
```

## Generated Workflow Structure

Safe-jobs compile to standard GitHub Actions jobs with automatic setup:

```yaml
deploy:
  needs: main-job
  runs-on: ubuntu-latest
  steps:
    - name: Download agent output artifact
      continue-on-error: true
      uses: actions/download-artifact@v5
      with:
        name: safe_output.jsonl
        path: /tmp/gh-aw/safe-jobs/
    - name: Setup Safe Job Environment Variables
      run: |
        echo "GITHUB_AW_AGENT_OUTPUT=/tmp/gh-aw/safe-jobs/safe_output.jsonl" >> $GITHUB_ENV
    - name: Deploy application
      run: |
        if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
          ENV=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "deploy") | .environment // "staging"')
          echo "Deploying to $ENV"
        fi
```

## Best Practices

### Security
- Use minimal required permissions for each safe-job
- Validate agent output before processing
- Use `continue-on-error: true` for artifact downloads to handle missing outputs gracefully

### Data Processing
- Always check if `$GITHUB_AW_AGENT_OUTPUT` exists before processing
- Use jq filters to extract specific data from agent output
- Provide fallback values for missing data

### Organization
- Group related safe-jobs in shared include files
- Use descriptive names for safe-jobs to avoid conflicts
- Document input requirements clearly

### Performance
- Set appropriate `timeout-minutes` for long-running jobs
- Use `if` conditions to control when safe-jobs execute
- Consider runner costs when choosing `runs-on` values

## Common Patterns

### Conditional Deployment
```yaml
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      if: contains(github.event.issue.labels.*.name, 'deploy')
      permissions:
        contents: write
      inputs:
        approved:
          description: "Whether deployment is approved"
          required: true
          type: boolean
      steps:
        - name: Deploy if requested
          run: |
            if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
              SHOULD_DEPLOY=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "deploy") | .approved // false')
              if [ "$SHOULD_DEPLOY" = "true" ]; then
                echo "Deploying application..."
              else
                echo "Deployment not approved by agent"
              fi
            fi
```

### Multi-step Processing
```yaml
safe-outputs:
  jobs:
    process-results:
      runs-on: ubuntu-latest
      inputs:
        format:
          description: "Output format"
          required: true
          type: choice
          options: ["json", "csv", "xml"]
      steps:
        - name: Extract data
          run: |
            if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
              cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r '.[] | select(.type == "result")' > /tmp/gh-aw/results.json
            fi
        - name: Process data
          run: |
            if [ -f "/tmp/gh-aw/results.json" ]; then
              echo "Processing $(wc -l < /tmp/gh-aw/results.json) results"
            fi
        - name: Upload results
          uses: actions/upload-artifact@v4
          with:
            name: processed-results
            path: /tmp/gh-aw/results.json
```

### Error Handling
```yaml
safe-outputs:
  jobs:
    robust-task:
      runs-on: ubuntu-latest
      inputs:
        retry_count:
          description: "Number of retries"
          required: false
          type: string
          default: "3"
      steps:
        - name: Safe processing
          run: |
            set -euo pipefail
            
            if [ ! -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
              echo "Warning: No agent output found, using defaults"
              echo '{"status": "default"}' > /tmp/gh-aw/config.json
            else
              cp "$GITHUB_AW_AGENT_OUTPUT" /tmp/gh-aw/config.json
            fi
            
            # Process with error handling
            if ! jq -e '.status' /tmp/gh-aw/config.json > /dev/null; then
              echo "Error: Invalid agent output format"
              exit 1
            fi
          
          STATUS=$(jq -r '.status' /tmp/gh-aw/config.json)
          echo "Processing with status: $STATUS"
```

## Schema Validation

Safe-jobs configurations are validated against comprehensive JSON schemas that ensure:

- **GitHub Actions Compatibility**: All job properties follow GitHub Actions standards
- **Input Validation**: workflow_dispatch input syntax is properly structured  
- **Permission Validation**: Permissions follow GitHub's permission model
- **Step Validation**: Steps have required properties (uses or run) with proper types

Invalid configurations will show clear error messages during compilation:
```
safe-jobs.deploy.steps[0]: must have either 'uses' or 'run' property
safe-jobs.deploy.permissions.invalid-scope: must be one of [read, write, none]
```