# Markdown Formatting Guidelines

Use these guidelines when generating markdown content for reports, documentation, and other output.

## Markdown Flavor

Use **GitHub-flavored markdown** (GFM) throughout:
- Supports tables with pipe syntax
- Fenced code blocks with language identifiers
- Strikethrough with `~~text~~`
- Automatic URL linking
- Task lists with checkboxes
- Emoji support with `:emoji:`

## Header Levels

**Start all headers at level 3 (###)** to maintain consistent hierarchy:

```markdown
### Main Section

Content here...

#### Subsection

More details...

##### Detail Level

Specific information...
```

**Rationale**: Level 1 (#) and Level 2 (##) headers are typically reserved for document titles and major divisions. Starting at h3 provides better nesting within existing document structures.

## Task Lists and Checkboxes

Use checkbox syntax for actionable items and task tracking:

```markdown
### Next Steps

- [ ] Review security findings
- [ ] Apply suggested fixes
- [x] Complete initial analysis
- [ ] Update documentation
```

**Formatting rules**:
- Use `- [ ]` for unchecked items (space between brackets)
- Use `- [x]` for checked/completed items
- Maintain consistent indentation for nested tasks
- Keep checkbox items concise and actionable

**Nested tasks**:
```markdown
- [ ] Complete Phase 1
  - [x] Subtask 1
  - [x] Subtask 2
  - [ ] Subtask 3
```

## Progressive Disclosure

Use HTML `<details>` and `<summary>` tags to collapse long content and improve readability:

```markdown
<details>
<summary><b>Full Report Details</b></summary>

### Detailed Analysis

Long content that can be expanded...

- Additional data points
- Extensive lists
- Detailed breakdowns

</details>
```

**Key requirements**:
- **Always bold the summary text** using `<b>` tags
- Place summary text between `<summary>` tags
- Use descriptive summary text (e.g., "Full Report", "Technical Details", "All Findings")
- Maintain proper markdown formatting inside details blocks
- Start with blank line after `</summary>` for proper markdown rendering

**Multiple collapsible sections**:
```markdown
<details>
<summary><b>Section 1: Overview</b></summary>

Content for section 1...

</details>

<details>
<summary><b>Section 2: Details</b></summary>

Content for section 2...

</details>
```

## Report Structure

When creating reports, use this recommended structure:

### Overview

1-2 paragraphs summarizing key findings, metrics, or results.

### Key Findings

Highlight important discoveries or metrics using:
- Bullet points for lists
- Tables for comparative data
- Inline code for technical terms

### Detailed Analysis

<details>
<summary><b>Full Report</b></summary>

#### Complete Data

Extended content including:
- Comprehensive statistics
- Detailed breakdowns
- Supporting evidence
- Technical specifications

</details>

### Recommendations

- [ ] Action item 1
- [ ] Action item 2
- [ ] Action item 3

## Tables

Use pipe syntax for tables with proper alignment:

```markdown
| Metric | Value | Change |
|--------|-------|--------|
| Issues | 42    | +5     |
| PRs    | 18    | -2     |
```

**Alignment options**:
- Left: `|:---|`
- Center: `|:---:|`
- Right: `|---:|`

## Code Blocks

Use fenced code blocks with language identifiers:

````markdown
```yaml
name: Example
value: 123
```

```bash
gh aw compile workflow.md
```

```json
{"status": "success"}
```
````

## Links and References

### Workflow Run References

Format workflow run IDs as links with section symbol (§):

```markdown
[§12345](https://github.com/owner/repo/actions/runs/12345)
```

**Best practices**:
- Include up to 3 most relevant run URLs
- Group under `**References:**` section at end
- Use section symbol (§) prefix for run IDs
- Do NOT add footer attribution (system adds automatically)

### General Links

```markdown
[Link text](https://example.com)
[Link with title](https://example.com "Hover text")
```

## Lists

**Unordered lists**:
```markdown
- First item
- Second item
  - Nested item
  - Another nested item
- Third item
```

**Ordered lists**:
```markdown
1. First step
2. Second step
   1. Substep A
   2. Substep B
3. Third step
```

## Emphasis

- **Bold**: `**text**` or `__text__`
- *Italic*: `*text*` or `_text_`
- ***Bold and italic***: `***text***`
- ~~Strikethrough~~: `~~text~~`
- `Inline code`: `` `code` ``

## Horizontal Rules

Separate major sections with horizontal rules:

```markdown
---
```

## Best Practices

### Accessibility
- Use descriptive link text (avoid "click here")
- Provide alt text for images: `![Description](url)`
- Structure content with proper heading hierarchy

### Readability
- Keep paragraphs concise (2-4 sentences)
- Use lists for multiple related items
- Apply progressive disclosure for lengthy content
- Bold important terms on first use

### Consistency
- Use consistent heading levels throughout
- Maintain uniform list formatting
- Apply the same emphasis patterns
- Follow checkbox conventions for all task lists

### Technical Content
- Use code blocks for commands, code, and configuration
- Specify language identifiers for syntax highlighting
- Format file paths as inline code: `` `path/to/file` ``
- Use tables for structured data comparison

## Examples

### Complete Report Example

```markdown
### Executive Summary

This report analyzes 150 workflow runs from the past week, identifying 3 critical issues requiring immediate attention.

### Statistics

| Metric | Count | Status |
|--------|-------|--------|
| Total Runs | 150 | ✅ |
| Failed Runs | 8 | ⚠️ |
| Success Rate | 94.7% | ✅ |

### Action Items

- [ ] Fix authentication timeout in workflow A
- [ ] Update Node.js version to 20.x
- [x] Review security scan results

<details>
<summary><b>Detailed Findings by Workflow</b></summary>

#### Workflow: daily-report.md

- **Runs**: 45
- **Success Rate**: 97.8%
- **Issues**: None

#### Workflow: security-scan.md

- **Runs**: 30
- **Success Rate**: 86.7%
- **Issues**: 4 authentication timeouts

</details>

**References:**
- [§123456](https://github.com/owner/repo/actions/runs/123456)
- [§123457](https://github.com/owner/repo/actions/runs/123457)
```

### Task List Example

```markdown
### Implementation Checklist

- [x] Design phase complete
- [ ] Development in progress
  - [x] Backend API
  - [x] Database schema
  - [ ] Frontend UI
  - [ ] Integration tests
- [ ] Documentation pending
- [ ] Deployment scheduled
```

### Progressive Disclosure Example

```markdown
### Security Findings

Found 12 security issues across 5 workflows.

<details>
<summary><b>Critical Issues (3)</b></summary>

#### 1. Hardcoded API Token

- **Location**: workflow-a.md, line 45
- **Severity**: Critical
- **Fix**: Use GitHub Secrets

</details>

<details>
<summary><b>Medium Issues (6)</b></summary>

#### 1. Outdated Dependencies

- **Affected**: 3 workflows
- **Recommendation**: Update to latest versions

</details>

<details>
<summary><b>Low Issues (3)</b></summary>

Minor code quality improvements suggested.

</details>
```

## Remember

- **Headers**: Start at h3 (###)
- **Checkboxes**: Use `- [ ]` and `- [x]` for task lists
- **Progressive Disclosure**: Use `<details><summary><b>Text</b></summary>` with bold summaries
- **GitHub-Flavored**: Leverage GFM features (tables, task lists, code blocks)
- **Accessibility**: Maintain clear hierarchy and descriptive text
- **Consistency**: Follow these guidelines throughout all markdown output
