# 12 Design Patterns from Peli's Agent Factory

**Fundamental behavioral patterns for successful agentic workflows**

[← Previous: 12 Lessons](02-twelve-lessons.md) | [Back to Index](../index.md) | [Next: Operational Patterns →](04-operational-patterns.md)

---

After developing 145 agents in Peli's Agent Factory, we identified 12 fundamental design patterns that capture the essential behaviors of successful agentic workflows. These patterns emerged organically from solving real problems - they weren't predetermined architectures, but rather patterns we discovered by building, observing, and iterating.

Every workflow in the factory fits into at least one of these patterns. Some workflows combine multiple patterns. Understanding these patterns will help you design your own effective agents.

## Pattern 1: The Read-Only Analyst 🔬

**Observe, analyze, and report without changing anything**

### Description

Agents that gather data, perform analysis, and publish insights through discussions or assets. No write permissions to code. Safe for continuous operation at any frequency.

### When to Use

- Building confidence in agent behavior
- Establishing baselines before automation
- Generating reports and metrics
- Deep research and investigation

### Examples

- [`audit-workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md) - Meta-agent that audits all other agents' runs
- [`portfolio-analyst`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md) - Identifies cost optimization opportunities
- [`session-insights`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-session-insights.md) - Analyzes Copilot usage patterns
- [`org-health-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/org-health-report.md) - Organization-wide health metrics
- [`scout`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/scout.md), [`archie`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/archie.md) - Deep research agents

### Key Characteristics

- `permissions: contents: read` only
- Output via discussions, issues, or artifact uploads
- Can run on any schedule without risk
- Builds trust through transparency
- Often creates visualizations and charts

---

## Pattern 2: The ChatOps Responder 💬

**On-demand assistance via slash commands**

### Description

Agents activated by `/command` mentions in issues or PRs. Role-gated for security. Respond with analysis, visualizations, or actions.

### When to Use

- Interactive code reviews
- On-demand optimizations
- User-initiated research
- Specialized assistance requiring authorization

### Examples

- [`q`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/q.md) - Workflow optimizer
- [`grumpy-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md) - Critical code review with personality
- [`poem-bot`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/poem-bot.md) - Creative verse generation
- [`mergefest`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md) - Branch merging automation
- [`pr-fix`](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md) - Fix failing CI checks on demand

### Key Characteristics

- Triggered by `/command` in issue/PR comments
- Often includes role-gating for security
- Provides immediate feedback
- Uses cache-memory to avoid duplicate work
- Clear personality and purpose

---

## Pattern 3: The Continuous Janitor 🧹

**Automated cleanup and maintenance**

### Description

Agents that propose incremental improvements through PRs. Run on schedules (daily/weekly). Create scoped changes with descriptive labels and commit messages. Human review before merging.

### When to Use

- Keeping dependencies up to date
- Maintaining documentation sync
- Formatting and style consistency
- Small refactorings and cleanups
- File organization improvements

### Examples

- [`daily-workflow-updater`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-workflow-updater.md) - Keeps actions and dependencies current
- [`glossary-maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/glossary-maintainer.md) - Syncs glossary with codebase
- [`daily-file-diet`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-file-diet.md) - Refactors oversized files
- [`hourly-ci-cleaner`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/hourly-ci-cleaner.md) - Repairs CI issues

### Key Characteristics

- Runs on fixed schedules
- Creates PRs for human review
- Makes small, focused changes
- Uses descriptive labels and commits
- Often includes "if no changes" guards

---

## Pattern 4: The Quality Guardian 🛡️

**Continuous validation and compliance enforcement**

### Description

Agents that validate system integrity through testing, scanning, and compliance checks. Run frequently (hourly/daily) to catch regressions early.

### When to Use

- Smoke testing infrastructure
- Security scanning
- Accessibility validation
- Schema consistency checks
- Infrastructure health monitoring

### Examples

- Smoke tests for [`copilot`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-copilot.md), [`claude`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-claude.md), [`codex`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-codex.md)
- [`schema-consistency-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/schema-consistency-checker.md)
- [`breaking-change-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/breaking-change-checker.md)
- [`firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall.md), [`mcp-inspector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md)
- [`daily-accessibility-review`](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md)

### Key Characteristics

