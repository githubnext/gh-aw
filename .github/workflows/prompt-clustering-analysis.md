---
name: Copilot Agent Prompt Clustering Analysis
on:
  schedule:
    # Every day at 7pm UTC (1 hour after copilot-agent-analysis)
    - cron: "0 19 * * *"
  workflow_dispatch:

permissions: read-all

engine: claude

network:
  allowed:
    - defaults
    - github
    - python

safe-outputs:
  create-discussion:
    title-prefix: "[prompt-clustering] "
    category: "audits"
    max: 1

imports:
  - shared/jqschema.md
  - shared/reporting.md
  - shared/mcp/gh-aw.md

cache:
  - key: prompt-clustering-cache-${{ github.run_id }}
    path: /tmp/gh-aw/prompt-cache
    restore-keys: |
      prompt-clustering-cache-

tools:
  cache-memory: true
  github:
    allowed:
      - search_pull_requests
      - pull_request_read
      - list_pull_requests
      - get_file_contents
      - list_commits
      - get_commit
  bash:
    - "find .github -name '*.md'"
    - "find .github -type f -exec cat {} +"
    - "ls -la .github"
    - "git log --oneline"
    - "git diff"
    - "gh pr list *"
    - "gh search prs *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "python3 *"
    - "pip3 *"

steps:
  - name: Install Python dependencies
    run: |
      pip3 install --user scikit-learn pandas numpy matplotlib seaborn nltk

  - name: Fetch Copilot PR data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directories
      mkdir -p /tmp/gh-aw/pr-data
      mkdir -p /tmp/gh-aw/prompt-cache

      # Calculate date 30 days ago
      DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

      # Search for PRs created by Copilot in the last 30 days
      echo "Fetching Copilot PRs from the last 30 days..."
      gh search prs --repo ${{ github.repository }} \
        --author "@copilot" \
        --created ">=$DATE_30_DAYS_AGO" \
        --json number,title,state,createdAt,closedAt,author,body,labels,url,assignees,repository,mergedAt \
        --limit 1000 \
        > /tmp/gh-aw/pr-data/copilot-prs.json

      # Generate schema for reference
      cat /tmp/gh-aw/pr-data/copilot-prs.json | /tmp/gh-aw/jqschema.sh > /tmp/gh-aw/pr-data/copilot-prs-schema.json

      echo "PR data saved to /tmp/gh-aw/pr-data/copilot-prs.json"
      echo "Schema saved to /tmp/gh-aw/pr-data/copilot-prs-schema.json"
      echo "Total PRs found: $(jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json)"

  - name: Download workflow logs for PR analysis
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Extract PR numbers from the data
      PR_NUMBERS=$(jq -r '.[].number' /tmp/gh-aw/pr-data/copilot-prs.json)
      
      # Create logs directory
      mkdir -p /tmp/gh-aw/workflow-logs
      
      echo "Downloading workflow logs to extract turn counts..."
      
      # For each PR, try to find associated workflow runs
      # We'll download logs using gh-aw logs command for copilot engine
      # This will give us the aw_info.json which contains turn counts
      
      # Download logs for the last 30 days of copilot workflows
      cd /tmp/gh-aw
      
      # Use gh-aw to download copilot workflow logs
      # We'll download to a temporary location then extract what we need
      echo "Using gh-aw logs to download copilot workflow artifacts..."

timeout_minutes: 20

---

# Copilot Agent Prompt Clustering Analysis

You are an AI analytics agent that performs advanced NLP analysis on prompts used in copilot agent tasks to identify patterns, clusters, and insights.

## Mission

Daily analysis of copilot agent task prompts using clustering techniques to identify common patterns, outliers, and opportunities for optimization.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Period**: Last 30 days
- **Available Data**:
  - `/tmp/gh-aw/pr-data/copilot-prs.json` - Full PR data for copilot-created PRs
  - `/tmp/gh-aw/prompt-cache/` - Cache directory for avoiding repeated work

## Task Overview

### Phase 1: Extract Task Prompts from PRs

The pre-fetched PR data is available at `/tmp/gh-aw/pr-data/copilot-prs.json`. Each PR created by the copilot agent contains:

1. **PR Body**: Contains the task description/prompt that was given to the agent
2. **PR Title**: A summary of the task
3. **PR Metadata**: State (merged/closed/open), creation/close dates, labels

**Extract the following from each PR**:

```bash
# Use jq to extract relevant fields
jq -r '.[] | {
  number: .number,
  title: .title,
  body: .body,
  state: .state,
  merged: (.mergedAt != null),
  created: .createdAt,
  closed: .closedAt,
  url: .url
}' /tmp/gh-aw/pr-data/copilot-prs.json > /tmp/gh-aw/pr-data/pr-prompts.jsonl
```

