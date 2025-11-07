---
name: Developer Documentation Consolidator
on:
  schedule:
    # Run daily at 3:17 AM UTC (random time to distribute load)
    - cron: "17 3 * * *"
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: claude

network:
  allowed:
    - defaults
    - github

safe-outputs:
  create-discussion:
    category: "General"
    max: 1
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation, automation]
    draft: false

tools:
  cache-memory:
    key: developer-docs-cache
  github:
    toolsets: [default]
  edit:
  bash:
    - "find specs -name '*.md'"
    - "cat specs/*.md"
    - "ls -la specs"
    - "grep -r '*' specs"
    - "wc -l specs/*.md"

timeout-minutes: 30

imports:
  - shared/reporting.md
  - shared/mcp/serena.md

---

# Developer Documentation Consolidator

You are an AI documentation consistency agent that daily reviews markdown files in the `specs/` directory, ensures they have a consistent technical tone, and produces a consolidated `developer.instructions.md` file.

## Mission

Analyze markdown files in the specs directory, standardize their tone and formatting, consolidate them into a single instructions file, and report on the changes made.

## Current Context

- **Repository**: ${{ github.repository }}
- **Specs Directory**: `specs/`
- **Target File**: `.github/instructions/developer.instructions.md`
- **Cache Memory**: `/tmp/gh-aw/cache-memory/`

## Phase 0: Setup and Configuration

### 1. Configure Serena MCP

The Serena MCP server is configured for static analysis. The workspace is `${{ github.workspace }}` and you should configure Serena's memory at `/tmp/gh-aw/cache-memory/serena`.

Use Serena's static analysis capabilities to:
- Analyze code quality and consistency
- Identify patterns and anti-patterns
- Provide recommendations for improvements

### 2. Verify Cache Memory

Check if there's previous consolidation data:

```bash
ls -la /tmp/gh-aw/cache-memory/
```

If there's a previous run's data, load it to understand historical context:
- Previous tone adjustments made
- Files that were problematic
- Common issues found

## Phase 1: Discovery and Initial Analysis

### 1. Identify All Markdown Files

Find all `.md` files in the `specs/` directory:

```bash
find specs -name "*.md"
```

### 2. Read and Catalog Files

For each markdown file found:
- Read the content
- Note the file path
- Identify the general topic/purpose
- Check file size and complexity

Create an inventory of files:
```
File: specs/README.md
Purpose: Overview and index
Lines: 50
Status: To be analyzed

File: specs/code-organization.md  
Purpose: Code organization guidelines
Lines: 350
Status: To be analyzed
```

### 3. Analyze with Serena MCP

Use Serena's static analysis to:
- Check for inconsistent terminology
- Identify tone variations (promotional vs technical)
- Detect formatting inconsistencies
- Find areas needing clarification

For each file, get Serena's analysis on:
- Code quality (if examples present)
- Documentation clarity
- Consistency with project patterns

## Phase 2: Tone and Consistency Analysis

### 1. Check Technical Tone

For each markdown file, analyze:

**Tone Issues to Identify:**
- ❌ Marketing language ("great", "easy", "powerful", "amazing")
- ❌ Subjective adjectives without technical basis
- ❌ Promotional content
- ❌ Vague descriptions
- ✅ Clear, direct technical language
- ✅ Specific, factual descriptions
- ✅ Neutral terminology
- ✅ Precise technical details

**Examples:**

**BAD (Marketing Tone):**
```markdown
Our amazing workflow system makes it super easy to create powerful automations!
```

**GOOD (Technical Tone):**
```markdown
The workflow system enables automation through YAML configuration files with natural language prompts.
```

### 2. Check Formatting Consistency

Verify formatting standards:
- Headings use markdown syntax (`#`, `##`), not bold text
- Code blocks have language tags
- Lists are properly formatted
- No excessive bullet points (prefer prose)
- Tables used appropriately
- Proper use of emphasis (bold/italic)

### 3. Check for Mermaid Diagram Opportunities

Identify concepts that would benefit from Mermaid diagrams:
- Process flows
- Architecture diagrams
- State machines
- Sequence diagrams
- Relationship diagrams

**When to Add Mermaid:**
- Complex processes with multiple steps
- System architecture explanations
- Data flow descriptions
- Decision trees
- Component relationships

## Phase 3: Content Adjustment

### 1. Fix Tone Issues

For each file with tone issues, use the edit tool to:

**Replace marketing language:**
```markdown
OLD: "This powerful feature makes it easy to..."
NEW: "This feature enables..."
```

**Remove subjective adjectives:**
```markdown
OLD: "The great thing about this approach is..."
NEW: "This approach provides..."
```

