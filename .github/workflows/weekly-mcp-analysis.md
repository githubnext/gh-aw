---
on:
  schedule:
    - cron: "0 14 * * 1"  # Weekly on Mondays at 2 PM UTC (9 AM EST)
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  issues: read
engine: copilot
timeout-minutes: 30
network:
  allowed:
    - defaults
    - github
  firewall: true
tools:
  edit:
  bash:
    - "*"
  web-fetch:
  github:
    toolsets:
      - repos
      - issues
      - discussions
safe-outputs:
  upload-assets:
  create-discussion:
    title-prefix: "🎯 "
    category: "General"
imports:
  - shared/reporting.md
  - shared/trends.md
---

# Weekly Master Control Program Analysis

You are analyzing the **master-control-program** repository (https://github.com/gokhanarkan/master-control-program) - an intelligent MCP meta-server that discovers and analyzes tools from 1000+ Model Context Protocol servers.

## Repository Context

The master-control-program is a Go-based orchestration system that:
- Syncs with the GitHub MCP Registry to discover MCP servers
- Extracts tools and schemas from server documentation
- Uses AI (Google Gemini) to semantically categorize and analyze servers
- Stores data in a Neo4j graph database
- Provides intelligent tool selection based on natural language queries
- Returns installation URLs for IDE integration

## Your Mission

Create a comprehensive weekly analysis discussion that provides insights into the repository's development, algorithms, and ecosystem trends.

## 📊 Required Trend Charts

**IMPORTANT**: Generate exactly 2 high-quality visualization charts.

### Chart Generation Process

**Phase 1: Data Collection**

Use GitHub API and web-fetch to gather:

1. **Repository Activity Data** (last 30 days):
   - Commits per day on main branch
   - Pull requests opened/merged/closed
   - Issues opened/closed
   - Contributors active per week

2. **Code Evolution Data** (if accessible via commits):
   - Lines of code changes over time
   - File change frequency in key directories (internal/, pkg/)
   - Language distribution changes

**Phase 2: Data Preparation**

Create CSV files in `/tmp/gh-aw/python/data/`:
- `repo_activity.csv` - Daily commit counts, PR activity, contributor counts
- `code_evolution.csv` - Code changes, file modifications

Each CSV needs date column and metric columns with headers.

**Phase 3: Chart Generation**

Generate **2 high-quality trend charts**:

**Chart 1: Development Activity Pulse**
- Multi-line chart showing:
  - Daily commit count (smoothed with 7-day moving average)
  - PRs merged per week (bar overlay)
  - Active contributors per week (line with markers)
- X-axis: Date (last 30 days)
- Y-axis: Count
- Title: "Master Control Program - Development Activity (Last 30 Days)"
- Save as: `/tmp/gh-aw/python/charts/mcp_activity_pulse.png`

**Chart 2: Code Evolution & Contribution Patterns**
- Stacked area or dual-axis chart showing:
  - Cumulative commits over time
  - Number of files changed per week (bar)
  - OR: Language breakdown if data available (Go, Python, etc.)
- X-axis: Date or Week (last 30 days)
- Y-axis: Count/Percentage
- Title: "Code Evolution & Contribution Patterns"
- Save as: `/tmp/gh-aw/python/charts/mcp_code_evolution.png`

**Chart Quality Requirements**:
- DPI: 300 minimum
- Figure size: 12x7 inches
- Use seaborn with 'darkgrid' or 'whitegrid' style
- Professional color palette (e.g., 'deep', 'muted', or custom)
- Grid lines for readability
- Large, clear labels and legend
- Annotations for significant events/peaks
- Use `plt.tight_layout()` before saving

**Phase 4: Upload & Embed**

1. Upload both charts using `upload asset` tool
2. Embed in discussion with this structure:

```markdown
## 📈 Development Pulse Visualized

### Activity Trends
![Development Activity](URL_FROM_CHART_1)

[2-3 sentence analysis of development velocity, contributor engagement, merge patterns]

### Code Evolution
![Code Evolution](URL_FROM_CHART_2)

[2-3 sentence analysis of codebase growth, refactoring patterns, or language usage]
```

### Python Implementation Notes

- Use pandas for data handling
- Use matplotlib.pyplot + seaborn for visualization
- Handle sparse data gracefully
- Apply date formatters: `plt.xticks(rotation=45)`
- Use `plt.figure(figsize=(12, 7), dpi=300)`

---

## Analysis Sections

Structure your discussion with these sections:

### 🎯 Executive Summary
Brief overview of the week's highlights - key commits, PRs, issues, or architectural changes.

### 🔍 Algorithm & Architecture Insights
Analyze recent changes to:
- Tool selection algorithms (AI prompt engineering, semantic matching)
- Graph database schema and queries
- Sync and discovery mechanisms
- Performance optimizations

Look for commits in:
- `internal/ai/` - AI selection logic
- `internal/graph/` - Neo4j operations
- `internal/discovery/` - Registry sync & tool extraction
- `internal/protocol/` - MCP protocol handling

### 📦 Registry & Ecosystem Trends
If there's discussion of:
- New MCP servers discovered
- Tool categorization improvements
- Server quality metrics
- Popular tool patterns

### 🚀 Performance & Technical Improvements
Highlight commits related to:
- Caching strategies
- Database optimizations
- Sync interval tuning
- Concurrency improvements
- Error handling enhancements

### 💡 Notable Code Changes
Summarize interesting technical decisions or refactorings in the codebase.

### 📋 Issue & PR Activity
Quick summary of:
- Issues opened/closed this week
- PRs merged and their impact
- Ongoing discussions or feature requests

### 🔮 Looking Ahead
Based on open issues, PRs, or comments - what might be coming next?

## Technical Requirements

1. **Fetch repository data** using GitHub API:
   - Commits to main branch (last 7 days)
   - Pull requests (opened, merged, closed)
   - Issues (opened, closed, comments)
   - File changes in key directories

2. **Analyze commit messages** for patterns:
   - Focus on commits touching algorithm files
   - Look for performance-related changes
   - Identify refactoring efforts

3. **Create discussion** using `create-discussion` safe output:
   ```
   TITLE: MCP Weekly Analysis - [Key highlight or theme]
   
   BODY: Your comprehensive analysis with charts embedded
   ```

4. **If minimal activity**: Still create a discussion noting the quiet week and highlighting the project's current state.

## Style Guidelines

- Technical but accessible
- Focus on architectural insights, not just activity metrics
- Connect code changes to their purpose
- Use emoji sparingly for section headers
- Include code snippets if relevant (algorithm changes, interesting patterns)
- Link to specific commits, PRs, issues where relevant

Remember: This is an analysis of an AI-powered tool discovery system. Emphasize algorithmic improvements, semantic understanding enhancements, and ecosystem growth trends. 🎯