The PR body typically contains:
- A section starting with "START COPILOT CODING AGENT" or similar marker
- The actual task description/prompt
- Technical context and requirements

**Task**: Parse the PR bodies to extract the actual prompt/task text. Look for patterns like:
- Text between markers (e.g., "START COPILOT CODING AGENT" and end markers)
- Issue references or task descriptions
- The first paragraph or section that describes what the agent should do

### Phase 2: Enrich Data with Workflow Metrics

For PRs that have associated workflow runs, we need to extract:

1. **Number of Turns**: How many iterations the agent took
2. **Duration**: How long the task took
3. **Success Metrics**: Token usage, cost, etc.

Use the `gh-aw` MCP server to:

```bash
# Download logs for recent copilot workflows
# This creates directories with aw_info.json containing turn counts
gh-aw logs --engine copilot --start-date -30d -o /tmp/gh-aw/workflow-logs
```

Then extract turn counts from `aw_info.json` files:

```bash
# Find all aw_info.json files and extract turn information
find /tmp/gh-aw/workflow-logs -name "aw_info.json" -exec jq '{
  run_id: .run_id,
  workflow: .workflow_name,
  engine: .engine,
  max_turns: .max_turns,
  actual_turns: .turns,
  duration: .duration,
  cost: .cost
}' {} \; > /tmp/gh-aw/pr-data/workflow-metrics.jsonl
```

**Match PRs to workflow runs** by:
- PR number (if available in workflow metadata)
- Timestamp proximity (PR creation time vs workflow run time)
- Repository context

### Phase 3: Prepare Data for Clustering

Create a structured dataset combining:
- Task prompt text (cleaned and preprocessed)
- PR metadata (outcome, duration)
- Workflow metrics (turns, cost)

Save as JSON or CSV for Python processing:

```bash
# Combine PR data with workflow metrics
jq -s '.' /tmp/gh-aw/pr-data/pr-prompts.jsonl > /tmp/gh-aw/pr-data/combined-data.json
```

### Phase 4: Python NLP Clustering Analysis

Create a Python script to perform clustering analysis on the prompts:

**Script**: `/tmp/gh-aw/analyze-prompts.py`

```python
#!/usr/bin/env python3
import json
import pandas as pd
import numpy as np
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.cluster import KMeans, DBSCAN
from sklearn.decomposition import PCA
import matplotlib.pyplot as plt
import seaborn as sns
from collections import Counter
import re

# Load data
with open('/tmp/gh-aw/pr-data/combined-data.json') as f:
    data = json.load(f)

# Extract prompts and metadata
prompts = []
outcomes = []
pr_numbers = []

for pr in data:
    if pr.get('body'):
        # Extract task text from PR body
        body = pr['body']
        
        # Clean the prompt text
        prompt = clean_prompt(body)
        
        if prompt and len(prompt) > 20:  # Minimum length
            prompts.append(prompt)
            outcomes.append('merged' if pr.get('merged') else pr.get('state'))
            pr_numbers.append(pr.get('number'))

# TF-IDF vectorization
vectorizer = TfidfVectorizer(
    max_features=100,
    stop_words='english',
    ngram_range=(1, 3),
    min_df=2
)
X = vectorizer.fit_transform(prompts)

# K-means clustering (try different k values)
optimal_k = find_optimal_clusters(X)
kmeans = KMeans(n_clusters=optimal_k, random_state=42)
clusters = kmeans.fit_predict(X)

# Analyze clusters
cluster_analysis = analyze_clusters(prompts, clusters, outcomes, pr_numbers)

# Generate report
generate_report(cluster_analysis, vectorizer, kmeans)
```

**Key Analysis Steps**:

1. **Text Preprocessing**:
   - Remove markdown formatting
   - Extract main task description
   - Remove URLs, code blocks, special characters
   - Tokenize and normalize

2. **Feature Extraction**:
   - TF-IDF vectorization
   - N-gram extraction (unigrams, bigrams, trigrams)
   - Identify key terms and phrases

3. **Clustering Algorithms**:
   - K-means clustering (try k=3-10)
   - DBSCAN for outlier detection
   - Determine optimal number of clusters using elbow method or silhouette score

4. **Cluster Analysis**:
   - For each cluster:
     - Extract top keywords/phrases
     - Count number of tasks
     - Calculate success rate (merged vs closed)
     - Calculate average turn count
     - Identify representative examples
   
5. **Insights**:
   - Which types of tasks are most common?
   - Which types have highest success rates?
   - Which types require most iterations?
   - Are there outliers (unusual tasks)?

**Helper Functions**:

