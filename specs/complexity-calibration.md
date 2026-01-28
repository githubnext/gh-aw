# Complexity Calibration for Workflow Responses

## Overview

This specification defines the complexity tier system for GitHub Agentic Workflows, enabling agents to calibrate documentation depth and feature sophistication based on request complexity. The system addresses the finding from research (githubnext/gh-aw#12193) that agents produce uniformly high-quality output without differentiating between simple and complex use cases.

## Problem Statement

Current agent behavior shows:
- **Uniform quality scores** (4.8-5.0) across all scenarios
- **No calibration** between basic and advanced requests
- **Over-engineering** for simple use cases
- **Cognitive overload** from excessive detail in straightforward scenarios

## Solution: Three-Tier Complexity System

### Tier Definitions

#### Basic Tier
**Characteristics:**
- Single, standard GitHub event trigger (push, pull_request, issues)
- No or minimal tool usage (bash commands, basic GitHub operations)
- Straightforward linear workflow logic
- Standard output formats
- No custom configuration or advanced features

**Examples:**
- "Run tests on pull requests"
- "Label issues based on title keywords"
- "Post a comment when a PR is opened"
- "Run linter on push to main branch"

**Agent Calibration:**
- **Documentation depth**: Concise explanations (1-2 paragraphs)
- **Example complexity**: Single, minimal working example
- **Feature suggestions**: None unless requested
- **Code comments**: Minimal, only for non-obvious logic
- **Error handling**: Basic GitHub Actions error propagation
- **Configuration options**: Show only required fields

#### Intermediate Tier
**Characteristics:**
- Multiple triggers or conditional workflows
- Tool combinations (bash + GitHub API, MCP servers)
- Conditional logic and branching
- Moderate configuration requirements
- Basic safe-outputs or safe-inputs usage

**Examples:**
- "Triage issues using labels and assign to team members based on expertise"
- "Run different test suites based on changed files"
- "Create PRs with dynamic content from external APIs"
- "Scheduled workflow that analyzes repository health metrics"

**Agent Calibration:**
- **Documentation depth**: Moderate explanations (3-5 paragraphs)
- **Example complexity**: 2-3 examples showing variations
- **Feature suggestions**: Mention related features when relevant
- **Code comments**: Explain conditional logic and tool integration
- **Error handling**: Include retry logic and informative error messages
- **Configuration options**: Show common configurations with defaults

#### Advanced Tier
**Characteristics:**
- Complex multi-trigger workflows with dependencies
- Custom toolchains and MCP server integrations
- Multi-stage processes with state management
- Performance optimization requirements
- Extensive safe-outputs/safe-inputs configuration
- Repository memory or campaign-based orchestration
- Security considerations and strict mode usage

**Examples:**
- "Multi-repo campaign for security vulnerability detection and remediation"
- "Hierarchical agent system for project management with delegation"
- "Performance-optimized workflow with caching and parallel execution"
- "Complex state machine with repo-memory persistence"

**Agent Calibration:**
- **Documentation depth**: Comprehensive explanations (6+ paragraphs)
- **Example complexity**: Multiple detailed examples with edge cases
- **Feature suggestions**: Proactively suggest optimization patterns
- **Code comments**: Extensive documentation of architecture and patterns
- **Error handling**: Sophisticated error recovery with fallbacks
- **Configuration options**: Show advanced options with security implications
- **Performance considerations**: Discuss timeout, concurrency, caching

## Request Complexity Detection

### Detection Signals

Agents should analyze requests for these complexity indicators:

#### Basic Indicators (Score: 1 point each)
- Single trigger keyword (push, pull_request, issues, etc.)
- Simple action verbs ("run", "check", "post")
- No tool mentions
- No conditional requirements
- No scheduling requirements

#### Intermediate Indicators (Score: 2 points each)
- Multiple triggers mentioned
- Conditional language ("if", "when", "based on")
- Tool integration mentioned (bash, GitHub API, jq)
- Schedule requirements
- Safe-outputs or safe-inputs mentioned
- Team/assignee logic

#### Advanced Indicators (Score: 3 points each)
- Multi-stage or orchestration keywords
- State management requirements
- Performance or optimization requirements
- Security/strict mode requirements
- Custom MCP servers or engines
- Campaign or multi-repo mentions
- Repo-memory or persistence requirements
- Complex error handling requirements

### Scoring Logic

```
Total Score = Sum of all indicator points

Basic Tier: Score 1-3
Intermediate Tier: Score 4-7
Advanced Tier: Score 8+
```

### Override Signals

Explicit complexity indicators override automatic detection:

- **Force Basic**: "Keep it simple", "minimal", "just the basics"
- **Force Advanced**: "comprehensive", "production-ready", "enterprise-grade", "with all options"

## Documentation Depth Guidelines

### Basic Tier Documentation

**Structure:**
```markdown
## [Feature Name]

[1-2 sentence description]

**Example:**
```yaml
[minimal working example]
```

**Configuration:**
- `field`: Description (required/optional)
```

**Length**: 50-150 words
**Code-to-text ratio**: High (more code, less text)

### Intermediate Tier Documentation

**Structure:**
```markdown
## [Feature Name]

[3-5 paragraph explanation covering:]
- What it does
- When to use it
- How it works
- Common use cases

**Examples:**

### Basic Example
[Simple case]

### Advanced Example
[With options]

**Configuration Options:**
| Option | Type | Description | Default |
|--------|------|-------------|---------|
[Table of common options]

**See Also:** [Related features]
```

**Length**: 200-500 words
**Code-to-text ratio**: Balanced

### Advanced Tier Documentation

**Structure:**
```markdown
## [Feature Name]

[Comprehensive explanation covering:]
- Overview and architecture
- Use cases and patterns
- Integration points
- Performance considerations
- Security implications

**Examples:**

### Basic Usage
[Simple example]

### Production Configuration
[Real-world example]

### Advanced Patterns
[Complex integration]

**Configuration Reference:**
[Complete configuration table with all options]

**Best Practices:**
- [Practice 1]
- [Practice 2]
- [Practice 3]

**Troubleshooting:**
[Common issues and solutions]

**Performance Tuning:**
[Optimization guidelines]

**Security Considerations:**
[Security best practices]

**See Also:** [Related features and resources]
```

**Length**: 500+ words
**Code-to-text ratio**: Lower (more explanation, context, and guidance)

## Feature Suggestion Guidelines

### Basic Tier
- **Suggestion policy**: No suggestions unless explicitly requested
- **Rationale**: Avoid overwhelming simple requests with unnecessary complexity

### Intermediate Tier
- **Suggestion policy**: Mention directly related features
- **Format**: Brief one-line mentions with links
- **Example**: "You might also consider using `safe-outputs` for GitHub API operations."

### Advanced Tier
- **Suggestion policy**: Proactively suggest optimization patterns
- **Format**: Dedicated section with detailed recommendations
- **Example**: Include a "Recommended Optimizations" section discussing caching, concurrency, and performance patterns

## Implementation Locations

### 1. Agent Instructions
**File**: `.github/agents/developer.instructions.md`

Add section:
```markdown
## Complexity Calibration

When generating workflow responses, detect and calibrate to request complexity:

[Tier definitions]
[Detection guidelines]
[Calibration instructions]
```

### 2. AGENTS.md
**File**: `AGENTS.md`

Add section after "## Important: Using Skills":
```markdown
## Response Complexity Calibration

Agents calibrate response complexity based on request sophistication...
```

### 3. User Documentation
**File**: `docs/src/content/docs/guides/complexity-tiers.md`

New guide document explaining:
- How complexity detection works
- How to request specific complexity levels
- Examples of each tier
- When to use each tier

### 4. Reference Documentation
**File**: `docs/src/content/docs/reference/agent-behavior.md`

New reference document covering:
- Technical details of complexity detection
- Scoring algorithm
- Override mechanisms
- Calibration guidelines for contributors

## Quality Assurance

### Acceptance Criteria

- [ ] Agent correctly detects request complexity (basic/intermediate/advanced)
- [ ] Documentation depth adjusts appropriately for each tier
- [ ] Basic requests receive focused, concise output
- [ ] Advanced requests receive comprehensive, detailed output
- [ ] Quality remains consistently high across all tiers
- [ ] Manual testing validates appropriate tier selection for diverse scenarios

### Testing Approach

**Test Scenarios:**

1. **Basic Tier Validation**
   - Request: "Run tests on pull requests"
   - Expected: Minimal documentation, single example, no feature suggestions
   
2. **Intermediate Tier Validation**
   - Request: "Triage issues and assign based on labels"
   - Expected: Moderate documentation, 2-3 examples, related feature mentions

3. **Advanced Tier Validation**
   - Request: "Multi-repo security scanning campaign with state persistence"
   - Expected: Comprehensive documentation, multiple examples, optimization suggestions

4. **Override Testing**
   - Request: "Run tests on pull requests (comprehensive guide)"
   - Expected: Advanced tier output despite basic request

### Success Metrics

- **Consistency**: 90%+ accuracy in tier detection across test scenarios
- **User Satisfaction**: Reduced feedback about "too much" or "too little" documentation
- **Efficiency**: Faster workflow creation for simple use cases
- **Depth**: More thorough guidance for complex use cases

## Migration Strategy

1. **Phase 1**: Add complexity detection to developer instructions (non-breaking)
2. **Phase 2**: Update AGENTS.md with calibration guidelines
3. **Phase 3**: Add user-facing documentation
4. **Phase 4**: Monitor usage and adjust tier boundaries based on feedback

## Future Enhancements

- **Adaptive learning**: Track user feedback to refine tier boundaries
- **Explicit tier request**: Allow users to specify desired complexity level in frontmatter
- **Context awareness**: Consider repository context (size, activity) in tier selection
- **Progressive disclosure**: Start with basic tier, offer "show more" for additional detail

## References

- Issue: githubnext/gh-aw#12193 - "Recommendations" section: "Calibrate Complexity"
- Research findings: Uniform quality scores (4.8-5.0) across all scenarios
- User feedback: Requests for simpler workflows and more detailed guides

---

**Version**: 1.0  
**Status**: Active  
**Last Updated**: 2026-01-28
