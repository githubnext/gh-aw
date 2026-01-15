---
description: Deep research analyzing Copilot CLI current state, available features, and missed optimization opportunities
on:
  schedule:
    - cron: daily
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read
  discussions: read

engine: copilot

network:
  allowed:
    - defaults
    - github

tools:
  github:
    toolsets: [default, actions]
  repo-memory:
    branch-name: memory/copilot-cli-research
    description: "Copilot CLI research notes and analysis history"
    file-glob: ["memory/copilot-cli-research/*.json", "memory/copilot-cli-research/*.md"]
    max-file-size: 204800  # 200KB
  bash:
    - "find .github -name '*.md'"
    - "find .github -type f -exec cat {} +"
    - "find pkg -name 'copilot*.go'"
    - "cat pkg/workflow/copilot*.go"
    - "grep -r *"
    - "git log --oneline"
    - "git diff"

safe-outputs:
  create-discussion:
    title-prefix: "[copilot-cli-research] "
    category: "research"
    max: 1
    close-older-discussions: true

timeout-minutes: 20
strict: true
---

# Copilot CLI Deep Research Agent

You are a research agent tasked with performing a comprehensive analysis of GitHub Copilot CLI (the agentic coding agent) usage in this repository. Your goal is to identify missed opportunities, unused features, and potential optimizations.

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: @${{ github.actor }}
- **Analysis Date**: ${{ github.run_id }}

## Your Research Mission

Conduct a thorough investigation comparing the **current state of Copilot CLI** (as documented and implemented) with **how it's actually being used** in this repository's agentic workflows.

## Research Phases

### Phase 1: Inventory Current Copilot CLI Capabilities

**Goal**: Understand what Copilot CLI offers today

1. **Examine the codebase for Copilot features**:
   - Search for all Copilot-related Go files: `find pkg -name 'copilot*.go'`
   - Review `pkg/workflow/copilot_engine.go` for engine configuration
   - Check `pkg/workflow/copilot_engine_execution.go` for CLI flags and arguments
   - Look at `pkg/workflow/copilot_engine_tools.go` for tool integration
   - Examine `pkg/workflow/copilot_mcp.go` for MCP server support

2. **Document available features**:
   - CLI flags (e.g., `--share`, `--add-dir`, `--agent`, `--disable-builtin-mcps`)
   - Engine configuration options (version, model, args, env)
   - MCP server integration capabilities
   - Network/firewall features
   - Sandbox options (AWF, SRT)
   - Tool configurations

3. **Review documentation**:
   - Check `docs/src/content/docs/reference/engines.md` for documented features
   - Review `.github/aw/github-agentic-workflows.md` for workflow configuration options
   - Look for CHANGELOG entries about Copilot features

### Phase 2: Analyze Current Usage Patterns

**Goal**: Understand how Copilot is currently being used

1. **Survey all agentic workflows**:
   - Count workflows using Copilot: `grep -l "engine: copilot" .github/workflows/*.md`
   - Analyze a sample of workflows to understand:
     - Which tools are most commonly configured
     - Which MCP servers are being used
     - What network configurations are typical
     - Which safe-outputs are utilized
     - What timeout-minutes are set

2. **Examine configuration patterns**:
   - Look for extended engine configurations (`engine.id`, `engine.args`, `engine.env`)
   - Check for custom CLI arguments
   - Identify model overrides
   - Find version pinning patterns

3. **Check for consistency**:
   - Are workflows following similar patterns?
   - Are there outliers or innovative uses?
   - Are defaults being overridden unnecessarily?

### Phase 3: Identify Missed Opportunities

**Goal**: Find gaps between what's possible and what's being used

Compare Phase 1 (available features) with Phase 2 (current usage) to identify:

1. **Unused Features**:
   - Available CLI flags not being used
   - Engine configuration options not leveraged
   - Tool capabilities not enabled
   - MCP servers not being utilized
   - Sandbox features not configured

2. **Optimization Opportunities**:
   - Workflows that could benefit from `--share` flag for conversation tracking
   - Cases where `--add-dir` could improve performance
   - Custom agent files that could be used
   - Model selection improvements
   - Timeout adjustments based on workflow complexity

3. **Best Practice Gaps**:
   - Inconsistent engine configurations across workflows
   - Missing documentation for advanced features
   - Opportunities for shared configurations
   - Security improvements (network restrictions, sandbox)

4. **Performance Enhancements**:
   - Workflows that could benefit from repo-memory caching
   - Opportunities to use more specific GitHub toolsets
   - Network allowlist optimizations
   - Timeout tuning

### Phase 4: Generate Recommendations

**Goal**: Provide actionable insights

For each missed opportunity identified in Phase 3:

1. **Prioritize by Impact**:
   - High: Security improvements, significant performance gains
   - Medium: Developer experience, consistency
   - Low: Nice-to-haves, minor optimizations

2. **Provide Specific Examples**:
   - Which workflows would benefit
   - How to implement the change
   - Expected benefits

3. **Consider Trade-offs**:
   - Complexity vs. benefit
   - Maintenance burden
   - Learning curve

### Phase 5: Use Repo Memory for Persistence

**Goal**: Track research over time and enable trend analysis

Use the repo-memory tool to maintain research history:

1. **Save Current Analysis**:
   ```bash
   mkdir -p /tmp/gh-aw/repo-memory/default/copilot-cli-research/
   
   # Save timestamp and summary
   cat > /tmp/gh-aw/repo-memory/default/copilot-cli-research/latest.json <<EOF
   {
     "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
     "total_workflows": <count>,
     "copilot_workflows": <count>,
     "features_available": [<list>],
     "features_used": [<list>],
     "opportunities_found": <count>
   }
   EOF
   ```