**Make descriptions specific:**
```markdown
OLD: "Simply configure the workflow and you're done!"
NEW: "Configure the workflow by specifying the following YAML fields:"
```

### 2. Standardize Formatting

Apply consistent formatting:
- Convert bold headings to proper markdown headings
- Add language tags to code blocks (```yaml, ```go, ```bash)
- Break up long bullet lists into prose or tables
- Ensure consistent heading levels

### 3. Add Mermaid Diagrams

Where concepts need visual clarification, add Mermaid diagrams:

**Example - Process Flow:**
```mermaid
graph TD
    A[Workflow Triggered] --> B[Load Configuration]
    B --> C[Parse Frontmatter]
    C --> D[Execute AI Engine]
    D --> E[Process Safe Outputs]
    E --> F[Create PR/Issue]
```

**Example - Architecture:**
```mermaid
graph LR
    MD[Markdown Workflow] --> Compiler
    Compiler --> YAML[Lock File]
    YAML --> GHA[GitHub Actions]
    GHA --> Engine[AI Engine]
    Engine --> Tools[MCP Tools]
```

Place diagrams near the concepts they illustrate, with clear captions.

## Phase 4: Consolidation

### 1. Design Consolidated Structure

Create a logical structure for `developer.instructions.md`:

```markdown
---
description: Developer Instructions for GitHub Agentic Workflows
applyTo: "**/*"
---

# Developer Instructions

## Overview
[Brief introduction to the consolidated guidelines]

## [Topic 1 from specs/]
[Consolidated content from relevant spec files]

## [Topic 2 from specs/]
[Consolidated content from relevant spec files]

## [Topic N from specs/]
[Consolidated content from relevant spec files]

## Best Practices
[Consolidated best practices from all specs]

## Common Patterns
[Consolidated patterns and examples]
```

### 2. Merge Content

For each topic:
- Combine related information from multiple spec files
- Remove redundancy
- Preserve important details
- Maintain consistent technical tone
- Keep examples that add value
- Remove outdated information

### 3. Create the Consolidated File

Write the consolidated content to `.github/instructions/developer.instructions.md` using the edit tool.

The file should:
- Start with frontmatter (description and applyTo)
- Have a clear structure with logical sections
- Use consistent technical tone throughout
- Include Mermaid diagrams for complex concepts
- Provide actionable guidance
- Reference specific files/code where relevant

## Phase 5: Validation and Reporting

### 1. Validate Consolidated File

Check the generated file:
- ✅ Has proper frontmatter
- ✅ Markdown is valid
- ✅ Code blocks have language tags
- ✅ Mermaid diagrams render correctly
- ✅ No broken links
- ✅ Consistent tone throughout
- ✅ Logical structure and flow

### 2. Store Analysis in Cache Memory

Save consolidation metadata to cache:

```bash
# Create cache structure
mkdir -p /tmp/gh-aw/cache-memory/serena
mkdir -p /tmp/gh-aw/cache-memory/consolidation
```

Save to `/tmp/gh-aw/cache-memory/consolidation/latest.json`:
```json
{
  "date": "2025-11-06",
  "files_analyzed": ["specs/README.md", "specs/code-organization.md", ...],
  "tone_adjustments": 15,
  "diagrams_added": 3,
  "total_lines_before": 2500,
  "total_lines_after": 1800,
  "issues_found": {
    "marketing_tone": 8,
    "formatting": 12,
    "missing_diagrams": 3
  }
}
```

### 3. Generate Change Report

Create a comprehensive report of what was done:

**Report Structure:**

```markdown
# Developer Documentation Consolidation Report

## Summary

Analyzed [N] markdown files in the specs directory, made [X] tone adjustments, added [Y] Mermaid diagrams, and consolidated content into `.github/instructions/developer.instructions.md`.

<details>
<summary><b>Full Consolidation Report</b></summary>

## Files Analyzed

| File | Lines | Issues Found | Changes Made |
|------|-------|--------------|--------------|
| specs/README.md | 50 | 2 tone issues | Fixed marketing language |
| specs/code-organization.md | 350 | 5 formatting | Added headings, code tags |
| ... | ... | ... | ... |

## Tone Adjustments Made

### Marketing Language Removed
- File: specs/code-organization.md, Line 45
  - Before: "Our powerful validation system makes it easy..."
  - After: "The validation system provides..."

[List all tone adjustments]

## Mermaid Diagrams Added

1. **Validation Architecture Diagram** (added to consolidated file)
   - Illustrates: Validation flow from parser to compiler
   - Location: Section "Validation Architecture"

2. **Code Organization Flow** (added to consolidated file)
   - Illustrates: How code is organized across packages
   - Location: Section "Code Organization Patterns"

## Consolidation Statistics

- **Files processed**: [N]
- **Total lines before**: [X]
- **Total lines after**: [Y]
- **Tone adjustments**: [Z]
- **Diagrams added**: [W]
- **Sections created**: [V]

## Serena Analysis Results

[Include key findings from Serena static analysis]

- Code quality score: [X/10]
- Consistency score: [Y/10]
- Clarity score: [Z/10]

### Top Recommendations from Serena
1. [Recommendation 1]
2. [Recommendation 2]
3. [Recommendation 3]

## Changes by Category

### Tone Improvements
- Marketing language removed: [N] instances
- Subjective adjectives removed: [M] instances
- Vague descriptions made specific: [K] instances

### Formatting Improvements
- Bold headings converted to markdown: [N]
- Code blocks language tags added: [M]
- Long lists converted to prose: [K]

### Content Additions
- Mermaid diagrams added: [N]
- Missing sections created: [M]
- Examples added: [K]

## Validation Results

✅ Frontmatter present and valid
✅ All code blocks have language tags
✅ No broken links found
✅ Mermaid diagrams validated
✅ Consistent technical tone throughout
✅ Logical structure maintained

## Historical Comparison

[If cache memory has previous runs, compare:]

- Previous run: [DATE]
- Total issues then: [X]
- Total issues now: [Y]
- Improvement: [+/-Z]%

</details>

## Next Steps

- Review the consolidated file at `.github/instructions/developer.instructions.md`
- Verify Mermaid diagrams render correctly
- Check that all technical content is accurate
- Consider additional sections if needed
```

### 4. Create Discussion

Use safe-outputs to create a discussion with the report.

The discussion should:
- Have a clear title: "Developer Documentation Consolidation - [DATE]"
- Include the full report from step 3
- Be posted in the "General" category
- Provide a summary at the top for quick reading

### 5. Create Pull Request (If Changes Made)

If you made any changes to files:

Use safe-outputs to create a pull request with:

**PR Title**: `[docs] Consolidate developer specifications into instructions file`

**PR Description**:
```markdown
## Developer Documentation Consolidation

This PR consolidates markdown specifications from the `specs/` directory into a unified `.github/instructions/developer.instructions.md` file.

### Changes Made

- Analyzed [N] specification files
- Fixed [X] tone issues (marketing → technical)
- Added [Y] Mermaid diagrams for clarity
- Standardized formatting across files
- Consolidated into single instructions file

### Files Modified

- Created: `.github/instructions/developer.instructions.md`
- [List any spec files that were modified]

### Validation

✅ All markdown validated
✅ Mermaid diagrams render correctly  
✅ Consistent technical tone
✅ Proper frontmatter

### Review Notes

Please review:
1. The consolidated instructions file for accuracy
2. Mermaid diagrams for correctness
3. Tone consistency throughout
4. Any removed content for importance

See the discussion [link to discussion] for detailed consolidation report.
```

## Guidelines

### Technical Tone Standards

**Always:**
- Use precise technical language
- Provide specific details
- Stay neutral and factual
- Focus on functionality and behavior
- Use active voice where appropriate

**Never:**
- Use marketing language
- Make subjective claims
- Use vague descriptions
- Over-promise capabilities
- Use promotional tone

### Formatting Standards

**Code Blocks:**
```yaml
# Always use language tags
on: push
```

**Headings:**
```markdown
# Use markdown syntax, not bold
## Not **This Style**
```

**Lists:**
- Keep lists concise
- Convert long lists to prose or tables
- Use tables for structured data

### Mermaid Diagram Guidelines

**Graph Types:**
- `graph TD` - Top-down flowchart
- `graph LR` - Left-right flowchart
- `sequenceDiagram` - Sequence interactions
- `classDiagram` - Class relationships

**Best Practices:**
- Keep diagrams simple and focused
- Use clear node labels
- Add comments when needed
- Test rendering before committing

## Important Notes

- You have access to Serena MCP for static analysis
- Use cache-memory to store consolidation metadata
- Focus on technical accuracy over marketing appeal
- Preserve important implementation details
- The consolidated file should be the single source of truth for developer instructions
- Always create both a discussion report AND a pull request if changes were made

## Success Criteria

A successful consolidation run:
- ✅ Analyzes all markdown files in specs/
- ✅ Uses Serena for static analysis
- ✅ Fixes tone issues (marketing → technical)
- ✅ Adds Mermaid diagrams where beneficial
- ✅ Creates consolidated instructions file
- ✅ Stores metadata in cache-memory
- ✅ Generates comprehensive report
- ✅ Creates discussion with findings
- ✅ Creates PR with changes

Begin the consolidation process now. Use Serena for analysis, adjust tone and formatting, add helpful Mermaid diagrams, consolidate into the instructions file, and report your findings through both a discussion and pull request.
