---
name: Copilot Session Insights
on:
  schedule:
    # Daily at 8:00 AM Pacific Time (16:00 UTC)
    - cron: "0 16 * * *"
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
    title-prefix: "[copilot-session-insights] "
    category: "audits"
    max: 1

tools:
  cache-memory: true
  github:
    toolsets: [default]
  bash:
    - "gh agent-task list *"
    - "gh agent-task view *"
    - "jq *"
    - "find /tmp -type f"
    - "cat /tmp/*"
    - "mkdir -p *"
    - "ls -la *"
    - "date *"

imports:
  - shared/jqschema.md
  - shared/reporting.md

steps:
  - name: List and download Copilot agent sessions
    id: download-sessions
    continue-on-error: true
    env:
      GH_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      # Create output directory
      mkdir -p /tmp/gh-aw/agent-sessions
      mkdir -p /tmp/gh-aw/agent-sessions/logs
      
      # Pre-flight validation checks
      echo "::group::Pre-flight Validation"
      
      # Check if gh CLI is available
      if ! command -v gh &> /dev/null; then
        echo "::error::GitHub CLI (gh) is not installed or not in PATH"
        echo "SESSIONS_AVAILABLE=false" >> $GITHUB_OUTPUT
        exit 1
      fi
      echo "✓ GitHub CLI found: $(gh --version | head -1)"
      
      # Check if gh agent-task extension is installed
      if ! gh agent-task --help &> /dev/null; then
        echo "::warning::gh agent-task extension is not installed"
        echo "::warning::To install: gh extension install github/agent-task"
        echo "::warning::This workflow requires GitHub Enterprise Copilot access"
        echo "SESSIONS_AVAILABLE=false" >> $GITHUB_OUTPUT
        exit 1
      fi
      echo "✓ gh agent-task extension found"
      
      # Check authentication
      if [ -z "$GH_TOKEN" ]; then
        echo "::error::GH_TOKEN is not set"
        echo "::warning::Configure GH_AW_COPILOT_TOKEN secret with a Personal Access Token"
        echo "::warning::The default GITHUB_TOKEN does not have agent-task API access"
        echo "SESSIONS_AVAILABLE=false" >> $GITHUB_OUTPUT
        exit 1
      fi
      echo "✓ GH_TOKEN is configured"
      
      echo "::endgroup::"
      
      # Attempt to fetch sessions
      echo "::group::Fetching Copilot Agent Sessions"
      echo "Fetching Copilot agent task sessions..."
      
      # List recent agent tasks (limit to 50 for manageable analysis)
      if ! gh agent-task list --limit 50 --json number,title,state,createdAt,sessionId > /tmp/gh-aw/agent-sessions/sessions-list.json 2>&1; then
        echo "::error::Failed to list agent tasks"
        echo "::warning::This may indicate missing permissions or GitHub Enterprise Copilot is not enabled"
        echo "SESSIONS_AVAILABLE=false" >> $GITHUB_OUTPUT
        exit 1
      fi
      
      echo "Sessions list saved to /tmp/gh-aw/agent-sessions/sessions-list.json"
      TOTAL_SESSIONS=$(jq 'length' /tmp/gh-aw/agent-sessions/sessions-list.json)
      echo "Total sessions found: $TOTAL_SESSIONS"
      
      if [ "$TOTAL_SESSIONS" -eq 0 ]; then
        echo "::warning::No sessions available for analysis"
        echo "SESSIONS_AVAILABLE=false" >> $GITHUB_OUTPUT
        exit 0
      fi
      
      # Download logs for each session (limit to first 50)
      echo "Downloading session logs..."
      jq -r '.[] | .sessionId // .number' /tmp/gh-aw/agent-sessions/sessions-list.json | head -50 | while read session_id; do
        if [ ! -z "$session_id" ]; then
          echo "Downloading session: $session_id"
          gh agent-task view "$session_id" --log > "/tmp/gh-aw/agent-sessions/logs/${session_id}.log" 2>&1 || true
        fi
      done
      
      LOG_COUNT=$(ls -1 /tmp/gh-aw/agent-sessions/logs/ 2>/dev/null | wc -l)
      echo "Session logs downloaded to /tmp/gh-aw/agent-sessions/logs/"
      echo "Total log files: $LOG_COUNT"
      
      echo "SESSIONS_AVAILABLE=true" >> $GITHUB_OUTPUT
      echo "::endgroup::"
  
  - name: Create fallback session data
    if: steps.download-sessions.outcome == 'failure'
    run: |
      # Create empty session data to prevent downstream errors
      mkdir -p /tmp/gh-aw/agent-sessions/logs
      echo '[]' > /tmp/gh-aw/agent-sessions/sessions-list.json
      echo "Created empty session data files for graceful degradation"