```python
def clean_prompt(text):
    """Extract and clean the task prompt from PR body."""
    # Remove markdown code blocks
    text = re.sub(r'```[\s\S]*?```', '', text)
    
    # Extract text after "START COPILOT" marker if present
    if 'START COPILOT' in text.upper():
        parts = re.split(r'START COPILOT.*?\n', text, flags=re.IGNORECASE)
        if len(parts) > 1:
            text = parts[1]
    
    # Remove URLs
    text = re.sub(r'http[s]?://\S+', '', text)
    
    # Remove special characters but keep sentence structure
    text = re.sub(r'[^\w\s\.\,\!\?]', ' ', text)
    
    # Normalize whitespace
    text = ' '.join(text.split())
    
    return text.strip()

def find_optimal_clusters(X, max_k=10):
    """Use elbow method to find optimal number of clusters."""
    inertias = []
    K_range = range(2, min(max_k, len(X)) + 1)
    
    for k in K_range:
        kmeans = KMeans(n_clusters=k, random_state=42)
        kmeans.fit(X)
        inertias.append(kmeans.inertia_)
    
    # Simple elbow detection - look for biggest drop
    diffs = np.diff(inertias)
    elbow = np.argmax(diffs) + 2  # +2 because of diff and range start
    
    return min(elbow, 7)  # Cap at 7 clusters for interpretability

def analyze_clusters(prompts, clusters, outcomes, pr_numbers):
    """Analyze each cluster to extract insights."""
    df = pd.DataFrame({
        'prompt': prompts,
        'cluster': clusters,
        'outcome': outcomes,
        'pr_number': pr_numbers
    })
    
    cluster_info = []
    
    for cluster_id in sorted(df['cluster'].unique()):
        cluster_df = df[df['cluster'] == cluster_id]
        
        info = {
            'cluster_id': cluster_id,
            'size': len(cluster_df),
            'merged_count': sum(cluster_df['outcome'] == 'merged'),
            'success_rate': sum(cluster_df['outcome'] == 'merged') / len(cluster_df),
            'example_prs': cluster_df['pr_number'].head(3).tolist(),
            'sample_prompts': cluster_df['prompt'].head(2).tolist()
        }
        
        cluster_info.append(info)
    
    return cluster_info

def generate_report(cluster_analysis, vectorizer, model):
    """Generate markdown report."""
    report = []
    
    report.append("# Clustering Analysis Results\n")
    report.append(f"\n**Total Clusters**: {len(cluster_analysis)}\n")
    
    # Get top terms per cluster
    order_centroids = model.cluster_centers_.argsort()[:, ::-1]
    terms = vectorizer.get_feature_names_out()
    
    for info in sorted(cluster_analysis, key=lambda x: x['size'], reverse=True):
        cluster_id = info['cluster_id']
        report.append(f"\n## Cluster {cluster_id + 1}\n")
        report.append(f"- **Size**: {info['size']} tasks\n")
        report.append(f"- **Success Rate**: {info['success_rate']:.1%}\n")
        
        # Top keywords for this cluster
        top_terms = [terms[i] for i in order_centroids[cluster_id, :5]]
        report.append(f"- **Keywords**: {', '.join(top_terms)}\n")
        
        report.append(f"- **Example PRs**: {', '.join(f'#{pr}' for pr in info['example_prs'])}\n")
    
    # Save report
    with open('/tmp/gh-aw/pr-data/clustering-report.md', 'w') as f:
        f.write('\n'.join(report))
    
    print('\n'.join(report))
    
    return '\n'.join(report)
```

**Run the analysis**:

```bash
cd /tmp/gh-aw
python3 analyze-prompts.py > /tmp/gh-aw/pr-data/analysis-output.txt
```

### Phase 5: Generate Daily Discussion Report

Create a comprehensive discussion report with:

1. **Overview**: Summary of analysis period and data
2. **General Insights**: 
   - Total tasks analyzed
   - Overall success rate
   - Common task patterns
   - Trends over time

3. **Cluster Analysis**:
   - Description of each cluster
   - Top keywords/themes
   - Success rates per cluster
   - Example tasks from each cluster

4. **Full Data Table**:
   - Table with all PRs analyzed
   - Columns: PR #, Title, Cluster, Outcome, Turns, Keywords

5. **Recommendations**:
   - Which types of tasks work best
   - Which types need improvement
   - Suggested prompt engineering improvements

**Report Template**:

```markdown
# 🔬 Copilot Agent Prompt Clustering Analysis - [DATE]

Daily NLP-based clustering analysis of copilot agent task prompts.

## Summary

**Analysis Period**: Last 30 days
**Total Tasks Analyzed**: [count]
**Clusters Identified**: [count]
**Overall Success Rate**: [percentage]%

<details>
<summary>Full Analysis Report</summary>

## General Insights

- **Most Common Task Type**: [cluster description]
- **Highest Success Rate**: [cluster with best success rate]
- **Most Complex Tasks**: [cluster with most turns/highest complexity]
- **Outliers**: [number of outlier tasks identified]

## Cluster Analysis

### Cluster 1: [Theme/Description]
- **Size**: X tasks ([percentage]% of total)
- **Success Rate**: [percentage]%
- **Average Turns**: [number]
- **Top Keywords**: keyword1, keyword2, keyword3
- **Characteristics**: [description of what makes this cluster unique]
- **Example PRs**: #123, #456, #789

[Representative task example]

---

### Cluster 2: [Theme/Description]
...

## Success Rate by Cluster

| Cluster | Tasks | Success Rate | Avg Turns | Top Keywords |
|---------|-------|--------------|-----------|--------------|
| 1       | 15    | 87%          | 3.2       | refactor, cleanup |
| 2       | 12    | 75%          | 4.1       | bug, fix, error |
| 3       | 8     | 100%         | 2.5       | docs, update |

## Full Data Table

| PR # | Title | Cluster | Outcome | Turns | Keywords |
|------|-------|---------|---------|-------|----------|
| 123  | Fix bug in parser | 2 | Merged | 4 | bug, fix, parser |
| 124  | Update docs | 3 | Merged | 2 | docs, update |
| 125  | Refactor logger | 1 | Merged | 3 | refactor, logger |

## Key Findings

1. **[Finding 1]**: [Description and data supporting this finding]
2. **[Finding 2]**: [Description and data supporting this finding]
3. **[Finding 3]**: [Description and data supporting this finding]

## Recommendations

Based on clustering analysis:

1. **[Recommendation 1]**: [Specific actionable recommendation]
2. **[Recommendation 2]**: [Specific actionable recommendation]
3. **[Recommendation 3]**: [Specific actionable recommendation]

</details>

---

_Generated by Prompt Clustering Analysis (Run: [run_id])_
```

### Phase 6: Cache Management

Use the cache to avoid re-analyzing the same PRs:

**Cache Strategy**:
1. Store processed prompts in `/tmp/gh-aw/prompt-cache/processed-prs.json`
2. Include PR number and last analyzed date
3. On next run, skip PRs that haven't changed
4. Update cache after each analysis

```bash
# Save processed PR list to cache
jq -r '.[].number' /tmp/gh-aw/pr-data/copilot-prs.json | sort > /tmp/gh-aw/prompt-cache/analyzed-prs.txt

# On next run, compare and only process new PRs
comm -13 /tmp/gh-aw/prompt-cache/analyzed-prs.txt <(new-prs) > /tmp/gh-aw/pr-data/new-prs.txt
```

## Important Guidelines

### Data Quality
- **Validate Data**: Ensure PR bodies contain actual task descriptions
- **Handle Missing Data**: Some PRs may have incomplete information
- **Clean Text**: Remove markdown, code blocks, and noise from prompts
- **Normalize**: Standardize text before clustering

### Clustering Quality
- **Choose Appropriate K**: Don't over-cluster (too many small clusters) or under-cluster
- **Validate Clusters**: Manually review sample tasks from each cluster
- **Handle Outliers**: Identify and report unusual tasks separately
- **Semantic Coherence**: Ensure clusters have meaningful themes

### Analysis Quality
- **Statistical Significance**: Require minimum cluster sizes for reporting
- **Actionable Insights**: Focus on findings that can improve agent performance
- **Trend Analysis**: Compare with previous analyses if cache data available
- **Reproducibility**: Document methodology for consistent analysis

### Reporting
- **Be Concise**: Use collapsible sections for detailed data
- **Visualize**: Include cluster visualizations if possible (save as images)
- **Provide Examples**: Show representative tasks from each cluster
- **Actionable**: Include specific recommendations based on findings

## Success Criteria

A successful analysis:
- ✅ Collects all copilot PR data from last 30 days
- ✅ Extracts task prompts from PR bodies
- ✅ Enriches with workflow metrics (turns, duration, cost)
- ✅ Performs NLP clustering with 3-7 meaningful clusters
- ✅ Identifies patterns and insights across clusters
- ✅ Generates comprehensive discussion report with data table
- ✅ Uses cache to avoid duplicate work
- ✅ Provides actionable recommendations

## Edge Cases

### Insufficient Data
If fewer than 10 PRs available:
- Report "Insufficient data for clustering analysis"
- Show summary statistics only
- Skip clustering step

### Clustering Failures
If clustering doesn't converge or produces poor results:
- Try different algorithms (DBSCAN instead of K-means)
- Adjust parameters (different k values, distance metrics)
- Report issues and fall back to simple categorization

### Missing Workflow Logs
If workflow logs unavailable for most PRs:
- Proceed with PR data only
- Note limitation in report
- Focus on prompt analysis without turn counts

Now analyze the prompts and generate your comprehensive report!
