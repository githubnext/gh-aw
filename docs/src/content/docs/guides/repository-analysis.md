---
title: Repository Analysis
description: Analyze external repositories to identify opportunities for agentic workflows
---

The Repository Analyzer is an agentic workflow that audits external GitHub repositories to identify opportunities for automation using agentic workflows.

## Overview

The repo-analyzer workflow comprehensively analyzes a target repository by:

1. **Surveying GitHub Actions workflows** - Identifying complexity, health issues, and automation gaps
2. **Analyzing source code patterns** - Finding maintenance needs and quality indicators
3. **Reviewing issue history** - Detecting recurring patterns and maintenance burden
4. **Discovering opportunities** - Recommending specific agentic workflows with effort/impact estimates
5. **Caching results** - Storing analysis for future comparison and trend tracking

## Running the Analyzer

### Prerequisites

1. Initialize your repository for agentic workflows:
   ```bash
   gh aw init
   ```

2. Add the repo-analyzer workflow:
   ```bash
   gh aw add githubnext/gh-aw .github/workflows/repo-analyzer.md
   ```

3. Compile the workflow:
   ```bash
   gh aw compile
   ```

### Analyzing a Repository

Run the analyzer on any public GitHub repository:

```bash
# Analyze the FStar repository
gh aw run repo-analyzer -F repo=FStarLang/FStar

# Analyze any other repository
gh aw run repo-analyzer -F repo=owner/repo-name
```

The workflow will:
- Use GitHub MCP tools to gather comprehensive repository data
- Analyze workflows, code patterns, and issue history
- Generate a detailed report as a GitHub Discussion
- Cache results in the repo-memory tool for future comparison

### Viewing Results

After the workflow completes:

1. Check the workflow run status:
   ```bash
   gh aw logs repo-analyzer
   ```

2. Find the generated discussion in your repository's Discussions tab under the "analysis" category

3. Review cached results:
   ```bash
   # View analysis cache
   ls -la /tmp/gh-aw/cache-memory/repo-analyses/
   ```

## Analysis Report Structure

The generated report includes:

### Executive Summary
- Repository overview (description, language, activity)
- Key findings and statistics
- High-level recommendations

### Current State Analysis

1. **GitHub Actions Workflows**
   - Existing workflows inventory
   - Health status and success rates
   - Identified automation gaps

2. **Source Code Patterns**
   - Code organization structure
   - Maintenance indicators (TODO comments, deprecated code)
   - Quality observations

3. **Issue History**
   - Issue statistics and lifecycle metrics
   - Recurring issue patterns
   - Top issue categories

### Identified Opportunities

For each opportunity, the report provides:
- Problem description
- Proposed agentic workflow solution
- Impact and effort estimates
- Time savings projections
- Example workflow template
- Expected benefits

### Implementation Roadmap

- **Phase 1: Quick Wins** (Weeks 1-2)
- **Phase 2: Core Automation** (Weeks 3-6)
- **Phase 3: Advanced Features** (Weeks 7-12)

### Example Workflow Templates

Complete, ready-to-use workflow templates for the top opportunities, including:
- Frontmatter configuration
- Permissions and tools
- Safe outputs
- Detailed instructions

## Example: Analyzing FStar

To analyze the [FStar](https://github.com/FStarLang/FStar) repository:

```bash
gh aw run repo-analyzer -F repo=FStarLang/FStar
```

This will analyze:
- FStar's GitHub Actions workflows (CI, testing, release automation)
- Source code patterns in the OCaml/F* codebase
- Issue history (bug reports, feature requests, maintenance issues)
- Opportunities for automation (test triage, release notes, documentation updates)

Expected findings might include opportunities for:
- Automated CI failure analysis
- Test result summarization
- Documentation generation
- Dependency update reviews
- Issue triage and labeling

## Caching and Historical Analysis

The workflow uses the `cache-memory` tool to store analysis results:

```
/tmp/gh-aw/cache-memory/
├── repo-analyses/
│   ├── index.json                    # Master index
│   └── owner-repo/
│       ├── latest.json               # Most recent analysis
│       └── YYYY-MM-DD.json          # Historical analyses
├── patterns/
│   ├── workflows.json                # Common patterns
│   ├── issues.json
│   └── code.json
└── opportunities/
    ├── by-impact.json
    ├── by-effort.json
    └── implemented.json
```

On subsequent runs, the workflow compares new findings with historical data to identify:
- New opportunities
- Resolved issues
- Trends over time
- Impact of implemented workflows

## Customizing the Analysis

You can modify the repo-analyzer workflow to:

1. **Focus on specific areas:**
   ```markdown
   ## Analysis Focus
   
   For this run, focus specifically on:
   - CI/CD pipeline optimization
   - Test automation opportunities
   - Release management
   ```

2. **Adjust analysis depth:**
   ```markdown
   ## Analysis Configuration
   
   - Maximum issues to analyze: 100
   - Workflow run history: last 30 days
   - Code search depth: top-level modules only
   ```

3. **Change report format:**
   ```markdown
   ## Report Format
   
   Create a compact executive summary focusing on:
   - Top 3 opportunities only
   - High-level effort estimates
   - Implementation timeline
   ```

## Integration with Other Tools

The repo-analyzer workflow integrates with:

- **GitHub MCP Server** - For comprehensive GitHub API access
- **Cache Memory** - For persistent analysis storage
- **Discussions** - For sharing reports with stakeholders
- **Upload Assets** - For charts and visualizations

## Best Practices

1. **Regular Analysis**: Run the analyzer periodically (monthly/quarterly) to track improvement trends

2. **Team Review**: Share the generated discussion with your team to prioritize opportunities

3. **Start Small**: Implement 2-3 high-impact, low-effort workflows first

4. **Measure Impact**: Track metrics before and after implementing workflows

5. **Iterate**: Use cached historical data to refine recommendations

## Troubleshooting

### GitHub API Rate Limits

If you hit rate limits:
- The workflow respects rate limits and will complete what it can
- Run again later to continue analysis
- Consider using a GitHub App token for higher rate limits

### Long Running Analysis

For large repositories:
- The workflow has a 60-minute timeout
- It will prioritize high-value analysis first
- Incomplete analysis will be noted in the report

### Missing Data

If some data is unavailable:
- The workflow gracefully handles missing information
- It will note gaps in the report
- Public repositories work best for comprehensive analysis

## Example Output

Here's a sample finding from analyzing a repository:

```markdown
### High-Priority Opportunity: Automated CI Failure Analysis

**Problem**: Developers spend 2-3 hours per week triaging CI failures

**Solution**: Agentic workflow that:
- Monitors CI failures daily
- Analyzes logs to identify root causes
- Groups similar failures
- Creates issues with diagnostic information

**Impact**: 🟢 High | **Effort**: 🟡 Medium

**Estimated Time Savings**: 8-12 hours/month

**Example Workflow**:
[Complete workflow template included in report]

**Benefits**:
- Faster failure triage (2 hours → 15 minutes)
- Better failure categorization
- Automated issue creation
- Historical failure tracking
```

## Next Steps

After receiving your analysis report:

1. Review the prioritized opportunities with your team
2. Select 2-3 workflows to implement first
3. Use the provided templates as starting points
4. Monitor the impact and iterate
5. Run the analyzer again to track progress

## Related Resources

- [Quick Start Guide](/setup/quick-start/)
- [GitHub MCP Server](/tools/github-mcp/)
- [Cache Memory Tool](/tools/cache-memory/)
- [Security Guidelines](/guides/security/)
