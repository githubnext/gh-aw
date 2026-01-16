---
title: Multi-Agent Migration
description: Example of migrating parallel workflows to SDK mode using multi-agent orchestration.
---

This example demonstrates converting a workflow with parallel jobs to SDK mode using multi-agent coordination.

## Original CLI Workflow

```yaml
---
title: Comprehensive Code Analysis
engine: copilot
on:
  pull_request:
    types: [opened, synchronize]
jobs:
  security-scan:
    steps:
      - name: Security analysis
        uses: copilot
        with:
          instructions: Perform security vulnerability scan
  
  performance-analysis:
    steps:
      - name: Performance review
        uses: copilot
        with:
          instructions: Analyze performance implications
  
  quality-check:
    steps:
      - name: Code quality review
        uses: copilot
        with:
          instructions: Review code quality and best practices
  
  aggregate:
    needs: [security-scan, performance-analysis, quality-check]
    steps:
      - name: Combine results
        uses: copilot
        with:
          instructions: Aggregate all findings and provide summary
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---
```

**Limitations:**
- Jobs don't run in parallel (GitHub Actions limitation)
- No shared context between jobs
- Aggregation job cannot access detailed findings
- Difficult to coordinate specialized agents

## Migrated SDK Workflow

```yaml
---
title: Comprehensive Code Analysis
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 150000
    allocation:
      security_expert: 50000
      performance_expert: 50000
      quality_expert: 40000
      coordinator: 10000
  
  agents:
    # Parallel specialist agents
    - name: security_expert
      role: "Security vulnerability analysis specialist"
      model: gpt-5
      parallel: true
      priority: 1
      session:
        persistent: true
        max-turns: 8
      budget:
        max-tokens: 50000
      tools:
        inline:
          - name: security_scan
            description: "Run security vulnerability scanner"
            timeout: 120
            implementation: |
              const { execSync } = require('child_process');
              try {
                execSync('npm audit --json > audit.json');
                const audit = require('./audit.json');
                return {
                  vulnerabilities: audit.metadata.vulnerabilities,
                  critical: audit.metadata.vulnerabilities.critical,
                  high: audit.metadata.vulnerabilities.high,
                  advisories: Object.values(audit.advisories).slice(0, 10)
                };
              } catch (error) {
                const audit = require('./audit.json');
                return {
                  vulnerabilities: audit.metadata.vulnerabilities,
                  advisories: Object.values(audit.advisories).slice(0, 10)
                };
              }
    
    - name: performance_expert
      role: "Performance analysis specialist"
      model: claude-sonnet-4
      parallel: true
      priority: 2
      session:
        persistent: true
        max-turns: 8
      budget:
        max-tokens: 50000
      tools:
        inline:
          - name: analyze_complexity
            description: "Analyze code complexity metrics"
            implementation: |
              const fs = require('fs');
              const path = require('path');
              
              function analyzeFile(filePath) {
                const content = fs.readFileSync(filePath, 'utf8');
                const lines = content.split('\n');
                
                return {
                  file: filePath,
                  lines: lines.length,
                  functions: (content.match(/function\s+\w+/g) || []).length,
                  loops: (content.match(/\b(for|while)\b/g) || []).length,
                  conditionals: (content.match(/\bif\b/g) || []).length
                };
              }
              
              function scanDirectory(dir) {
                const metrics = [];
                const files = fs.readdirSync(dir, { withFileTypes: true });
                
                for (const file of files) {
                  const fullPath = path.join(dir, file.name);
                  if (file.isDirectory() && file.name !== 'node_modules') {
                    metrics.push(...scanDirectory(fullPath));
                  } else if (file.isFile() && /\.(js|ts)$/.test(file.name)) {
                    metrics.push(analyzeFile(fullPath));
                  }
                }
                
                return metrics;
              }
              
              const metrics = scanDirectory('src');
              const totalLines = metrics.reduce((sum, m) => sum + m.lines, 0);
              const avgComplexity = metrics.reduce((sum, m) => 
                sum + m.loops + m.conditionals, 0) / metrics.length;
              
              return {
                files_analyzed: metrics.length,
                total_lines: totalLines,
                average_complexity: avgComplexity.toFixed(2),
                high_complexity_files: metrics
                  .filter(m => (m.loops + m.conditionals) > 10)
                  .map(m => ({ file: m.file, complexity: m.loops + m.conditionals }))
              };
    
    - name: quality_expert
      role: "Code quality and best practices specialist"
      model: gpt-5
      parallel: true
      priority: 3
      session:
        persistent: true
        max-turns: 8
      budget:
        max-tokens: 40000
      tools:
        inline:
          - name: check_best_practices
            description: "Check adherence to coding best practices"
            implementation: |
              const fs = require('fs');
              const path = require('path');
              
              function checkFile(filePath) {
                const content = fs.readFileSync(filePath, 'utf8');
                const issues = [];
                
                // Check for long functions
                const functionLengths = [];
                let inFunction = false;
                let functionStart = 0;
                
                content.split('\n').forEach((line, idx) => {
                  if (/function\s+\w+/.test(line)) {
                    inFunction = true;
                    functionStart = idx;
                  }
                  if (inFunction && line.includes('}')) {
                    const length = idx - functionStart;
                    if (length > 50) {
                      issues.push({
                        type: 'long_function',
                        line: functionStart,
                        length: length
                      });
                    }
                    inFunction = false;
                  }
                });
                
                // Check for magic numbers
                const magicNumbers = content.match(/\b\d{3,}\b/g);
                if (magicNumbers) {
                  issues.push({
                    type: 'magic_numbers',
                    count: magicNumbers.length
                  });
                }
                
                // Check for TODO comments
                const todos = content.split('\n').filter(line => 
                  line.includes('TODO')
                );
                if (todos.length > 0) {
                  issues.push({
                    type: 'todos',
                    count: todos.length
                  });
                }
                
                return {
                  file: filePath,
                  issues: issues
                };
              }
              
              function scanDirectory(dir) {
                const results = [];
                const files = fs.readdirSync(dir, { withFileTypes: true });
                
                for (const file of files) {
                  const fullPath = path.join(dir, file.name);
                  if (file.isDirectory() && file.name !== 'node_modules') {
                    results.push(...scanDirectory(fullPath));
                  } else if (file.isFile() && /\.(js|ts)$/.test(file.name)) {
                    results.push(checkFile(fullPath));
                  }
                }
                
                return results;
              }
              
              const results = scanDirectory('src');
              const totalIssues = results.reduce((sum, r) => 
                sum + r.issues.length, 0);
              
              return {
                files_checked: results.length,
                total_issues: totalIssues,
                files_with_issues: results.filter(r => r.issues.length > 0)
              };
    
    # Coordinator agent
    - name: coordinator
      role: "Coordinate and aggregate all specialist findings"
      dependencies: [security_expert, performance_expert, quality_expert]
      session:
        restore: merge  # Merge context from all specialists
        max-turns: 5
      budget:
        max-tokens: 10000
  
  coordination:
    strategy: hybrid
    parallel-stages:
      - agents: [security_expert, performance_expert, quality_expert]
    sequential-stages:
      - agents: [coordinator]
    merge: aggregate
  
  events:
    streaming: true
    handlers:
      - on: agent_completion
        action: log
      
      - on: token_usage
        action: custom
        filter: "data.budget_used_percent >= 80"
        config:
          implementation: |
            const core = require('@actions/core');
            core.warning(`Agent ${event.agent_name} using ${event.data.budget_used_percent}% of budget`);
on:
  pull_request:
    types: [opened, synchronize]
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---

# Comprehensive Code Analysis

Multi-agent code analysis with specialized experts working in parallel.

## Specialist Agent Instructions

### Security Expert
Perform comprehensive security analysis:
- Use security_scan tool to check for vulnerabilities
- Review authentication and authorization code
- Check for common security anti-patterns
- Identify potential injection vulnerabilities
- Provide detailed security recommendations

### Performance Expert
Analyze performance implications:
- Use analyze_complexity tool to measure code complexity
- Identify inefficient algorithms
- Check for potential performance bottlenecks
- Review database query patterns
- Suggest optimization opportunities

### Quality Expert
Review code quality and best practices:
- Use check_best_practices tool to scan for issues
- Check code organization and structure
- Review error handling patterns
- Assess test coverage
- Verify documentation completeness

## Coordinator Instructions

After all specialists complete their analysis:
1. Aggregate findings from all three experts
2. Identify critical issues requiring immediate attention
3. Prioritize recommendations by impact and effort
4. Provide comprehensive summary with:
   - Executive summary
   - Critical issues (must fix)
   - Important improvements (should fix)
   - Optional enhancements (nice to have)
5. Post detailed comment to PR with all findings
```

