---
description: Safe-output reference for runtime defaults, custom jobs, scripts, actions, global configuration, and output variables.
---

# Safe Outputs: Runtime and Extensibility


  The report-incomplete safe-output is automatically enabled by default and is distinct from `noop`. Use it when required tools or data are unavailable and the task cannot be meaningfully performed (e.g., MCP server crash, missing authentication, inaccessible repository). When an agent emits `report_incomplete`, gh-aw activates failure handling even when the agent process exits 0 — preventing empty outputs from being classified as successful. This ensures every unrecoverable failure is tracked.

- `jobs:` - Custom safe-output jobs registered as MCP tools for third-party integrations

  ```yaml
  safe-outputs:
    jobs:
      send-notification:
        description: "Send a notification to an external service"
        runs-on: ubuntu-latest
        output: "Notification sent successfully!"
        inputs:
          message:
            description: "The message to send"
            required: true
            type: string
        permissions:
          contents: read
        env:
          API_KEY: ${{ secrets.API_KEY }}
        steps:
          - name: Send notification
            run: |
              MESSAGE=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r '.items[] | select(.type == "send_notification") | .message')
              curl -H "Authorization: $API_KEY" -d "$MESSAGE" https://api.example.com/notify
  ```

  Custom safe-output jobs define post-processing GitHub Actions jobs registered as MCP tools. Agents call the tool by its normalized name (dashes converted to underscores, e.g., `send_notification`). The job runs after the agent completes with access to `$GH_AW_AGENT_OUTPUT` (the path to agent output JSON). Use this to integrate with Slack, Discord, external APIs, databases, or any service requiring secrets. Import from shared files using the `imports:` field.

- `scripts:` - Inline JavaScript handlers running inside the safe-outputs job handler loop

  ```yaml
  safe-outputs:
    scripts:
      post-slack-message:
        description: "Post a message to Slack"
        inputs:
          channel:
            description: "Target Slack channel"
            type: string
            default: "#general"
        script: |
          // 'channel' is available from config inputs; 'item' contains runtime message values
          await fetch(process.env.SLACK_WEBHOOK_URL, {
            method: "POST",
            body: JSON.stringify({ text: item.message, channel })
          });
  ```

  Unlike `jobs:` (which create separate GitHub Actions jobs), scripts execute in-process alongside built-in handlers. Write only the handler body — the compiler generates the outer wrapper with config input destructuring and `async function handleX(item, resolvedTemporaryIds) { ... }`. Script names with dashes are normalized to underscores (e.g., `post-slack-message` → `post_slack_message`). The handler receives `item` (runtime message with input values) and `resolvedTemporaryIds` (map of temporary IDs).

- `actions:` - Custom GitHub Actions mounted as MCP tools for the AI agent (resolved at compile time)

  ```yaml
  safe-outputs:
    actions:
      my-action:
        uses: owner/repo/path@ref         # Required: GitHub Action reference (tag, SHA, or branch)
        description: "Custom description" # Optional: override action's description from action.yml
        env:
          API_KEY: ${{ secrets.API_KEY }} # Optional: environment variables for the injected step
  ```

  Actions are resolved at compile time — the compiler fetches `action.yml` and parses inputs automatically, exposing them as MCP tool parameters. The agent calls the action by its normalized name (dashes converted to underscores). Each action runs as an injected step in the safe-outputs job. Local actions (`./path/to/action`) are also supported.

**Global Safe Output Configuration:**
- `github-token:` - Custom GitHub token for all safe output jobs

  ```yaml
  safe-outputs:
    create-issue:
    add-comment:
    github-token: ${{ secrets.GH_AW_SAFE_OUTPUTS_TOKEN }}  # Use custom PAT instead of GITHUB_TOKEN
  ```

  Useful when you need additional permissions or want to perform actions across repositories.
- `allowed-domains:` - Allowed domains for URLs in safe output content (array)
  - URLs from unlisted domains are replaced with `(redacted)`
  - GitHub domains are always included by default
- `allowed-github-references:` - Allowed repositories for GitHub-style references (array)
  - Controls which GitHub references (`#123`, `owner/repo#456`) are allowed in workflow output
  - References to unlisted repositories are escaped with backticks to prevent timeline items
  - Configuration options:
    - `[]` - Escape all references (prevents all timeline items)
    - `["repo"]` - Allow only the target repository's references
    - `["repo", "owner/other-repo"]` - Allow specific repositories
    - Not specified (default) - All references allowed
  - Example:

    ```yaml
    safe-outputs:
      allowed-github-references: []  # Escape all references
      create-issue:
        target-repo: "my-org/main-repo"
    ```

    With `[]`, references like `#123` become `` `#123` `` and `other/repo#456` becomes `` `other/repo#456` ``, preventing timeline clutter while preserving information.
