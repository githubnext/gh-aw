# Copilot CLI Tool Permission Flags Guide (v0.0.370+)

## Overview

Copilot CLI v0.0.370 introduces a clear distinction between two types of tool control:

1. **Tool Availability** (`--available-tools`, `--excluded-tools`) - What the model can see
2. **Tool Permissions** (`--allow-tool`, `--deny-tool`) - What requires user approval

This separation enables more granular control and better security through defense in depth.

## Flag Reference

### Tool Availability Flags

#### `--available-tools [tools...]`
**Purpose:** Restricts which tools the model can see (allowlist)

**Behavior:**
- Only specified tools are visible to the model
- All other tools are hidden and unavailable
- Acts as a strict allowlist filter

**Use Cases:**
- Creating specialized agents with limited capabilities
- Restricting to read-only operations
- Preventing model from attempting dangerous operations

**Example:**
```bash
copilot --available-tools 'github(get_file_contents)' 'github(list_commits)' \
        --allow-all-tools \
        --prompt "Analyze repository structure"
```

#### `--excluded-tools [tools...]`
**Purpose:** Hides specific tools from the model (denylist)

**Behavior:**
- Specified tools are not visible to the model
- All other tools remain available
- Acts as a denylist filter

**Use Cases:**
- Blocking dangerous operations while allowing flexibility
- Removing specific problematic tools
- Creating "safe by default" configurations

**Example:**
```bash
copilot --excluded-tools 'shell(rm:*)' 'shell(git push)' \
        --allow-all-tools \
        --prompt "Help me refactor the code"
```

### Tool Permission Flags

#### `--allow-tool [tools...]`
**Purpose:** Pre-approves tools to run without confirmation

**Behavior:**
- Specified tools execute without user prompts
- Required for non-interactive mode
- Only applies to tools visible to the model
- Does not expose filtered tools

**Use Cases:**
- Automating workflows in CI/CD
- Running unattended operations
- Pre-approving known-safe operations

**Example:**
```bash
copilot --allow-tool 'github' 'shell(git:*)' 'write' \
        --prompt "Create a pull request with changes"
```

#### `--deny-tool [tools...]`
**Purpose:** Denies permission for specific tools

**Behavior:**
- Specified tools are always denied
- Takes precedence over all allow rules
- Prevents execution even if otherwise allowed

**Use Cases:**
- Blocking specific dangerous operations
- Overriding wildcards or allow-all settings
- Implementing mandatory restrictions

**Example:**
```bash
copilot --allow-tool 'shell(git:*)' \
        --deny-tool 'shell(git push)' \
        --prompt "Work with git repository"
```

#### `--allow-all-tools`
**Purpose:** Pre-approves all visible tools

**Behavior:**
- All tools execute without prompts
- Required for non-interactive execution
- Respects availability filters
- Can be overridden by `--deny-tool`

**Use Cases:**
- Fully automated workflows
- Trusted environments
- Development and testing

**Example:**
```bash
copilot --allow-all-tools \
        --prompt "Complete the implementation"
```

## Flag Precedence

Flags are applied in this order:

1. **Availability Filters** (determine what model can see)
   - `--available-tools` OR `--excluded-tools`
   
2. **Permission Rules** (determine what can execute without confirmation)
   - `--deny-tool` (highest precedence)
   - `--allow-tool` / `--allow-all-tools`

### Example: Precedence in Action

```bash
copilot --available-tools 'github' 'shell(git:*)' 'write' \
        --deny-tool 'shell(git push)' \
        --allow-tool 'github' 'shell(git:*)' 'write'
```

**Result:**
- Model sees: `github`, `shell(git:*)`, `write`
- Auto-approved: `github`, `shell(git:*)` (except push), `write`
- Denied: `shell(git push)` - even though allowed by `shell(git:*)`

## Use Case Examples

### Safe Read-Only Agent

**Goal:** Agent can only read repository contents

```bash
copilot --available-tools 'github(get_file_contents)' 'github(list_commits)' \
        --allow-all-tools \
        --prompt "Analyze code quality"
```

**Why this works:**
- Model only sees read operations
- All visible tools are pre-approved
- No write or dangerous operations possible

### Flexible Agent with Safety Rails

**Goal:** Agent has broad capabilities but specific operations are blocked

```bash
copilot --excluded-tools 'shell(rm:*)' 'shell(git push)' \
        --allow-all-tools \
        --prompt "Refactor the authentication module"
```

**Why this works:**
- Model sees most tools for flexibility
- Dangerous operations are hidden
- All remaining tools are auto-approved

### Granular Production Agent

**Goal:** Maximum control with explicit permissions