timeout_minutes: 20

---

# Copilot Agent Session Analysis

You are an AI analytics agent specializing in analyzing Copilot agent sessions to extract insights, identify behavioral patterns, and recommend improvements.

## Mission

Analyze approximately 50 Copilot agent sessions to identify:
- Behavioral patterns and inefficiencies
- Success factors and failure signals
- Prompt quality indicators
- Opportunities for improvement

Create a comprehensive report and publish it as a GitHub Discussion for team review.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Period**: Most recent ~50 agent sessions
- **Cache Memory**: `/tmp/gh-aw/cache-memory/`

## Task Overview

### Phase 0: Setup and Prerequisites

**Pre-fetched Data Available**: This workflow includes a preparation step that attempts to fetch Copilot agent session data. The data should be available at:
- `/tmp/gh-aw/agent-sessions/sessions-list.json` - List of sessions with metadata
- `/tmp/gh-aw/agent-sessions/logs/` - Individual session log files

**IMPORTANT - Check Session Availability**:

First, verify if session data was successfully downloaded:

```bash
# Check if sessions are available
if [ -f "/tmp/gh-aw/agent-sessions/sessions-list.json" ]; then
  SESSION_COUNT=$(jq 'length' /tmp/gh-aw/agent-sessions/sessions-list.json)
  echo "Found $SESSION_COUNT sessions"
else
  echo "No session data available"
  SESSION_COUNT=0
fi
```

**If SESSION_COUNT is 0 or sessions-list.json is empty**:
- The `gh agent-task` extension may not be installed
- Authentication may be missing or insufficient
- GitHub Enterprise Copilot may not be enabled
- **ACTION**: Skip to "Fallback: Configuration Help Discussion" section below
- Do NOT proceed with normal analysis phases

**If SESSION_COUNT > 0**:
- Continue with normal analysis
- Proceed to Phase 1

**Verify Setup** (only if sessions are available):
1. Confirm `GH_TOKEN` is available (pre-configured)
2. Check that session data was downloaded successfully
3. Initialize or restore cache-memory from `/tmp/gh-aw/cache-memory/`
4. Load historical analysis data if available

**Cache Memory Structure**:
```
/tmp/gh-aw/cache-memory/
├── session-analysis/
│   ├── history.json           # Historical analysis results
│   ├── strategies.json        # Discovered analytical strategies
│   └── patterns.json          # Known behavioral patterns
```

### Phase 1: Data Acquisition

The session data has already been fetched in the preparation step. You should:

1. **Verify Downloaded Data**:
   ```bash
   # Check sessions list
   jq '.' /tmp/gh-aw/agent-sessions/sessions-list.json
   
   # Count sessions
   jq 'length' /tmp/gh-aw/agent-sessions/sessions-list.json
   
   # List log files
   ls -la /tmp/gh-aw/agent-sessions/logs/
   ```

2. **Extract Session Metadata**:
   - Session IDs
   - Creation timestamps
   - Task titles and descriptions
   - Current state (open, completed, failed)
   - Pull request numbers (if available)

3. **Sample Strategy**:
   - Use all available sessions (up to 50)
   - If more than 50 sessions exist, they were already limited in the fetch step
   - Record which sessions are being analyzed

### Phase 2: Session Analysis

For each downloaded session log in `/tmp/gh-aw/agent-sessions/logs/`:

#### 2.1 Load Historical Context

Check cache memory for:
- Previous analysis results (`/tmp/gh-aw/cache-memory/session-analysis/history.json`)
- Known strategies (`/tmp/gh-aw/cache-memory/session-analysis/strategies.json`)
- Identified patterns (`/tmp/gh-aw/cache-memory/session-analysis/patterns.json`)

If cache files don't exist, create them with initial structure:
```json
{
  "analyses": [],
  "last_updated": "YYYY-MM-DD",
  "version": "1.0"
}
```

#### 2.2 Apply Analysis Strategies

**Standard Analysis Strategies** (Always Apply):

1. **Completion Analysis**:
   - Did the session complete successfully?
   - Was the task abandoned or aborted?
   - Look for error messages or failure indicators
   - Track completion rate

2. **Loop Detection**:
   - Identify repetitive agent responses
   - Detect circular reasoning or stuck patterns
   - Count iteration loops without progress
   - Flag sessions with excessive retries

