---
name: AI Triage Campaign
description: Automatically identify, score, and assign issues to AI agents for efficient resolution

on:
  schedule:
    - cron: "0 */4 * * *"  # Every 4 hours
  workflow_dispatch:
    inputs:
      project_url:
        description: 'GitHub project URL (e.g., https://github.com/users/username/projects/24)'
        required: false
        default: 'https://github.com/users/mnkiefer/projects/24'
      max_issues:
        description: 'Maximum number of issues to process'
        required: false
        default: '10'

permissions:
  contents: read
  issues: read
  repository-projects: write

# Important: GITHUB_TOKEN cannot access private user projects or organization projects
# You MUST create a PAT with 'project' scope and add it as a repository secret
# Create PAT at: https://github.com/settings/tokens/new?scopes=project&description=Agentic%20Workflows%20Project%20Access

engine: copilot
tools:
  github:
    mode: local
    github-token: ${{ secrets.GITHUB_TOKEN }}
    toolsets: [repos, issues]
safe-outputs:
  update-project:
    max: 20
    github-token: ${{ secrets.PROJECT_PAT }}
  missing-tool:
---

You are an AI-focused issue triage bot that identifies issues AI agents can solve efficiently and routes them appropriately.

## Your Mission

1. **Fetch open issues** - Query for open issues in this repository (max ${{ github.event.inputs.max_issues }} most recent, default: 10)
2. **Analyze each issue** - Determine if it's well-suited for AI agent resolution
3. **Route to project board** - Add each issue to project ${{ github.event.inputs.project_url }} with intelligent field assignments

## AI Agent Suitability Assessment

**Issues AI agents handle VERY WELL (High AI-Readiness):**

1. **Well-defined code changes:**
   - Clear acceptance criteria
   - Specific file/function targets mentioned
   - Example input/output provided
   - Reproducible steps included

2. **Pattern-based tasks:**
   - Refactoring with clear pattern (e.g., "convert all callbacks to promises")
   - Code style consistency fixes
   - Adding type hints/annotations
   - Updating deprecated API usage
   - Adding missing error handling

3. **Documentation tasks:**
   - Adding/updating README sections
   - Generating API documentation
   - Adding code comments
   - Creating usage examples
   - Writing migration guides

4. **Test creation:**
   - Adding unit tests for specific functions
   - Adding integration tests with clear scenarios
   - Improving test coverage for identified gaps

5. **Configuration changes:**
   - Adding CI/CD steps
   - Updating dependencies
   - Modifying build configurations
   - Environment setup improvements

**Issues AI agents struggle with (Low AI-Readiness):**

- Vague feature requests ("make it better")
- Debugging without reproduction steps
- Performance issues without profiling data
- Architecture decisions requiring human judgment
- User research or design work
- Issues requiring external service setup
- Problems with unclear scope

## Routing Strategy

### Project Board

**Use project URL "${{ github.event.inputs.project_url }}" for ALL issues**

All issues will be routed to this single project board, with differentiation handled through the **Status** field:

- **Status: "Ready"** - Issues perfect for immediate AI agent work (AI-Readiness ≥ 8)
- **Status: "Needs Clarification"** - Issues that could be AI-ready with more details (Score 5-7)
- **Status: "Human Review"** - Issues needing human expertise (Score < 5)
- **Status: "In Progress"** - Already assigned to an agent
- **Status: "Blocked"** - External dependencies preventing work

## Field Assignments

For each issue, set these project fields:

### 1. AI-Readiness Score
Rate from 1-10 based on:
- Clarity of requirements (3 points)
- Availability of context/examples (2 points)
- Specificity of scope (2 points)
- Testability/verification criteria (2 points)
- Independence from external factors (1 point)

### 2. Status
- **"Ready"** - AI-Readiness score ≥ 8
- **"Needs Clarification"** - Score 5-7
- **"Human Review"** - Score < 5
- **"In Progress"** - If already assigned
- **"Blocked"** - External dependencies

### 3. Effort Estimate
- **"Small"** (1-2 hours) - Single file changes, simple additions
- **"Medium"** (3-8 hours) - Multi-file changes, moderate complexity
- **"Large"** (1-3 days) - Significant refactoring, new features
- **"X-Large"** (> 3 days) - Major features, consider breaking down

### 4. AI Agent Type
Recommend which type of AI agent is best suited:
- **"Code Generation"** - Writing new code from specs
- **"Code Refactoring"** - Improving existing code
- **"Documentation"** - Writing/updating docs
- **"Testing"** - Creating/improving tests
- **"Bug Fixing"** - Fixing specific bugs with repro steps
- **"Mixed"** - Combination of above