2. **Load Previous Analysis** (if exists):
   ```bash
   if [ -f /tmp/gh-aw/repo-memory/default/copilot-cli-research/latest.json ]; then
     cat /tmp/gh-aw/repo-memory/default/copilot-cli-research/latest.json
     # Compare with current findings to show trends
   fi
   ```

3. **Maintain Research Notes**:
   - Create `memory/copilot-cli-research/notes.md` with ongoing observations
   - Track which recommendations have been implemented
   - Note new features as they're added

## Output Format

Create a GitHub discussion with your comprehensive findings:

### Discussion Title
`Copilot CLI Deep Research - [Current Date]`

### Discussion Structure

```markdown
# 🔍 Copilot CLI Deep Research Report

**Analysis Date**: [Date]
**Repository**: ${{ github.repository }}
**Scope**: [X] total workflows, [Y] using Copilot engine

---

## 📊 Executive Summary

[2-3 paragraphs summarizing key findings, most important opportunities, and overall assessment]

---

## 1️⃣ Current State Analysis

### Copilot CLI Capabilities Inventory
- **Version Information**: [Current version used]
- **Available Features**: [List of all documented features]
- **Configuration Options**: [CLI flags, engine config, etc.]

### Usage Statistics
- **Total Workflows**: [count]
- **Copilot Workflows**: [count] ([percentage]%)
- **Most Common Tools**: [list]
- **Most Common Configurations**: [patterns]

---

## 2️⃣ Feature Usage Matrix

| Feature Category | Available Features | Used | Not Used | Usage Rate |
|------------------|-------------------|------|----------|------------|
| CLI Flags | [list] | [list] | [list] | [%] |
| Engine Config | [list] | [list] | [list] | [%] |
| MCP Servers | [list] | [list] | [list] | [%] |
| Network Config | [list] | [list] | [list] | [%] |
| Sandbox Options | [list] | [list] | [list] | [%] |

---

## 3️⃣ Missed Opportunities

### 🔴 High Priority

#### Opportunity 1: [Title]
- **What**: [Description of the unused feature]
- **Why It Matters**: [Impact/benefit]
- **Where**: [Which workflows could benefit]
- **How to Implement**: [Specific steps or example]
- **Example**:
  ```yaml
  [Code example]
  ```

[Repeat for each high-priority opportunity]

### 🟡 Medium Priority

[Same structure as high priority]

### 🟢 Low Priority

[Same structure as high priority]

---

## 4️⃣ Specific Workflow Recommendations

### Workflow: `example-workflow.md`
- **Current State**: [brief description]
- **Recommended Changes**: [list of specific improvements]
- **Expected Benefits**: [what improvements would bring]

[Repeat for notable workflows]

---

## 5️⃣ Trends & Insights

[If previous research exists in repo-memory]
- **Changes Since Last Analysis**: [what's improved or changed]
- **Adoption Trends**: [are recommendations being implemented?]
- **New Features**: [what's been added to Copilot CLI]

[If no previous research]
- This is the first comprehensive analysis. Future research will track trends.

---

## 6️⃣ Best Practice Guidelines

Based on this research, here are recommended best practices:

1. **[Practice 1]**: [Description and rationale]
2. **[Practice 2]**: [Description and rationale]
3. **[Practice 3]**: [Description and rationale]

---

## 7️⃣ Action Items

**Immediate Actions** (this week):
- [ ] [Action 1]
- [ ] [Action 2]

**Short-term** (this month):
- [ ] [Action 3]
- [ ] [Action 4]

**Long-term** (this quarter):
- [ ] [Action 5]
- [ ] [Action 6]

---

## 📚 References

- Copilot Engine Documentation: [link]
- GitHub Agentic Workflows Instructions: [link]
- Related Workflows: [links]
- Previous Research: [link to repo-memory if exists]

---

_Generated by Copilot CLI Deep Research (Run: ${{ github.run_id }})_
```

## Important Guidelines

### Research Quality
- **Be thorough**: Review ALL Copilot-related code files and documentation
- **Be specific**: Provide concrete examples and code snippets
- **Be accurate**: Verify all claims by checking actual code/config
- **Be actionable**: Every recommendation should be implementable

### Analysis Depth
- Don't just list features - analyze WHY they're not being used
- Consider the trade-offs and context for each recommendation
- Look for patterns and themes across multiple workflows
- Think about the developer experience and learning curve

### Repo Memory Usage
- Always check for previous analysis to show progress over time
- Save comprehensive data for future trend analysis
- Keep notes organized and structured for easy retrieval
- Update the analysis after each run

### Discussion Quality
- Use clear headings and structure for easy navigation
- Include code examples and specific workflow names
- Prioritize recommendations by impact
- Make it easy to scan and find key information

## Success Criteria

A successful research report should:
- ✅ Identify at least 5-10 missed opportunities
- ✅ Provide specific, actionable recommendations with examples
- ✅ Use data and statistics to support findings
- ✅ Save analysis to repo-memory for future tracking
- ✅ Create a well-structured, readable discussion
- ✅ Reference actual code and workflows by name
- ✅ Include both quick wins and long-term improvements
- ✅ Consider security, performance, and developer experience

**Remember**: The goal is to help the team make better use of Copilot CLI's capabilities and improve the overall quality of agentic workflows in this repository.