- `messages:` - Custom message templates for safe-output footer and notification messages (object)
  - Available placeholders: `{workflow_name}`, `{run_url}`, `{agentic_workflow_url}`, `{triggering_number}`, `{workflow_source}`, `{workflow_source_url}`, `{operation}`, `{event_type}`, `{status}`, `{ai_credits}`, `{ai_credits_formatted}`, `{ai_credits_suffix}`
  - Message types:
    - `footer:` - Custom footer for AI-generated content
    - `footer-install:` - Installation instructions appended to footer
    - `footer-workflow-recompile:` - Footer for workflow recompile tracking issues (placeholder: `{repository}`)
    - `footer-workflow-recompile-comment:` - Footer for comments on workflow recompile issues (placeholder: `{repository}`)
    - `run-started:` - Workflow activation notification
    - `run-success:` - Successful completion message
    - `run-failure:` - Failure notification message
    - `detection-failure:` - Detection job failure message
    - `agent-failure-issue:` - Footer for agent failure tracking issues
    - `agent-failure-comment:` - Footer for comments on agent failure tracking issues
    - `staged-title:` - Staged mode preview title
    - `staged-description:` - Staged mode preview description
    - `append-only-comments:` - Create new comments instead of editing existing ones (boolean, default: false)
    - `pull-request-created:` - Custom message when a PR is created. Placeholders: `{item_number}`, `{item_url}`
    - `issue-created:` - Custom message when an issue is created. Placeholders: `{item_number}`, `{item_url}`
    - `commit-pushed:` - Custom message when a commit is pushed. Placeholders: `{commit_sha}`, `{short_sha}`, `{commit_url}`
    - `body-header:` - Custom header text prepended to every message body (issues, comments, PRs, discussions). Placeholders: `{workflow_name}`, `{run_url}`
  - Example:

    ```yaml
    safe-outputs:
      messages:
        append-only-comments: true
        footer: "> Generated by [{workflow_name}]({run_url})"
        run-started: "[{workflow_name}]({run_url}) started processing this {event_type}."
    ```

- `mentions:` - Configuration for @mention filtering in safe outputs (boolean or object)
  - Boolean format: `false` - Always escape mentions; `true` - Always allow (error in strict mode)
  - Object format for fine-grained control:

    ```yaml
    safe-outputs:
      mentions:
        allow-team-members: true    # Allow repository collaborators (default: true)
        allow-context: true          # Allow mentions from event context (default: true)
        allowed: [copilot, user1]    # Always allow specific users/bots
        max: 50                      # Maximum mentions per message (default: 50)
    ```

  - Team members include collaborators with any permission level (excluding bots unless explicitly listed)
  - Context mentions include issue/PR authors, assignees, and commenters
- `runs-on:` - Runner specification for all safe-outputs jobs (string)
  - Defaults to `ubuntu-slim` (1-vCPU runner)
  - Examples: `ubuntu-latest`, `windows-latest`, `self-hosted`
  - Applies to activation, create-issue, add-comment, and other safe-output jobs
- `footer:` - Global footer control for all safe outputs (boolean, default: `true`)
  - When `false`, omits visible AI-generated footer content from all created/updated entities (issues, PRs, discussions, releases) while still including XML markers for searchability
  - Individual safe-output types can override this setting
- `staged:` - Preview mode for all safe outputs (boolean)
  - When `true`, emits step summary messages instead of making GitHub API calls; useful for testing without side effects
- `env:` - Environment variables passed to all safe output jobs (object)
  - Values typically reference secrets: `MY_VAR: ${{ secrets.MY_SECRET }}`
- `steps:` - Custom steps injected into all safe-output jobs, running after repository checkout and before safe-output code (array)
  - Useful for installing dependencies or performing setup needed by safe-output logic
  - Example:

    ```yaml
    safe-outputs:
      steps:
        - name: Install custom dependencies
          run: npm install my-package
      create-issue:
    ```

- `max-bot-mentions:` - Maximum bot trigger references (e.g. `@copilot`, `@github-actions`) allowed in output before all excess are escaped with backticks (integer or expression, default: 10)
  - Set to `0` to escape all bot trigger phrases
  - Example: `max-bot-mentions: 3`
- `activation-comments:` - Disable all activation and fallback comments (boolean or expression, default: `true`)
  - When `false`, disables run-started, run-success, run-failure, and PR/issue creation link comments
  - Supports templatable boolean: `false`, `true`, or GitHub Actions expressions like `${{ inputs.activation-comments }}`

**Templatable Integer Fields**: The `max`, `expires`, and `max-bot-mentions` fields (and most other numeric/boolean fields) accept GitHub Actions expression strings in addition to literal values, enabling runtime-configured limits:

```yaml
safe-outputs:
  max-bot-mentions: ${{ inputs.max-mentions }}
  create-issue:
    max: ${{ inputs.max-issues }}
    expires: ${{ inputs.expires-days }}
```

Fields that influence permission computation (`add-comment.discussions`, `create-pull-request.fallback-as-issue`) remain literal booleans.

- `max-patch-size:` - Maximum allowed git patch size in kilobytes (integer, default: 1024 KB = 1 MB)
  - Patches exceeding this size are rejected to prevent accidental large changes
- `max-patch-files:` - Maximum allowed number of unique files in a create-pull-request patch (integer, default: 100)
  - Counts unique file paths deduplicated across multi-commit patches; reflects how many distinct files the agent is pushing per iteration
  - Increase this limit for long-running branches that touch many files
- `group-reports:` - Group workflow failure reports as sub-issues (boolean, default: `false`)
  - When `true`, creates a parent `[aw] Failed runs` issue that tracks all workflow failures as sub-issues; useful for larger repositories
- `report-failure-as-issue:` - Control whether workflow failures are reported as GitHub issues (boolean, default: `true`)
  - When `false`, suppresses automatic failure issue creation for this workflow
  - Use to silence noisy failure reports for workflows where failures are expected or handled externally
- `failure-issue-repo:` - Repository to create failure tracking issues in (string, format: `"owner/repo"`)
  - Defaults to the current repository when not specified
  - Use when the current repository has issues disabled: `failure-issue-repo: "myorg/infra-alerts"`
- `id-token:` - Override the id-token permission for the safe-outputs job (string: `"write"` or `"none"`)
  - `"write"`: force-enable `id-token: write` permission (required for OIDC authentication with cloud providers)
  - `"none"`: suppress automatic detection and prevent adding `id-token: write` even when vault/OIDC actions are detected in steps
  - Default: auto-detects known OIDC/vault actions (e.g., `aws-actions/configure-aws-credentials`, `azure/login`, `hashicorp/vault-action`) and adds `id-token: write` automatically
- `concurrency-group:` - Concurrency group for the safe-outputs job (string)
  - When set, the safe-outputs job uses this concurrency group with `cancel-in-progress: false`
  - Supports GitHub Actions expressions, e.g., `"safe-outputs-${{ github.repository }}"`
- `needs:` - Additional custom workflow jobs the safe-outputs job depends on (array)
  - Example: `needs: [secrets_fetcher]`
  - Use when the safe-outputs job requires outputs from a custom job defined in `jobs:`
- `environment:` - Override the GitHub deployment environment for the safe-outputs job (string)
  - Defaults to the top-level `environment:` field when not specified
  - Use when the main job and safe-outputs job need different deployment environments for protection rules
- `github-app:` - GitHub App credentials for minting installation access tokens (object)
  - When configured, generates a token from the app and uses it for all safe output operations (alternative to `github-token`)
  - Fields:
    - `client-id:` - GitHub App client ID (required, e.g., `${{ vars.APP_ID }}`). Use `app-id:` for legacy compatibility.
    - `private-key:` - GitHub App private key (required, e.g., `${{ secrets.APP_PRIVATE_KEY }}`)
    - `owner:` - Optional App installation owner (defaults to current repository owner)
    - `repositories:` - Optional list of repositories to grant access to
  - Example:

    ```yaml
    safe-outputs:
      github-app:
        client-id: ${{ vars.APP_ID }}
        private-key: ${{ secrets.APP_PRIVATE_KEY }}
      create-issue:
    ```

- `threat-detection:` - Threat detection configuration (auto-enabled for all safe-outputs workflows)
  - Automatically enabled by default; customizable via explicit configuration
  - Fields:
    - `enabled:` - Enable/disable threat detection (boolean, default: `true`)
    - `prompt:` - Additional instructions appended to threat detection analysis (string)
    - `engine:` - AI engine for threat detection (engine config or `false` to disable AI detection)
    - `steps:` - Extra job steps to run after detection (array)
  - Example to disable AI-based detection (use custom steps only):

    ```yaml
    safe-outputs:
      threat-detection:
        engine: false
        steps:
          - name: Custom check
            run: echo "Custom threat check"
    ```


## Output Variables

The safe-outputs job emits named step outputs for the first successful result of each type:

| Safe Output | Step Output Variables |
|---|---|
| `create-issue` | `created_issue_number`, `created_issue_url` |
| `create-pull-request` | `created_pr_number`, `created_pr_url` |
| `add-comment` | `comment_id`, `comment_url` |
| `push-to-pull-request-branch` | `push_commit_sha`, `push_commit_url` |