## Migration Benefits

### Before (CLI with Sequential Jobs)

❌ Jobs run sequentially (slow)  
❌ No context sharing between jobs  
❌ Aggregation lacks access to detailed findings  
❌ Cannot coordinate specialized analysis  
❌ Difficult to balance workload

### After (SDK with Multi-Agent)

✅ Specialists run in parallel (fast)  
✅ Coordinator has full context from all agents  
✅ Each agent specialized for specific domain  
✅ Intelligent workload distribution  
✅ Budget controls per agent

## Key SDK Features

### 1. Parallel Agent Execution

```yaml
agents:
  - name: security_expert
    parallel: true
  - name: performance_expert
    parallel: true
  - name: quality_expert
    parallel: true
```

All three specialists run simultaneously, reducing total execution time.

### 2. Context Merging

```yaml
- name: coordinator
  dependencies: [security_expert, performance_expert, quality_expert]
  session:
    restore: merge  # Gets context from all specialists
```

Coordinator has complete view of all specialist findings.

### 3. Budget Allocation

```yaml
budget:
  allocation:
    security_expert: 50000
    performance_expert: 50000
    quality_expert: 40000
    coordinator: 10000
```

Fair distribution prevents any agent from consuming all resources.

### 4. Agent Specialization

Each agent has:
- Specific role and expertise
- Custom tools for their domain
- Appropriate model (GPT-5 or Claude)
- Dedicated budget

