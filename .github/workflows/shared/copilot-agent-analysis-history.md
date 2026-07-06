## Historical Data Cold-Start Rebuild

**When to Rebuild:**
- History file doesn't exist
- History file has gaps (missing dates in the last 3 days)
- Insufficient data for trend analysis (< 3 days)

**Rebuilding Strategy:**
1. **Assess Current State**: Check how many days of data you have
2. **Target Collection**: Aim for 3 days maximum (for concise trends)
3. **One Day at a Time**: Query PRs for each missing date separately to avoid context explosion

**For Each Missing Day:**
```
# Query PRs for specific date using keyword search
repo:${{ github.repository }} is:pr "START COPILOT CODING AGENT" created:YYYY-MM-DD..YYYY-MM-DD
```

Or use `list_pull_requests` with date filtering and filter results by agent criteria (see `agent_prs_total` in scratchpad/metrics-glossary.md for scope).

**Process:**
- Start with the oldest missing date in your target range (maximum 3 days ago)
- For each date:
  1. Search for PRs created on that date
  2. Analyze each PR (same as Phase 2)
  3. Calculate daily metrics (same as Phase 4.2)
  4. Add to history.json
  5. Save immediately to preserve progress
- Stop at 3 days total

**Important Constraints:**
- Process dates in chronological order (oldest first)
- Save after processing each day
- **Maximum 3 days** of historical data for concise reporting
- Prioritize data quality over quantity