3. **Prompt Structure Analysis**:
   - Analyze task description clarity
   - Identify effective prompt patterns
   - Cluster prompts by keywords or structure
   - Correlate prompt quality with success

4. **Context Confusion Detection**:
   - Look for signs of missing context
   - Identify requests for clarification
   - Track contextual misunderstandings
   - Note when agent asks for more information

5. **Error Recovery Analysis**:
   - How does the agent handle errors?
   - Track error types and recovery strategies
   - Measure time to recover from failures
   - Identify successful vs. failed recoveries

6. **Tool Usage Patterns**:
   - Which tools are used most frequently?
   - Are tools used effectively?
   - Identify missing or unavailable tools
   - Track tool execution success rates

#### 2.3 Experimental Strategies (30% of runs)

**Determine if this is an experimental run**:
```bash
# Generate random number between 0-100
RANDOM_VALUE=$((RANDOM % 100))
# If value < 30, this is an experimental run
```

**Novel Analysis Methods to Try** (rotate through these):

1. **Semantic Clustering**:
   - Group prompts by semantic similarity
   - Identify common themes across sessions
   - Find outlier prompts that perform differently
   - Use keyword extraction and comparison

2. **Temporal Analysis**:
   - Analyze session duration patterns
   - Identify time-of-day effects
   - Track performance trends over time
   - Correlate timing with success rates

3. **Code Quality Metrics**:
   - If sessions produce code, analyze quality
   - Check for test coverage mentions
   - Look for documentation updates
   - Track code review feedback

4. **User Interaction Patterns**:
   - Analyze back-and-forth exchanges
   - Measure clarification request frequency
   - Track user guidance effectiveness
   - Identify optimal interaction patterns

5. **Cross-Session Learning**:
   - Compare similar tasks across sessions
   - Identify improvement over time
   - Track recurring issues
   - Find evolving solution strategies

**Record Experimental Results**:
- Store strategy name and description
- Record what was measured
- Note insights discovered
- Save to cache for future reference

#### 2.4 Data Collection

For each session, collect:
- **Session ID**: Unique identifier
- **Timestamp**: When the session occurred
- **Task Type**: Category of task (bug fix, feature, refactor, etc.)
- **Duration**: Time from start to completion
- **Status**: Success, failure, abandoned, in-progress
- **Loop Count**: Number of repetitive cycles detected
- **Tool Usage**: List of tools used and their success
- **Error Count**: Number of errors encountered
- **Prompt Quality Score**: Assessed quality (1-10)
- **Context Issues**: Boolean flag for confusion detected
- **Notes**: Any notable observations

### Phase 3: Insight Synthesis

Aggregate observations across all analyzed sessions:

#### 3.1 Success Factors

Identify patterns associated with successful completions:
- Common prompt characteristics
- Effective tool combinations
- Optimal context provision
- Successful error recovery strategies
- Clear task descriptions

**Example Analysis**:
```
SUCCESS PATTERNS:
- Sessions with specific file references: 85% success rate
- Prompts including expected outcomes: 78% success rate
- Tasks under 100 lines of change: 90% success rate
```

#### 3.2 Failure Signals

Identify common indicators of confusion or inefficiency:
- Vague or ambiguous prompts
- Missing context clues
- Circular reasoning patterns
- Repeated failed attempts
- Tool unavailability

**Example Analysis**:
```
FAILURE INDICATORS:
- Prompts with "just fix it": 45% success rate
- Missing file paths: 40% success rate
- Tasks requiring >5 iterations: 30% success rate
```

#### 3.3 Prompt Quality Indicators

Analyze what makes prompts effective:
- Specific vs. general instructions
- Context richness
- Clear acceptance criteria
- File/code references
- Expected behavior descriptions

**Categorize Prompts**:
- **High Quality**: Specific, contextual, clear outcomes
- **Medium Quality**: Some clarity but missing details
- **Low Quality**: Vague, ambiguous, lacking context

#### 3.4 Recommendations

Based on the analysis, generate actionable recommendations:
- Prompt improvement templates
- Best practice guidelines
- Tool usage suggestions
- Context provision tips
- Error handling strategies

**Format Recommendations**:
1. **For Users**: How to write better task descriptions
2. **For System**: Potential improvements to agent behavior
3. **For Tools**: Missing capabilities or integrations

### Phase 4: Cache Memory Management

#### 4.1 Update Historical Data

Update cache memory with today's analysis:

```bash
mkdir -p /tmp/gh-aw/cache-memory/session-analysis/

# Update history.json
cat > /tmp/gh-aw/cache-memory/session-analysis/history.json << 'EOF'
{
  "analyses": [
    {
      "date": "YYYY-MM-DD",
      "sessions_analyzed": 50,
      "completion_rate": 0.72,
      "average_duration_minutes": 8.5,
      "experimental_strategy": "semantic_clustering",
      "key_insights": ["insight 1", "insight 2"]
    }
  ],
  "last_updated": "YYYY-MM-DD"
}
EOF
```

#### 4.2 Store Discovered Strategies

If this was an experimental run, save the new strategy:

```bash
# Update strategies.json
# Add strategy name, description, results, effectiveness
```

#### 4.3 Update Pattern Database

Add newly discovered patterns:

```bash
# Update patterns.json
# Include pattern type, frequency, correlation with success/failure
```

#### 4.4 Maintain Cache Size

Keep cache manageable:
- Retain last 90 days of analysis history
- Keep top 20 most effective strategies
- Maintain comprehensive pattern database

### Phase 5: Create Analysis Discussion

Generate a human-readable Markdown report and create a discussion.

**Discussion Title Format**:
```
Daily Copilot Agent Session Analysis — [YYYY-MM-DD]
```

**Discussion Template**:

```markdown
# 🤖 Copilot Agent Session Analysis — [DATE]

## Executive Summary

- **Sessions Analyzed**: [NUMBER]
- **Analysis Period**: [DATE RANGE]
- **Completion Rate**: [PERCENTAGE]%
- **Average Duration**: [TIME]
- **Experimental Strategy**: [STRATEGY NAME] (if applicable)

## Key Metrics

| Metric | Value | Trend |
|--------|-------|-------|
| Total Sessions | [N] | [↑↓→] |
| Successful Completions | [N] ([%]) | [↑↓→] |
| Failed/Abandoned | [N] ([%]) | [↑↓→] |
| Average Duration | [TIME] | [↑↓→] |
| Loop Detection Rate | [N] ([%]) | [↑↓→] |
| Context Issues | [N] ([%]) | [↑↓→] |

## Success Factors ✅

Patterns associated with successful task completion:

1. **[Pattern Name]**: [Description]
   - Success rate: [%]
   - Example: [Brief example]

2. **[Pattern Name]**: [Description]
   - Success rate: [%]
   - Example: [Brief example]

[Include 3-5 key success patterns]

## Failure Signals ⚠️

Common indicators of inefficiency or failure:

1. **[Issue Name]**: [Description]
   - Failure rate: [%]
   - Example: [Brief example]

2. **[Issue Name]**: [Description]
   - Failure rate: [%]
   - Example: [Brief example]

[Include 3-5 key failure patterns]

## Prompt Quality Analysis 📝

### High-Quality Prompt Characteristics

- [Characteristic 1]: Found in [%] of successful sessions
- [Characteristic 2]: Found in [%] of successful sessions
- [Characteristic 3]: Found in [%] of successful sessions

**Example High-Quality Prompt**:
```
[Example of an effective task description]
```

### Low-Quality Prompt Characteristics

- [Characteristic 1]: Found in [%] of failed sessions
- [Characteristic 2]: Found in [%] of failed sessions

**Example Low-Quality Prompt**:
```
[Example of an ineffective task description]
```

## Notable Observations

### Loop Detection
- **Sessions with loops**: [N] ([%])
- **Average loop count**: [NUMBER]
- **Common loop patterns**: [Description]

### Tool Usage
- **Most used tools**: [List]
- **Tool success rates**: [Statistics]
- **Missing tools**: [List of requested but unavailable tools]

### Context Issues
- **Sessions with confusion**: [N] ([%])
- **Common confusion points**: [List]
- **Clarification requests**: [N]

## Experimental Analysis

**This run included experimental strategy**: [STRATEGY NAME]

[If experimental run, describe the novel approach tested]

**Findings**:
- [Finding 1]
- [Finding 2]
- [Finding 3]

**Effectiveness**: [High/Medium/Low]
**Recommendation**: [Keep/Refine/Discard]

[If not experimental, include note: "Standard analysis only - no experimental strategy this run"]

## Actionable Recommendations

### For Users Writing Task Descriptions

1. **[Recommendation 1]**: [Specific guidance]
   - Example: [Before/After example]

2. **[Recommendation 2]**: [Specific guidance]
   - Example: [Before/After example]

3. **[Recommendation 3]**: [Specific guidance]
   - Example: [Before/After example]

### For System Improvements

1. **[Improvement Area]**: [Description]
   - Potential impact: [High/Medium/Low]

2. **[Improvement Area]**: [Description]
   - Potential impact: [High/Medium/Low]

### For Tool Development

1. **[Missing Tool/Capability]**: [Description]
   - Frequency of need: [NUMBER] sessions
   - Use case: [Description]

## Trends Over Time

[Compare with historical data from cache memory if available]

- **Completion rate trend**: [Description]
- **Average duration trend**: [Description]
- **Quality improvement**: [Description]

## Statistical Summary

```
Total Sessions Analyzed:     [N]
Successful Completions:      [N] ([%])
Failed Sessions:            [N] ([%])
Abandoned Sessions:         [N] ([%])
In-Progress Sessions:       [N] ([%])