- Frequent execution (hourly to daily)
- Clear pass/fail criteria
- Creates issues when validation fails
- Minimal false positives
- Fast execution (heartbeat pattern)

---

## Pattern 5: The Issue & PR Manager 🔗

**Intelligent workflow automation for issues and pull requests**

### Description

Agents that triage, link, label, close, and coordinate issues and PRs. React to events or run on schedules.

### When to Use

- Automating issue triage
- Linking related issues
- Managing sub-issues
- Coordinating merges
- Optimizing issue templates

### Examples

- [`issue-triage-agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md) - Auto-labels and categorizes
- [`issue-arborist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-arborist.md) - Links related issues
- [`mergefest`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md) - Merge coordination
- [`sub-issue-closer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/sub-issue-closer.md) - Closes completed sub-issues

### Key Characteristics

- Event-driven (issue/PR triggers)
- Uses safe outputs for modifications
- Often includes intelligent classification
- Maintains issue relationships
- Respects user intent and context

---

## Pattern 6: The Multi-Phase Improver 🔄

**Progressive work across multiple days with human checkpoints**

### Description

Agents that tackle complex improvements too large for single runs. Three phases: (1) Research and create plan discussion, (2) Infer/setup build infrastructure, (3) Implement changes via PR. Check state each run to determine current phase.

### When to Use

- Large refactoring projects
- Test coverage improvements
- Performance optimization campaigns
- Backlog reduction initiatives
- Quality improvement programs

### Examples

- [`daily-backlog-burner`](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md) - Systematic backlog reduction
- [`daily-perf-improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md) - Performance optimization
- [`daily-test-improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md) - Test coverage enhancement
- [`daily-qa`](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md) - Continuous quality assurance

### Key Characteristics

- Multi-day operation
- Three distinct phases with checkpoints
- Uses repo-memory for state persistence
- Human approval between phases
- Creates comprehensive documentation

---

## Pattern 7: The Code Intelligence Agent 🔍

**Semantic analysis and pattern detection**

### Description

Agents using specialized code analysis tools (Serena, ast-grep) to detect patterns, duplicates, anti-patterns, and refactoring opportunities.

### When to Use

- Finding duplicate code
- Detecting anti-patterns
- Identifying refactoring opportunities
- Analyzing code style consistency
- Type system improvements

### Examples

- [`duplicate-code-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/duplicate-code-detector.md) - Finds code duplicates
- [`semantic-function-refactor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/semantic-function-refactor.md) - Refactoring opportunities
- [`terminal-stylist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/terminal-stylist.md) - Console output analysis
- [`go-pattern-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/go-pattern-detector.md) - Go-specific patterns
- [`typist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/typist.md) - Type analysis

### Key Characteristics

- Uses specialized analysis tools (MCP servers)
- Language-aware or cross-language
- Creates detailed issues with code locations
- Often proposes concrete fixes
- Integrates with IDE workflows

---

## Pattern 8: The Content & Documentation Agent 📝

**Maintain knowledge artifacts synchronized with code**

### Description

Agents that keep documentation, glossaries, slide decks, blog posts, and other content fresh by monitoring codebase changes and updating corresponding docs.

### When to Use

- Keeping docs synchronized
- Maintaining glossaries
- Updating slide decks
- Analyzing multimedia content
- Generating documentation

### Examples

- [`glossary-maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/glossary-maintainer.md) - Glossary synchronization
- [`technical-doc-writer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/technical-doc-writer.md) - Technical documentation
- [`slide-deck-maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/slide-deck-maintainer.md) - Presentation maintenance
- [`ubuntu-image-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ubuntu-image-analyzer.md) - Environment documentation

### Key Characteristics

- Monitors code changes
- Creates documentation PRs
- Uses document analysis tools (markitdown)
- Maintains consistency
- Often includes visualization

---

## Pattern 9: The Meta-Agent Optimizer 🎯

**Agents that monitor and optimize other agents**

### Description

Agents that analyze the agent ecosystem itself. Download workflow logs, classify failures, detect missing tools, track performance metrics, identify cost optimization opportunities.

### When to Use

- Managing agent ecosystems at scale
- Cost optimization
- Performance monitoring
- Failure pattern detection
- Tool availability validation

### Examples

