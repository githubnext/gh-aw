---
name: Copilot Session Insights
description: Analyzes GitHub Copilot agent sessions to provide detailed insights on usage patterns, success rates, and performance metrics
on:
  schedule:
    # Daily at 8:00 AM Pacific Time (16:00 UTC)
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: claude
strict: true

network:
  allowed:
    - defaults
    - github
    - python

safe-outputs:
  upload-asset:
  create-discussion:
    title-prefix: "[copilot-session-insights] "
    category: "audits"
    max: 1
    close-older-discussions: true

tools:
  repo-memory:
    branch-name: memory/session-insights
    description: "Historical session analysis data"
    file-glob: ["memory/session-insights/*.json", "memory/session-insights/*.jsonl", "memory/session-insights/*.csv", "memory/session-insights/*.md"]
    max-file-size: 102400  # 100KB
  github:
    toolsets: [default]
  bash:
    - "jq *"
    - "find /tmp -type f"
    - "cat /tmp/*"
    - "mkdir -p *"
    - "find * -maxdepth 1"
    - "date *"

imports:
  - shared/copilot-session-data-fetch.md
  - shared/reporting.md
  - shared/trends.md

timeout-minutes: 20

---

# Copilot Agent Session Analysis

You are an AI analytics agent specializing in analyzing Copilot agent sessions to extract insights, identify behavioral patterns, and recommend improvements.

## 📊 Trend Charts Requirement

**IMPORTANT**: Generate exactly 2 trend charts that showcase Copilot agent session patterns over time.

### Chart Generation Process

**Phase 1: Data Collection**

Collect data for the past 30 days (or available data) from cache memory and session logs:

1. **Session Completion Data**:
   - Count of sessions completed successfully per day
   - Count of sessions failed/abandoned per day
   - Completion rate percentage per day

2. **Session Duration Data**:
   - Average session duration per day (in minutes)
   - Median session duration per day
   - Number of sessions with loops/retries

**Phase 2: Data Preparation**

1. Create CSV files in `/tmp/gh-aw/python/data/` with the collected data:
   - `session_completion.csv` - Daily completion counts and rates
   - `session_duration.csv` - Daily duration statistics

2. Each CSV should have a date column and metric columns with appropriate headers

**Phase 3: Chart Generation**

Generate exactly **2 high-quality trend charts**:

**Chart 1: Session Completion Trends**
- Multi-line chart showing:
  - Successful completions (line, green)
  - Failed/abandoned sessions (line, red)
  - Completion rate percentage (line with secondary y-axis)
- X-axis: Date (last 30 days)
- Y-axis: Count (left), Percentage (right)
- Save as: `/tmp/gh-aw/python/charts/session_completion_trends.png`

**Chart 2: Session Duration & Efficiency**
- Dual visualization showing:
  - Average session duration (line)
  - Median session duration (line)
  - Sessions with loops (bar chart overlay)
- X-axis: Date (last 30 days)
- Y-axis: Duration in minutes
- Save as: `/tmp/gh-aw/python/charts/session_duration_trends.png`

**Chart Quality Requirements**:
- DPI: 300 minimum
- Figure size: 12x7 inches for better readability
- Use seaborn styling with a professional color palette
- Include grid lines for easier reading
- Clear, large labels and legend
- Title with context (e.g., "Session Completion Rates - Last 30 Days")
- Annotations for significant changes or anomalies

**Phase 4: Upload Charts**

1. Upload both charts using the `upload asset` tool
2. Collect the returned URLs for embedding in the discussion

**Phase 5: Embed Charts in Discussion**

Include the charts in your analysis report with this structure:

```markdown
## 📈 Session Trends Analysis

### Completion Patterns
![Session Completion Trends](URL_FROM_UPLOAD_ASSET_CHART_1)

[Brief 2-3 sentence analysis of completion trends, highlighting improvements in success rates or concerning patterns]

### Duration & Efficiency
![Session Duration Trends](URL_FROM_UPLOAD_ASSET_CHART_2)

[Brief 2-3 sentence analysis of session duration patterns, noting efficiency improvements or areas needing attention]
```

### Python Implementation Notes

- Use pandas for data manipulation and date handling
- Use matplotlib.pyplot and seaborn for visualization
- Set appropriate date formatters for x-axis labels
- Use `plt.xticks(rotation=45)` for readable date labels
- Apply `plt.tight_layout()` before saving
- Handle cases where data might be sparse or missing

### Error Handling

If insufficient data is available (less than 7 days):
- Generate the charts with available data
- Add a note in the analysis mentioning the limited data range
- Consider using a bar chart instead of line chart for very sparse data

---

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

**Pre-fetched Data Available**: This workflow includes a shared component (`copilot-session-data-fetch.md`) that fetches Copilot agent session data. The data should be available at:
- `/tmp/gh-aw/session-data/sessions-list.json` - List of sessions with metadata
- `/tmp/gh-aw/session-data/logs/` - Individual session log files

**Verify Setup**:
1. Confirm session data was downloaded successfully
2. Initialize or restore cache-memory from `/tmp/gh-aw/cache-memory/`
3. Load historical analysis data if available

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
   jq '.' /tmp/gh-aw/session-data/sessions-list.json
   
   # Count sessions
   jq 'length' /tmp/gh-aw/session-data/sessions-list.json
   
   # List log files
   find /tmp/gh-aw/session-data/logs/ -maxdepth 1 -ls
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

For each downloaded session log in `/tmp/gh-aw/session-data/logs/`:

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

Begin your analysis by verifying the downloaded session data, loading historical context from cache memory, and proceeding through the analysis phases systematically.