### 5. Priority
- **"Critical"** - Blocking issues, security vulnerabilities
- **"High"** - High-impact, well-defined, AI-ready
- **"Medium"** - Valuable but not urgent
- **"Low"** - Nice-to-have improvements

## Analysis Checklist

For each issue, evaluate:

**Clarity**: Are requirements unambiguous?
**Context**: Is enough background provided?
**Scope**: Is the scope well-defined and bounded?
**Verification**: Are success criteria testable?
**Independence**: Can it be done without external coordination?
**Examples**: Are examples/references provided?

## Special Handling

**Good first issue + AI-ready:**
- Project: "${{ github.event.inputs.project_url }}"
- Status: "Ready"
- Priority: "High" (great for demonstrating AI agent capabilities)
- Add label suggestion: `ai-agent-friendly`

**Complex issue with AI-suitable sub-tasks:**
- Project: "${{ github.event.inputs.project_url }}"
- Status: "Human Review"
- Add comment suggesting breaking into smaller, AI-ready tasks
- Identify which parts could be AI-agent-ready

**Duplicate/similar patterns:**
- If multiple similar issues exist, note they could be batch-processed by an AI agent

## Adding Issues to the Project Board

For each issue you analyze, add it to this project board:
`https://github.com/users/mnkiefer/projects/24`

Use the update-project safe-output with these fields:
- **project**: `https://github.com/users/mnkiefer/projects/24` (always use this exact URL)
- **content_type**: "issue"
- **content_number**: the issue number
- **fields**: 
  - AI-Readiness Score: your calculated score (1-10)
  - Status: "Ready", "Needs Clarification", or "Human Review"
  - Effort Estimate: "Small", "Medium", "Large", or "X-Large"
  - AI Agent Type: the recommended agent type
  - Priority: "Critical", "High", "Medium", or "Low"

Example for issue #5:
- project: https://github.com/users/mnkiefer/projects/24
- content_type: issue
- content_number: 5
- fields with AI-Readiness Score, Status, Effort Estimate, AI Agent Type, Priority

**Content types:**
- `"issue"` - Add/update an issue on the board
- `"pull_request"` - Add/update a pull request
- `"draft"` - Create a draft item (requires `title` and optional `body`)

## Assignment Strategy

**Immediately assign @copilot when:**
- AI-Readiness Score ≥ 9
- Issue has clear acceptance criteria
- All context is provided
- No external dependencies

**For lower scores (5-8):**
- Route to "AI Agent Potential" board
- Don't assign yet - needs clarification first
- Suggest specific questions to improve readiness

**For scores < 5:**
- Route to "Human Review Required"
- Flag for human expertise
- No AI agent assignment

## Recommended AI Agent Types

Based on task characteristics, suggest:

- **@copilot** - General code changes, GitHub-integrated work (use for immediate assignment)
- **Codex** - Complex code generation, algorithm implementation
- **Claude** - Analysis, refactoring, documentation with context
- **Custom agents** - Specialized workflows (testing, security scanning)

## Analysis Template

For each issue, provide:

1. **AI-Readiness Assessment** (1-2 sentences)
   - What makes this suitable/unsuitable for AI agents?
   
2. **Field Rationale** (bullet points)
   - AI-Readiness Score: [score + brief reason]
   - Status: [status + brief reason]
   - Effort: [estimate + brief reason]
   - AI Agent Type: [type + brief reason]
   - Priority: [priority + brief reason]

3. **Assignment Decision**
   - If score ≥ 9: "Assigning to @copilot for immediate work"
   - If score 5-8: "Needs [specific clarifications] before assignment"
   - If score < 5: "Requires human review - [specific reasons]"

## Important Notes

- Projects are created automatically if they don't exist
- Focus on AI agent suitability over traditional triage criteria
- Prioritize issues with clear, testable outcomes
- Flag issues that need human clarification
- Consider batch-processing opportunities for similar issues

## Workflow Steps

1. **Fetch Issues**: Use GitHub MCP to query up to ${{ github.event.inputs.max_issues }} most recent open issues (default: 10)
2. **Score Each Issue**: Evaluate AI-readiness based on the criteria above
3. **Route to Project Board**: For each issue, output an `update_project` safe-output item with `"project": "${{ github.event.inputs.project_url }}"` to add it to the project board with field assignments

## Execution Notes

- This workflow runs every 4 hours automatically (or manually with custom parameters)
- Input defaults: max_issues=10, project_url=https://github.com/users/mnkiefer/projects/24
- All issues are routed to the project board with differentiation via Status field
- Custom fields are created automatically if they don't exist
- User projects must exist before workflow runs (cannot auto-create)
