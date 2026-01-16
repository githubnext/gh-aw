---
title: "How Agentic Workflows Work"
description: "The technical foundation: from natural language to secure execution"
authors:
  - dsyme
  - pelikhan
  - mnkiefer
date: 2026-02-05
draft: true
prev:
  link: /gh-aw/blog/2026-02-02-security-lessons/
  label: Security Lessons
next:
  link: /gh-aw/blog/2026-02-08-authoring-workflows/
  label: Authoring Workflows
---

[Previous Article](/gh-aw/blog/2026-02-02-security-lessons/)

---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

*Aha!* Time for a deep plunge into the *bubbly depths* of Peli's Agent Factory! Having explored the [security vault](/gh-aw/blog/2026-02-02-security-lessons/), we shall now peek behind the curtain and discover the *magnificent machinery* - the technical foundation that makes it all work!

Ever wonder what actually happens when you write an agentic workflow? Let's take a journey from a simple Markdown file all the way to secure, auditable execution in GitHub Actions.

Every agent in Peli's Agent Factory follows the same basic lifecycle, transforming natural language descriptions into production-ready workflows. Understanding this architecture helps you design effective agents and debug issues when they pop up.

Let's walk through the complete journey together!

## The Three-Stage Lifecycle

```text
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Write     │      │   Compile   │      │     Run     │
│  (Markdown) │ ───> │   (YAML)    │ ───> │  (Actions)  │
└─────────────┘      └─────────────┘      └─────────────┘
  Natural Lang         Secure Lock          Team Visible
  Declarative          Validated            Auditable
  Human-Friendly       Machine-Ready        Observable
```

Three stages, each with a clear purpose. Let's explore what happens at each step!

## Stage 1: Write in Natural Language

Agentic workflows start as **Markdown files** that combine natural language prompts with declarative configuration. Think of it as writing instructions for a helpful robot.

### Anatomy of a Workflow File

```markdown
---
description: Investigates failed CI workflows to identify root causes
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
permissions:
  contents: read
  issues: write
tools:
  github:
    toolsets: [issues, pull-requests]
  bash:
    commands: [git, jq]
network:
  allowed:
    - "api.github.com"
safe_outputs:
  create_issue:
    title_prefix: "[CI Doctor]"
    labels: ["ci", "automated"]
    max_items: 3
    expire: "+7d"
---

# CI Doctor

When a CI workflow fails, investigate the root cause:

1. Download the workflow logs
2. Analyze the failure patterns
3. Identify the root cause
4. Create an issue with diagnostic information

Include:
- Failure summary
- Relevant log excerpts
- Suggested fixes
- Related issues or PRs
```

### Frontmatter: Declarative Configuration

The YAML frontmatter defines **how** the workflow runs:

**Triggers** (`on:`):

- Schedule: `schedule: "0 9 * * 1-5"` (weekdays at 9am)
- Events: `workflow_run`, `issues`, `pull_request`
- Manual: `workflow_dispatch`
- Comments: `issue_comment` with body filters

**Permissions** (`permissions:`):

- Start with `contents: read` (always!)
- Add write permissions sparingly
- Scope to specific resources

**Tools** (`tools:`):

- GitHub API toolsets
- Bash commands (explicitly listed)
- MCP servers for specialized capabilities
- Each tool is explicitly enumerated (no wildcards!)

**Network** (`network:`):

- Allowlisted domains only
- Prevents data exfiltration
- Enforced at infrastructure level

**Safe Outputs** (`safe_outputs:`):

- Templates for creating issues/PRs
- Built-in guardrails (max_items, expire, etc.)
- All writes go through safe outputs

### Prompt: Natural Language Instructions

The Markdown content after frontmatter is the **agent's prompt** - natural language instructions describing what the agent should do, how it should behave, and what outputs to create.

**Effective prompts:**

- Start with clear objective
- Provide step-by-step guidance
- Include examples of good outputs
- Set tone and personality
- Reference available tools
- Specify output format

This is where you get to be creative! The prompt is your chance to give the agent personality and clear direction.

## Stage 2: Compile to Secure Workflows

The `gh aw compile` command transforms natural language workflows into GitHub Actions YAML files with embedded security controls.

### Compilation Process

