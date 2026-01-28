---
title: Agent Behavior
description: Technical reference for how AI agents behave in GitHub Agentic Workflows, including complexity detection, response calibration, and override mechanisms
sidebar:
  order: 950
---

This document provides technical details about how AI agents behave when processing workflow requests in GitHub Agentic Workflows.

## Complexity Detection Algorithm

Agents automatically analyze workflow requests to determine appropriate response complexity using a scoring system.

### Scoring Categories

#### Basic Indicators (1 point each)

Presence of any of these indicators adds 1 point to the complexity score:

- **Single trigger keywords**: `push`, `pull_request`, `issues`, `discussion`, `workflow_dispatch`
- **Simple action verbs**: `run`, `check`, `test`, `lint`, `build`, `deploy`, `post`, `comment`
- **No tool mentions**: Request doesn't mention tools, MCP servers, or APIs
- **No conditional language**: Lacks `if`, `when`, `based on`, `depending on`
- **No scheduling**: Doesn't mention `schedule`, `cron`, `daily`, `weekly`

**Example:** "Run tests on pull requests"
- Single trigger (`pull_request`): +1
- Simple verb (`run`): +1
- No tools: +0 (absence of complexity)
- **Total: 2 points → Basic Tier**

#### Intermediate Indicators (2 points each)

Presence of any of these indicators adds 2 points:

- **Multiple triggers**: More than one trigger type mentioned
- **Conditional language**: `if`, `when`, `based on`, `depending on`, `according to`
- **Tool integration**: Mentions `bash`, `jq`, `gh`, GitHub API, safe-outputs, safe-inputs
- **Schedule requirements**: `schedule`, `cron`, `daily`, `weekly`, `hourly`
- **Team/assignee logic**: Mentions `assign`, `route`, `team`, `owner`, `maintainer`
- **Label/milestone operations**: Operations on labels, milestones, or project boards
- **File/path conditions**: Logic based on changed files or directory structure

**Example:** "Triage issues using labels and assign to team members based on expertise"
- Single trigger (`issues`): +1
- Tool integration (labels API): +2
- Conditional language (`based on`): +2
- Team logic (`assign`, `team members`): +2
- **Total: 7 points → Intermediate Tier**

#### Advanced Indicators (3 points each)

Presence of any of these indicators adds 3 points:

- **Multi-stage/orchestration**: `campaign`, `multi-repo`, `orchestrate`, `coordinate`, `hierarchical`
- **State management**: `repo-memory`, `persistence`, `state machine`, `workflow state`
- **Performance optimization**: `caching`, `parallel`, `concurrent`, `optimize`, `performance`
- **Security/compliance**: `strict mode`, `security`, `compliance`, `audit`, `vulnerability`
- **Custom MCP servers**: Mentions custom or non-standard MCP servers
- **Custom engines**: Non-default engine configuration
- **Complex error handling**: `retry`, `fallback`, `circuit breaker`, `error recovery`
- **Multi-repository**: `multi-repo`, `cross-repo`, `repository scan`, `organization-wide`

**Example:** "Multi-repo security scanning campaign with state persistence and automated remediation"
- Multi-stage (`campaign`, `multi-repo`): +3
- State management (`persistence`): +3
- Security (`security scanning`): +3
- Complex logic (`automated remediation`): +3
- **Total: 12 points → Advanced Tier**

### Tier Assignment Rules

```
if total_score >= 8:
    tier = "advanced"
elif total_score >= 4:
    tier = "intermediate"
else:
    tier = "basic"
```

**Thresholds:**
- **Basic Tier**: 1-3 points
- **Intermediate Tier**: 4-7 points
- **Advanced Tier**: 8+ points

### Override Mechanisms

User-provided override phrases supersede automatic detection:

#### Force Basic Tier

These phrases force basic tier regardless of calculated score:

- "keep it simple"
- "minimal"
- "just the basics"
- "quick and easy"
- "bare bones"
- "no frills"
- "straightforward"

**Example:** "Multi-repo security scanning (keep it simple)"
- Calculated score: 9 (Advanced)
- Override: "keep it simple" → **Basic Tier**

#### Force Advanced Tier

These phrases force advanced tier regardless of calculated score:

- "comprehensive"
- "production-ready"
- "enterprise-grade"
- "with all options"
- "full documentation"
- "detailed guide"
- "complete reference"

**Example:** "Run tests on pull requests (comprehensive guide)"
- Calculated score: 2 (Basic)
- Override: "comprehensive guide" → **Advanced Tier**

## Response Calibration

Once tier is determined, agents calibrate their responses across multiple dimensions:

### Documentation Depth

| Tier | Word Count | Paragraph Count | Structure |
|------|-----------|----------------|-----------|
| **Basic** | 50-150 | 1-2 | Minimal: description + example |
| **Intermediate** | 200-500 | 3-5 | Moderate: overview + use cases + examples + options |
| **Advanced** | 500+ | 6+ | Comprehensive: architecture + patterns + examples + best practices + troubleshooting |

### Example Complexity

| Tier | Count | Characteristics |
|------|-------|----------------|
| **Basic** | 1 | Minimal working example, no variations |
| **Intermediate** | 2-3 | Basic + advanced examples showing options |
| **Advanced** | 3+ | Basic + production + edge cases + patterns |

### Code Comments

| Tier | Density | Focus |
|------|---------|-------|
| **Basic** | Minimal | Only non-obvious logic |
| **Intermediate** | Moderate | Explain conditionals and integrations |
| **Advanced** | Extensive | Architecture, patterns, trade-offs |

### Feature Suggestions

