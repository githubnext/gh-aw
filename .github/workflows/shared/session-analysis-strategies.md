---
# Session Analysis Strategies
# Reusable analysis patterns for Copilot session analysis
#
# Usage:
#   imports:
#     - shared/session-analysis-strategies.md
#
# This import provides:
# - Standard and experimental analysis strategies
# - Cache memory management patterns
# - Pattern detection methodologies
---

# Session Analysis Strategies

Comprehensive strategies for analyzing Copilot agent sessions to extract insights, identify patterns, and recommend improvements.

## Standard Analysis Strategies

These strategies should be applied to every session analysis:

### 1. Completion Analysis
- Did the session complete successfully?
- Was the task abandoned or aborted?
- Look for error messages or failure indicators
- Track completion rate

### 2. Loop Detection
- Identify repetitive agent responses
- Detect circular reasoning or stuck patterns
- Count iteration loops without progress
- Flag sessions with excessive retries

### 3. Prompt Structure Analysis
- Analyze task description clarity
- Identify effective prompt patterns
- Cluster prompts by keywords or structure
- Correlate prompt quality with success

### 4. Context Confusion Detection
- Look for signs of missing context
- Identify requests for clarification
- Track contextual misunderstandings
- Note when agent asks for more information

### 5. Error Recovery Analysis
- How does the agent handle errors?
- Track error types and recovery strategies
- Measure time to recover from failures
- Identify successful vs. failed recoveries

### 6. Tool Usage Patterns
- Which tools are used most frequently?
- Are tools used effectively?
- Identify missing or unavailable tools
- Track tool execution success rates

## Experimental Strategies (30% of runs)

**Determine if this is an experimental run**:
```bash
# Generate random number between 0-100 using shell's RANDOM variable
# Note: Requires bash shell. On systems without bash, use: $(od -An -N1 -tu1 /dev/urandom | awk '{print $1}')
RANDOM_VALUE=$((RANDOM % 100))
# If value < 30, this is an experimental run
```

**Novel Analysis Methods to Try** (rotate through these):

### 1. Semantic Clustering
- Group prompts by semantic similarity
- Identify common themes across sessions
- Find outlier prompts that perform differently
- Use keyword extraction and comparison

### 2. Temporal Analysis
- Analyze session duration patterns
- Identify time-of-day effects
- Track performance trends over time
- Correlate timing with success rates

### 3. Code Quality Metrics
- If sessions produce code, analyze quality
- Check for test coverage mentions
- Look for documentation updates
- Track code review feedback

### 4. User Interaction Patterns
- Analyze back-and-forth exchanges
- Measure clarification request frequency
- Track user guidance effectiveness
- Identify optimal interaction patterns

### 5. Cross-Session Learning
- Compare similar tasks across sessions
- Identify improvement over time
- Track recurring issues
- Find evolving solution strategies

**Record Experimental Results**:
- Store strategy name and description
- Record what was measured
- Note insights discovered
- Save to cache for future reference

## Data Collection per Session

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

## Cache Memory Management

### Cache Structure
```
/tmp/gh-aw/cache-memory/session-analysis/
├── history.json           # Historical analysis results
├── strategies.json        # Discovered analytical strategies
└── patterns.json          # Known behavioral patterns
```

### Initialize Cache

If cache files don't exist, create them with initial structure:
```bash
mkdir -p /tmp/gh-aw/cache-memory/session-analysis/

cat > /tmp/gh-aw/cache-memory/session-analysis/history.json << 'EOF'
{
  "analyses": [],
  "last_updated": "YYYY-MM-DD",
  "version": "1.0"
}
EOF
```

### Update Historical Data

Update cache memory with today's analysis:
```bash
# Update history.json with today's results
# Include: date, sessions_analyzed, completion_rate, average_duration_minutes
# Include: experimental_strategy (if applicable), key_insights array
```

### Store Discovered Strategies

If this was an experimental run, save the new strategy:
- Strategy name and description
- Results and effectiveness metrics
- Save to strategies.json

### Update Pattern Database

Add newly discovered patterns:
- Pattern type and frequency
- Correlation with success/failure
- Save to patterns.json

### Maintain Cache Size

Keep cache manageable:
- Retain last 90 days of analysis history
- Keep top 20 most effective strategies
- Maintain comprehensive pattern database

## Insight Synthesis

Aggregate observations across all analyzed sessions:

### Success Factors

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

### Failure Signals

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

### Prompt Quality Indicators

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

## Recommendations Format

Based on the analysis, generate actionable recommendations:

1. **For Users**: How to write better task descriptions
2. **For System**: Potential improvements to agent behavior
3. **For Tools**: Missing capabilities or integrations

Include:
- Prompt improvement templates
- Best practice guidelines
- Tool usage suggestions
- Context provision tips
- Error handling strategies