```text
┌──────────────┐
│ workflow.md  │
└──────┬───────┘
       │
       ├─► Parse frontmatter
       │   └─► Validate schema
       │
       ├─► Load imports
       │   └─► Merge configurations
       │
       ├─► Validate security
       │   ├─► Check permissions
       │   ├─► Verify tool allowlists
       │   ├─► Validate network rules
       │   └─► Audit safe outputs
       │
       ├─► Generate GitHub Actions jobs
       │   ├─► Setup job (environment prep)
       │   ├─► Agent job (AI execution)
       │   └─► Safe output jobs (writes)
       │
       └─► Write workflow.lock.yml
           └─► Locked, validated, ready to run
```

### What Compilation Does

**1. Schema Validation**

Ensures frontmatter conforms to GitHub Agentic Workflows schema:

- Valid trigger syntax
- Recognized permissions
- Known tool configurations
- Correct safe output templates

**2. Security Validation**

Enforces security policies:

- Permissions don't exceed requirements
- Tools are explicitly listed (no wildcards in strict mode)
- Network access is constrained
- Safe outputs have appropriate limits

**3. Import Resolution**

Loads and merges imported components:

- Fetch shared component files
- Merge tool configurations
- Combine prompt instructions
- Resolve version pins

**4. Job Generation**

Creates GitHub Actions jobs:

**Setup Job**: Prepares environment

- Installs required tools
- Configures MCP servers
- Sets up network restrictions
- Prepares safe output infrastructure

**Agent Job**: Runs AI agent

- Provides access to tools
- Executes prompt against AI engine
- Captures outputs
- Handles errors gracefully

**Safe Output Jobs**: Applies changes

- Processes safe output requests
- Validates against templates
- Applies guardrails (max_items, etc.)
- Creates issues/PRs/comments

**5. Lock File Generation**

Produces a `.lock.yml` file:

- Contains complete GitHub Actions workflow
- Includes security validations
- Embeds prompt and configuration
- Ready for deployment

### Example Compilation

**Input: `ci-doctor.md`** (50 lines of natural language)

**Output: `ci-doctor.lock.yml`** (300 lines of validated YAML)

The lock file includes:

- All environment setup
- Tool installations
- Security controls
- Agent execution logic
- Safe output processing
- Error handling
- Cleanup steps

## Stage 3: Run and Produce Artifacts

Compiled workflows execute on GitHub Actions runners, producing team-visible artifacts.

### Execution Flow

```text
Workflow Triggered
    │
    ├─► Setup Job
    │   ├─► Install gh-aw CLI
    │   ├─► Configure MCP servers
    │   ├─► Setup network restrictions
    │   └─► Prepare safe output handlers
    │
    ├─► Agent Job
    │   ├─► Load prompt
    │   ├─► Gather context (issues, PRs, files)
    │   ├─► Execute against AI engine
    │   │   └─► Agent uses tools as needed
    │   ├─► Generate safe output requests
    │   └─► Upload artifacts
    │
    └─► Safe Output Jobs (parallel)
        ├─► Create Issue (if requested)
        ├─► Create PR (if requested)
        ├─► Add Comment (if requested)
        └─► Upload Assets (if requested)
```

### Agent Execution Environment

The agent runs in a sandboxed environment with:

**Tools Available:**

- GitHub API (via MCP toolsets)
- Bash commands (allowlisted)
- File system access (repository only)
- MCP servers (as configured)

**Context Provided:**

- Trigger event details
- Repository state
- Recent issues/PRs
- Relevant files
- Previous workflow runs

**Constraints Applied:**

- Network allowlist enforced
- Tool restrictions active
- Permission boundaries set
- Safe output templates ready

### Output Types

Agents produce several types of outputs:

**1. Issues**

```yaml
safe_outputs:
  create_issue:
    title: "CI failure in test suite"
    body: "Detailed analysis..."
    labels: ["ci", "automated"]
```

**2. Pull Requests**

```yaml
safe_outputs:
  create_pull_request:
    title: "Fix dependency vulnerability"
    body: "Updates package X..."
    branch: "agent/fix-vuln-123"
```

**3. Comments**

```yaml
safe_outputs:
  add_comment:
    issue_number: 42
    body: "Analysis complete..."
```

**4. Discussions**

```yaml
safe_outputs:
  create_discussion:
    title: "Weekly Metrics Report"
    body: "## Report\n..."
    category: "Reports"
```

**5. Artifacts**