| Tier | Policy | Format |
|------|--------|--------|
| **Basic** | None | No suggestions unless requested |
| **Intermediate** | Relevant | Brief mentions with links |
| **Advanced** | Proactive | Dedicated section with recommendations |

### Code-to-Text Ratio

| Tier | Ratio | Emphasis |
|------|-------|----------|
| **Basic** | High | Show, don't tell |
| **Intermediate** | Balanced | Equal code and explanation |
| **Advanced** | Lower | Context and guidance |

## Quality Standards

All tiers maintain identical quality standards:

### Required Elements (All Tiers)

✅ **Correct syntax**: Valid YAML and workflow structure  
✅ **Working code**: Examples that execute successfully  
✅ **Error handling**: Appropriate to complexity level  
✅ **Security**: Best practices applied  
✅ **Documentation**: Clear and professional

### Variable Elements (By Tier)

📊 **Documentation depth**: Adjusted to tier  
📊 **Example count**: More examples in higher tiers  
📊 **Feature coverage**: Comprehensive in advanced, focused in basic  
📊 **Optimization guidance**: Present in advanced, minimal in basic

**Key Principle**: The difference is depth and sophistication, not quality or correctness.

## Implementation Patterns

### Basic Tier Template

```markdown
## [Feature Name]

[1-2 sentence description of what it does]

**Example:**

```yaml
[minimal working example]
```

**Configuration:**
- `required-field`: Description
- `optional-field`: Description (optional)
```

### Intermediate Tier Template

```markdown
## [Feature Name]

[3-5 paragraph explanation covering:]
- What it does and why it's useful
- When to use it
- How it integrates with other features
- Common use cases

**Examples:**

### Basic Usage
[Simple example]

### With Options
[Example showing configuration]

**Configuration Options:**

| Option | Type | Description | Default |
|--------|------|-------------|---------|
[Table of common options]

**Related Features:** [Links to related documentation]
```

### Advanced Tier Template

```markdown
## [Feature Name]

[Comprehensive explanation covering:]
- Overview and architecture
- Design rationale and trade-offs
- Integration points and dependencies
- Use cases and patterns
- Performance characteristics
- Security implications

**Examples:**

### Quick Start
[Minimal example]

### Production Configuration
[Real-world example]

### Advanced Patterns
[Complex integration]

### Edge Cases
[Handling special scenarios]

**Complete Configuration Reference:**

[Detailed table with all options]

**Architecture:**

[Diagram or detailed explanation of how it works]

**Best Practices:**

1. [Practice 1 with rationale]
2. [Practice 2 with rationale]
3. [Practice 3 with rationale]

**Performance Tuning:**

- [Optimization 1]
- [Optimization 2]
- [Optimization 3]

**Security Considerations:**

- [Security practice 1]
- [Security practice 2]
- [Security practice 3]

**Troubleshooting:**

| Issue | Cause | Solution |
|-------|-------|----------|
[Common issues table]

**Related Resources:**

- [Link 1]
- [Link 2]
- [Link 3]
```

## Testing and Validation

### Test Scenarios

Implementations should be validated against these reference scenarios:

#### Basic Tier Test

**Input:** "Run tests on pull requests"

**Expected Behavior:**
- Tier: Basic (score: 2)
- Documentation: 50-150 words
- Examples: 1 minimal example
- Comments: Minimal
- Suggestions: None

#### Intermediate Tier Test

**Input:** "Triage issues using labels and assign to team members based on expertise"

**Expected Behavior:**
- Tier: Intermediate (score: 5-7)
- Documentation: 200-500 words
- Examples: 2-3 variations
- Comments: Moderate (explain logic)
- Suggestions: Mention related features

#### Advanced Tier Test

**Input:** "Multi-repo security scanning campaign with state persistence"

**Expected Behavior:**
- Tier: Advanced (score: 9+)
- Documentation: 500+ words
- Examples: 3+ with edge cases
- Comments: Extensive (architecture)
- Suggestions: Proactive optimizations

#### Override Test

**Input:** "Run tests on pull requests (comprehensive, production-ready)"

**Expected Behavior:**
- Tier: Advanced (forced by override)
- Documentation: 500+ words
- Examples: 3+ with edge cases
- Comments: Extensive
- Suggestions: Include optimization recommendations

## Monitoring and Feedback

### Success Metrics

Track these metrics to evaluate calibration effectiveness:

- **Tier detection accuracy**: % of requests correctly classified
- **User satisfaction**: Feedback on response appropriateness
- **Iteration rate**: How often users request more/less detail
- **Time to completion**: Faster for basic, appropriate depth for advanced

### Feedback Loop

Users can provide feedback to improve calibration:

1. **Too much detail**: "This is more complex than I needed"
2. **Too little detail**: "Can you provide more comprehensive documentation?"
3. **Wrong tier**: "This should be [basic/intermediate/advanced]"

Feedback is used to:
- Refine indicator weights
- Adjust tier thresholds
- Improve override detection
- Enhance calibration patterns

## Related Documentation

- **User Guide**: [Workflow Complexity Tiers](/gh-aw/guides/complexity-tiers/) - How to use complexity tiers
- **Developer Spec**: [Complexity Calibration](https://github.com/githubnext/gh-aw/blob/main/specs/complexity-calibration.md) - Complete technical specification
- **Agent Instructions**: [developer.instructions.md](https://github.com/githubnext/gh-aw/blob/main/.github/agents/developer.instructions.md) - Agent-specific guidelines

---

**Version**: 1.0  
**Last Updated**: 2026-01-28  
**Status**: Active