- [`audit-workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md) - Comprehensive workflow auditing
- [`agent-performance-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/agent-performance-analyzer.md) - Agent quality metrics
- [`portfolio-analyst`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md) - Cost optimization
- [`workflow-health-manager`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md) - Health monitoring

### Key Characteristics

- Accesses workflow run data
- Analyzes logs and metrics
- Identifies systemic issues
- Provides actionable recommendations
- Essential for scale

---

## Pattern 10: The Meta-Agent Orchestrator 🚦

**Orchestrate multi-step workflows via state machines**

### Description

Agents that coordinate complex workflows through campaigns or task queue patterns. Track state across runs (open/in-progress/completed).

### When to Use

- Campaign management
- Multi-step coordination
- Workflow generation
- Development monitoring
- Task distribution

### Examples

- [`campaign-generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md) - Creates and coordinates campaigns
- [`workflow-generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-generator.md) - Generates new workflows
- [`dev-hawk`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dev-hawk.md) - Development monitoring

### Key Characteristics

- Manages state across runs
- Uses GitHub primitives (issues, projects)
- Coordinates multiple agents
- Implements workflow patterns
- Often dispatcher-based

---

## Pattern 11: The ML & Analytics Agent 🤖

**Advanced insights through machine learning and NLP**

### Description

Agents that apply clustering, NLP, statistical analysis, or ML techniques to extract patterns from historical data. Generate visualizations and trend reports.

### When to Use

- Pattern discovery in large datasets
- NLP on conversations
- Clustering similar items
- Trend analysis
- Longitudinal studies

### Examples

- [`copilot-session-insights`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-session-insights.md) - Session usage analysis
- [`copilot-pr-nlp-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-nlp-analysis.md) - NLP on PR conversations
- [`prompt-clustering`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/prompt-clustering-analysis.md) - Clusters and categorizes prompts

### Key Characteristics

- Uses ML/statistical techniques
- Requires historical data
- Often uses repo-memory
- Generates visualizations
- Discovers non-obvious patterns

---

## Pattern 12: The Security & Moderation Agent 🔒

**Protect repositories from threats and enforce policies**

### Description

Agents that guard repositories through vulnerability scanning, secret detection, spam filtering, malicious code analysis, and compliance enforcement.

### When to Use

- Security vulnerability scanning
- Secret detection
- Spam and abuse prevention
- Compliance enforcement
- Security fix generation

### Examples

- [`security-compliance`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-compliance.md) - Vulnerability campaigns
- [`firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall.md) - Network security testing
- [`daily-secrets-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-secrets-analysis.md) - Secret scanning
- [`ai-moderator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ai-moderator.md) - Comment spam filtering
- [`security-fix-pr`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-fix-pr.md) - Automated security fixes

### Key Characteristics

- Security-focused permissions
- High accuracy requirements
- Often regulatory-driven
- Creates actionable alerts
- May include auto-remediation

---

## Combining Patterns

Many successful workflows combine multiple patterns. For example:

- **Read-Only Analyst + ML Analytics** - Analyze historical data and generate insights
- **ChatOps Responder + Multi-Phase Improver** - User triggers a multi-day improvement project
- **Quality Guardian + Security Agent** - Validate both quality and security continuously
- **Meta-Agent Optimizer + Meta-Agent Orchestrator** - Monitor and coordinate the ecosystem

## Choosing the Right Pattern

When designing a new agent, ask:

1. **Does it modify anything?** → If no, start with Read-Only Analyst
2. **Is it user-triggered?** → Consider ChatOps Responder
3. **Should it run automatically?** → Choose between Janitor (PRs) or Guardian (validation)
4. **Is it managing other agents?** → Use Meta-Agent Optimizer or Orchestrator
5. **Does it need multiple phases?** → Use Multi-Phase Improver
6. **Is it security-related?** → Apply Security & Moderation pattern

## What's Next?

These design patterns describe *what* agents do behaviorally. But *how* they operate within GitHub's ecosystem requires understanding operational patterns.

In the next article, we'll explore 9 operational patterns for running agents effectively on GitHub.

[← Previous: 12 Lessons](02-twelve-lessons.md) | [Back to Index](../index.md) | [Next: Operational Patterns →](04-operational-patterns.md)