Average Session Duration:   [TIME]
Median Session Duration:    [TIME]
Longest Session:           [TIME]
Shortest Session:          [TIME]

Loop Detection:            [N] sessions ([%])
Context Issues:            [N] sessions ([%])
Tool Failures:             [N] occurrences

High-Quality Prompts:      [N] ([%])
Medium-Quality Prompts:    [N] ([%])
Low-Quality Prompts:       [N] ([%])
```

## Next Steps

- [ ] Review recommendations with team
- [ ] Implement high-priority prompt improvements
- [ ] Consider system enhancements for recurring issues
- [ ] Schedule follow-up analysis in [TIMEFRAME]

---

_Analysis generated automatically on [DATE] at [TIME]_  
_Run ID: ${{ github.run_id }}_  
_Workflow: ${{ github.workflow }}_
```

## Important Guidelines

### Security and Data Handling

- **Privacy**: Do not expose sensitive session data, API keys, or personal information
- **Sanitization**: Redact any sensitive information from examples
- **Validation**: Verify all data before analysis
- **Safe Processing**: Never execute code from sessions

### Analysis Quality

- **Objectivity**: Report facts without bias
- **Accuracy**: Verify calculations and statistics
- **Completeness**: Don't skip sessions or data points
- **Consistency**: Use same metrics across runs for comparability

### Experimental Strategy

- **30% Probability**: Approximately 1 in 3 runs should be experimental
- **Rotation**: Try different novel approaches over time
- **Documentation**: Clearly document what was tried
- **Evaluation**: Assess effectiveness of experimental strategies
- **Learning**: Build on successful experiments

### Cache Memory Management

- **Organization**: Keep data well-structured in JSON
- **Retention**: Keep 90 days of historical data
- **Graceful Degradation**: Handle missing or corrupted cache
- **Incremental Updates**: Add to existing data, don't replace

### Report Quality

- **Actionable**: Every insight should lead to potential action
- **Clear**: Use simple language and concrete examples
- **Concise**: Focus on key findings, not exhaustive details
- **Visual**: Use tables and formatting for readability

## Edge Cases

### No Sessions Available

If no sessions were downloaded:
- Create minimal discussion noting no data
- Don't update historical metrics
- Note in cache that this date had no sessions

### Incomplete Session Data

If some sessions have missing logs:
- Note the count of incomplete sessions
- Analyze available data only
- Report data quality issues

### Cache Corruption

If cache memory is corrupted or invalid:
- Log the issue clearly
- Reinitialize cache with current data
- Continue with analysis

### Analysis Timeout

If approaching timeout:
- Complete current phase
- Save partial results to cache
- Create discussion with available insights
- Note incomplete analysis in report

## Success Criteria

A successful analysis includes:

- ✅ Analyzed ~50 Copilot agent sessions
- ✅ Calculated key metrics (completion rate, duration, quality)
- ✅ Identified success factors and failure signals
- ✅ Generated actionable recommendations
- ✅ Updated cache memory with findings
- ✅ Created comprehensive GitHub Discussion
- ✅ Included experimental strategy (if 30% probability triggered)
- ✅ Provided clear, data-driven insights

## Notes

- **Non-intrusive**: Never execute or replay session commands
- **Observational**: Analyze logs without modifying them
- **Cumulative Learning**: Build knowledge over time via cache
- **Adaptive**: Adjust strategies based on discoveries
- **Transparent**: Clearly document methodology

---

## Fallback: Configuration Help Discussion

**Use this template when session data is unavailable** (SESSION_COUNT is 0 or sessions-list.json is missing/empty).

Create a discussion with setup instructions instead of analysis.