- Charts and visualizations
- Data files (CSV, JSON)
- Reports (PDF, HTML)
- Logs and debug info

### Auditable Artifacts

Every agent action creates a permanent record:

**Workflow Runs**: Full execution logs

- Start/end times
- Tool invocations
- API calls made
- Errors encountered

**Issues/PRs/Comments**: Timestamped, attributed

- Who triggered (user or schedule)
- What workflow ran
- When it executed
- What it created

**Discussions**: Permanent, searchable

- Historical reports
- Trend analysis
- Team knowledge base

**Artifacts**: Versioned, downloadable

- Charts and data files
- Debug logs
- Intermediate results

## The AI Engine Interface

Workflows can use different AI engines:

### Copilot Engine (Default)

```yaml
engine: copilot
model: claude-sonnet-4  # or other models
```

GitHub Copilot provides:

- Code-aware context
- GitHub API integration
- Secure execution environment
- Usage tracked in Copilot subscription

### Claude Engine

```yaml
engine: claude
model: claude-sonnet-4
```

Anthropic Claude (via API key) provides:

- Long context windows
- Strong reasoning
- Detailed analysis
- Requires ANTHROPIC_API_KEY secret

### Codex Engine

```yaml
engine: codex
```

Azure OpenAI Codex provides:

- Enterprise integration
- Fine-tuned models
- Compliance features
- Requires Azure credentials

### Custom Engine

```yaml
engine: custom
endpoint: "https://my-ai.example.com"
```

Bring your own AI provider:

- Custom endpoints
- Proprietary models
- Specialized fine-tuning
- Full control over AI stack

## Tool Architecture: MCP Servers

Model Context Protocol (MCP) servers provide specialized capabilities:

### How MCP Works

```
Agent ─────> MCP Gateway ─────> MCP Servers
                                 ├─► GitHub API
                                 ├─► Bash commands
                                 ├─► Serena (code analysis)
                                 ├─► Tavily (web search)
                                 └─► Custom tools
```

### MCP Server Types

**Built-in Servers:**

- `github` - GitHub API operations
- `bash` - Shell command execution
- `filesystem` - File operations

**External Servers:**

- `serena` - Semantic code analysis
- `tavily` - Web search
- `markitdown` - Document conversion
- `ast-grep` - Structural code search

### MCP Configuration

```yaml
tools:
  github:
    toolsets: [repos, issues, pull-requests]
  bash:
    commands: [git, jq, python]
  serena:
    mode: remote
    version: latest
```

## Error Handling and Debugging

When workflows fail, several debugging mechanisms help:

### Workflow Logs

Every run produces detailed logs:

- Job-level logs (setup, agent, outputs)
- Step-level logs (individual actions)
- Tool invocation logs (what tools were called)
- Error messages (when things fail)

### Safe Output Validation Errors

If safe output validation fails:

- Clear error messages explain why
- Template requirements highlighted
- Example corrections provided
- Workflow continues (fail gracefully)

### MCP Server Diagnostics

The [`mcp-inspector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md) workflow validates:

- Server availability
- Authentication status
- Network connectivity
- Configuration correctness

### Meta-Agent Monitoring

The [`audit-workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md) agent:

- Tracks all workflow runs
- Classifies failures
- Identifies patterns
- Creates issues for persistent problems

## Performance Considerations

### Execution Time

Typical workflow execution:

- Setup: 30-60 seconds
- Agent execution: 1-5 minutes
- Safe outputs: 10-30 seconds
- **Total**: 2-6 minutes per run

### Cost Factors

Workflow costs include:

- GitHub Actions compute (free tier or paid)
- AI engine API calls (per-token pricing)
- MCP server usage (varies by provider)
- Storage (for artifacts)

### Optimization Strategies

**Reduce API calls:**

- Cache repeated queries
- Batch operations
- Use repo-memory for persistence

**Optimize prompts:**

- Be concise but complete
- Avoid redundant context
- Use imports for shared logic

**Limit tool scope:**

- Request only needed permissions
- Use specific GitHub API toolsets
- Constrain bash commands

## What's Next?

Now that you understand how workflows work under the hood, you're ready to start authoring your own agents for Peli's Agent Factory.

In our next article, we'll provide a practical guide to creating effective agentic workflows, with examples and best practices from the factory.

_More articles in this series coming soon._

[Previous Article](/gh-aw/blog/2026-02-02-security-lessons/)