```bash
copilot --available-tools 'github' 'shell(git:*)' 'write' \
        --deny-tool 'shell(git push)' 'shell(git force-push)' \
        --allow-tool 'github(create_issue)' 'github(add_comment)' \
        --prompt "Create issue for bug found in code"
```

**Why this works:**
- Model sees git, GitHub, and file operations
- Dangerous git operations are explicitly denied
- Only safe GitHub operations are auto-approved
- Other operations require manual confirmation

### Development/Testing Agent

**Goal:** Full access for trusted development environment

```bash
copilot --allow-all-tools \
        --prompt "Implement the new feature"
```

**Why this works:**
- All tools are visible (no availability filters)
- All tools are pre-approved
- Suitable for local development only

## Tool Pattern Syntax

### Shell Commands

```bash
shell(command:*?)    # Shell command patterns
```

**Examples:**
- `shell(echo)` - Exact match: only `echo`
- `shell(git:*)` - Prefix match: `git status`, `git commit`, etc.
- `shell` - All shell commands

**Note:** Wildcard matching uses command stems, so `shell(git:*)` won't match `gitea`

### File Operations

```bash
write    # File creation and modification
```

**Covers:**
- Creating files
- Modifying files
- Does NOT include shell redirections (use `--allow-all-tools` for those)

### MCP Server Tools

```bash
<server-name>(tool-name?)    # MCP server tool patterns
```

**Examples:**
- `github(get_file_contents)` - Specific tool from GitHub server
- `github` - All tools from GitHub server
- `MyMCP(my_tool)` - Specific tool from custom server
- `MyMCP` - All tools from custom server

## Migration Guide

### From Pre-v0.0.370

**Before:**
```bash
copilot --allow-tool 'github' 'shell(git:*)' \
        --prompt "Create a pull request"
```

**After (same behavior):**
```bash
copilot --allow-tool 'github' 'shell(git:*)' \
        --prompt "Create a pull request"
```

**After (enhanced security):**
```bash
copilot --available-tools 'github' 'shell(git:*)' 'write' \
        --allow-tool 'github' 'shell(git:*)' 'write' \
        --prompt "Create a pull request"
```

### Key Changes

1. **Backward Compatible:** Old workflows continue to work
2. **New Capability:** Can now control model visibility separately
3. **Defense in Depth:** Combine both approaches for maximum security

### When to Update

**Update if you want:**
- Prevent model from attempting unavailable operations
- Implement stricter security controls
- Create specialized agents with limited toolsets

**Don't update if:**
- Current permission model meets your needs
- Workflow already works correctly
- Simplicity is preferred over granular control

## Best Practices

1. **Use Availability for Security:** Control what the model can see
2. **Use Permissions for UX:** Control what requires approval
3. **Start Restrictive:** Begin with limited tools, expand as needed
4. **Test Thoroughly:** Verify both availability and permission behavior
5. **Document Choices:** Explain why tools are allowed/denied
6. **Combine Approaches:** Use both availability and permission flags for defense in depth

## Common Patterns

### Pattern 1: Read-Only Access
```bash
--available-tools 'github(get_*)' 'github(list_*)' --allow-all-tools
```

### Pattern 2: Safe Git Operations
```bash
--available-tools 'shell(git:*)' --deny-tool 'shell(git push)' --allow-tool 'shell(git:*)'
```

### Pattern 3: No Destructive Operations
```bash
--excluded-tools 'shell(rm:*)' 'shell(git push)' --allow-all-tools
```

### Pattern 4: Specific GitHub Workflows
```bash
--available-tools 'github(create_issue)' 'github(add_comment)' --allow-all-tools
```

## Troubleshooting

### Tool Not Available to Model

**Symptom:** Model reports tool doesn't exist or is unavailable

**Check:**
- Is tool listed in `--available-tools`?
- Is tool excluded by `--excluded-tools`?
- Does tool name match exactly?

**Solution:** Add tool to `--available-tools` or remove from `--excluded-tools`

### Tool Requires Confirmation

**Symptom:** Model attempts to use tool but prompts for approval

**Check:**
- Is tool included in `--allow-tool`?
- Is tool denied by `--deny-tool`?
- Is `--allow-all-tools` used?

**Solution:** Add tool to `--allow-tool` or use `--allow-all-tools`

### Tool Denied Despite Allow Rule

**Symptom:** Tool is denied even though it's in allow list

**Check:**
- Is tool in `--deny-tool`?
- Does deny pattern match the tool?

**Reason:** `--deny-tool` takes precedence over all allow rules

**Solution:** Remove from `--deny-tool` or adjust deny pattern

## Additional Resources

- Copilot CLI Help: `copilot help permissions`
- GitHub Agentic Workflows Docs: https://githubnext.github.io/gh-aw
- Copilot CLI Skill: `/home/runner/work/gh-aw/gh-aw/skills/copilot-cli/SKILL.md`