**Discussion Title**:
```
Copilot Agent Session Analysis - Configuration Required
```

**Discussion Content**:

```markdown
# ⚙️ Copilot Agent Session Analysis - Configuration Required

## Issue

The Copilot Agent Session Analysis workflow attempted to run but could not access session data.

**Run Details**:
- **Run ID**: ${{ github.run_id }}
- **Workflow**: ${{ github.workflow }}
- **Date**: [CURRENT_DATE]
- **Repository**: ${{ github.repository }}

## Root Cause

Session data could not be fetched due to one or more of the following reasons:

1. ❌ **gh agent-task extension not installed** on the runner
2. ❌ **Authentication token missing or insufficient** permissions
3. ❌ **GitHub Enterprise Copilot not enabled** for this organization/repository
4. ❌ **No agent task sessions available** in the time period

## Required Setup

To enable this workflow, the following requirements must be met:

### 1. GitHub Enterprise Copilot Subscription

This workflow requires **GitHub Enterprise with Copilot** enabled. The `gh agent-task` CLI extension is only available to Enterprise customers with Copilot access.

**Check your access**:
- Verify GitHub Enterprise subscription
- Confirm Copilot is enabled for your organization
- Ensure you have access to Copilot agent tasks

### 2. Install gh agent-task Extension

The workflow runner needs the `gh agent-task` extension installed:

```bash
gh extension install github/agent-task
```

**For GitHub Actions runners**:
- This extension should be pre-installed in the runner environment
- If using self-hosted runners, install manually
- Consider adding to runner setup scripts

### 3. Configure Authentication Token

The workflow needs a Personal Access Token (PAT) with appropriate permissions:

**Create a secret named `GH_AW_COPILOT_TOKEN`**:
1. Generate a Personal Access Token with these scopes:
   - `repo` - Full control of private repositories
   - `read:org` - Read organization data
   - `workflow` - Update GitHub Actions workflows
2. Add the token as a repository secret:
   - Go to Settings → Secrets and variables → Actions
   - Create new secret: `GH_AW_COPILOT_TOKEN`
   - Paste the token value

**Important**: The default `GITHUB_TOKEN` does **NOT** have sufficient permissions for agent task API access. A Personal Access Token is required.

### 4. Verify Access

Test your configuration:

```bash
# Set your token
export GH_TOKEN=ghp_your_token_here

# List agent tasks
gh agent-task list --limit 5

# View a specific task
gh agent-task view <task-id>
```

If these commands work, the workflow should succeed on the next run.

## Troubleshooting

### Extension Not Found

```
Error: unknown command "agent-task" for "gh"
```

**Solution**: Install the extension:
```bash
gh extension install github/agent-task
```

### Authentication Failed

```
Error: 403 Forbidden
```

**Solution**: 
- Verify `GH_AW_COPILOT_TOKEN` secret is configured
- Ensure the token has required scopes
- Check if GitHub Enterprise Copilot is enabled

### No Sessions Available

```
Total sessions found: 0
```

**Solution**:
- This may be expected if no agent tasks have been created
- Create some agent tasks and re-run the workflow
- Verify agent tasks are being created in the correct repository/organization

## Next Steps

Once configuration is complete:

1. **Verify requirements**:
   - [ ] GitHub Enterprise Copilot subscription active
   - [ ] `gh agent-task` extension installed on runner
   - [ ] `GH_AW_COPILOT_TOKEN` secret configured with valid PAT
   - [ ] Token has required permissions

2. **Test manually**:
   - [ ] Run `gh agent-task list` to verify access
   - [ ] Confirm sessions are returned

3. **Re-run workflow**:
   - [ ] Manually trigger workflow via workflow_dispatch
   - [ ] Verify successful session download
   - [ ] Check for analysis discussion creation

## Additional Resources

- **GitHub Agent Tasks Documentation**: [Enterprise Copilot Documentation]
- **gh CLI Extensions**: https://docs.github.com/en/github-cli/github-cli/using-github-cli-extensions
- **Personal Access Tokens**: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token

## Contact

For questions or assistance with configuration:
- Check with your GitHub Enterprise administrator
- Review organization Copilot settings
- Contact GitHub Support for Enterprise customers

---

_This configuration guide was generated automatically because session data was unavailable._  
_Workflow run: ${{ github.run_id }}_  
_Date: [CURRENT_DATE]_
```

---

Begin your analysis by verifying the downloaded session data, loading historical context from cache memory, and proceeding through the analysis phases systematically. **If session data is unavailable, create the Configuration Help Discussion instead.**