### 5. Hybrid Coordination

```yaml
coordination:
  parallel-stages:
    - agents: [security_expert, performance_expert, quality_expert]
  sequential-stages:
    - agents: [coordinator]
```

Specialists run in parallel, then coordinator aggregates sequentially.

## Execution Flow

```
                  Start
                    │
                    ▼
        ┌───────────────────────┐
        │   Parallel Stage 1    │
        │                       │
        │  ┌─────────────────┐  │
        │  │ Security Expert │  │
        │  └─────────────────┘  │
        │  ┌─────────────────┐  │
        │  │ Perf Expert     │  │
        │  └─────────────────┘  │
        │  ┌─────────────────┐  │
        │  │ Quality Expert  │  │
        │  └─────────────────┘  │
        └───────────┬───────────┘
                    │ (merge contexts)
                    ▼
        ┌───────────────────────┐
        │   Sequential Stage 2  │
        │                       │
        │  ┌─────────────────┐  │
        │  │  Coordinator    │  │
        │  │  (aggregate)    │  │
        │  └─────────────────┘  │
        └───────────┬───────────┘
                    │
                    ▼
                Complete
```

## Performance Comparison

### CLI (Sequential)
```
Security:     2 min
Performance:  2 min
Quality:      2 min
Aggregate:    1 min
─────────────────────
Total:        7 min
```

### SDK (Parallel)
```
Parallel:
  Security:   2 min ─┐
  Performance: 2 min ─┤ (concurrent)
  Quality:    2 min ─┘
Coordinate:   1 min
─────────────────────
Total:        3 min  (57% faster)
```

## Migration Effort

**Time:** 1-2 days  
**Complexity:** High  
**Testing:** Extensive testing required

## Testing Strategy

### 1. Test Individual Agents

```bash
# Test security agent only
gh aw run analysis.md --agent security_expert

# Test performance agent only
gh aw run analysis.md --agent performance_expert

# Test quality agent only
gh aw run analysis.md --agent quality_expert
```

### 2. Test Parallel Execution

```bash
# Run all specialists in parallel
gh aw run analysis.md --stage parallel

# Monitor concurrent execution
gh aw logs analysis --filter agent_name
```

### 3. Test Context Merging

```bash
# Verify coordinator receives all contexts
gh aw run analysis.md
gh aw logs analysis --agent coordinator --show-context
```

### 4. Test Budget Allocation

```bash
# Monitor token usage per agent
gh aw logs analysis --filter token_usage --group-by agent
```

## Advanced Enhancements

### Add Voting Mechanism

```yaml
- name: consensus_coordinator
  role: "Make decision based on specialist votes"
  tools:
    inline:
      - name: count_votes
        implementation: |
          const votes = {
            approve: 0,
            reject: 0,
            conditional: 0
          };
          
          [security_expert, performance_expert, quality_expert]
            .forEach(agent => {
              if (agent.critical_issues === 0) votes.approve++;
              else if (agent.critical_issues > 3) votes.reject++;
              else votes.conditional++;
            });
          
          return {
            votes,
            decision: votes.reject > 0 ? 'reject' : 
                     votes.conditional > 1 ? 'conditional' : 'approve'
          };
```

### Add Progressive Depth

```yaml
agents:
  # Quick scan agents
  - name: quick_security
    role: "Quick security scan"
    parallel: true
    priority: 1
  
  # Deep dive agents (if quick scan finds issues)
  - name: deep_security
    role: "Thorough security analysis"
    dependencies: [quick_security]
    condition: "quick_security.issues > 0"
```

### Add Dynamic Agent Creation

```yaml
tools:
  inline:
    - name: spawn_specialist
      description: "Create specialist agent for specific issue"
      parameters:
        type: object
        required: [issue_type]
        properties:
          issue_type:
            type: string
            enum: [sql_injection, xss, csrf]
      implementation: |
        // Dynamically create specialized agent
        const specialist = createAgent({
          name: `${issue_type}_specialist`,
          role: `Deep analysis of ${issue_type} vulnerabilities`
        });
        return { agent_id: specialist.id };
```

## Troubleshooting

### Issue: Agent Conflicts

Agents modifying same files:

```yaml
tools:
  locking:
    enabled: true
    strategy: optimistic
```

### Issue: Budget Overruns

One agent using too much:

```yaml
agents:
  - name: greedy_agent
    budget:
      max-tokens: 30000  # Hard limit
      action: terminate  # Stop at limit
```

### Issue: Slow Coordination

Coordinator taking too long:

```yaml
- name: coordinator
  session:
    restore: merge
    prune-strategy: smart  # Only keep important context
    state-size-limit: 50000
```

## Related Examples

- [Simple Workflow Migration](./simple-workflow/) - Basic migration
- [Multi-Turn Review](./multi-turn-review/) - Context retention
- [Custom Tools Migration](./custom-tools/) - Inline tools
