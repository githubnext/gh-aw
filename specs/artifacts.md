# Artifact File Locations Reference

This document provides a reference for artifact file locations across all agentic workflows.
It is generated automatically and meant to be used by agents when generating file paths in JavaScript and Go code.

## Overview

When artifacts are uploaded, GitHub Actions strips the common parent directory from file paths.
When artifacts are downloaded, files are extracted based on the download mode:

- **Download by name**: Files extracted directly to `path/` (e.g., `path/file.txt`)
- **Download by pattern (no merge)**: Files in `path/artifact-name/` (e.g., `path/artifact-1/file.txt`)
- **Download by pattern (merge)**: Files extracted directly to `path/` (e.g., `path/file.txt`)

## Summary by Job

This section provides an overview of artifacts organized by job name, with duplicates merged across workflows.

### Job: `agent`

**Artifacts Uploaded:**

- `agent-artifacts`
  - **Paths**: `/tmp/gh-aw/agent-stdio.log`, `/tmp/gh-aw/aw-prompts/prompt.txt`, `/tmp/gh-aw/aw.patch`, `/tmp/gh-aw/aw_info.json`, `/tmp/gh-aw/mcp-logs/`, `/tmp/gh-aw/safe-inputs/logs/`, `/tmp/gh-aw/sandbox/firewall/logs/`
  - **Used in**: 78 workflow(s) - agent-performance-analyzer.md, ai-moderator.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-custom-error-patterns.md, example-permissions-warning.md, example-workflow-analyzer.md, firewall.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, metrics-collector.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt-custom-config.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md
- `agent-output`
  - **Paths**: `${{ env.GH_AW_AGENT_OUTPUT }}`
  - **Used in**: 73 workflow(s) - agent-performance-analyzer.md, ai-moderator.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-workflow-analyzer.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md
- `agent_outputs`
  - **Paths**: `/tmp/gh-aw/mcp-config/logs/`, `/tmp/gh-aw/redacted-urls.log`, `/tmp/gh-aw/sandbox/agent/logs/`
  - **Used in**: 65 workflow(s) - agent-performance-analyzer.md, ai-moderator.md, archie.md, artifacts-summary.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-custom-error-patterns.md, example-permissions-warning.md, firewall.md, glossary-maintainer.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, metrics-collector.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-srt-custom-config.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md
- `cache-memory`
  - **Paths**: `/tmp/gh-aw/cache-memory`
  - **Used in**: 31 workflow(s) - ci-coach.md, ci-doctor.md, cloclo.md, copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, grumpy-reviewer.md, issue-template-optimizer.md, mcp-inspector.md, org-health-report.md, pdf-summary.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, scout.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, weekly-issue-summary.md
- `cache-memory-focus-areas`
  - **Paths**: `/tmp/gh-aw/cache-memory-focus-areas`
  - **Used in**: 1 workflow(s) - repository-quality-improver.md
- `data-charts`
  - **Paths**: `/tmp/gh-aw/python/charts/*.png`
  - **Used in**: 10 workflow(s) - copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, github-mcp-structural-analysis.md, org-health-report.md, python-data-charts.md, stale-repo-identifier.md, weekly-issue-summary.md
- `playwright-debug-logs-${{ github.run_id }}`
  - **Paths**: `/tmp/gh-aw/playwright-debug-logs/`
  - **Used in**: 1 workflow(s) - smoke-copilot-playwright.md
- `python-source-and-data`
  - **Paths**: `/tmp/gh-aw/python/*.py`, `/tmp/gh-aw/python/data/*`
  - **Used in**: 10 workflow(s) - copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, github-mcp-structural-analysis.md, org-health-report.md, python-data-charts.md, stale-repo-identifier.md, weekly-issue-summary.md
- `repo-memory-default`
  - **Paths**: `/tmp/gh-aw/repo-memory/default`
  - **Used in**: 8 workflow(s) - agent-performance-analyzer.md, copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-news.md, deep-report.md, metrics-collector.md, security-compliance.md, workflow-health-manager.md
- `safe-output`
  - **Paths**: `${{ env.GH_AW_SAFE_OUTPUTS }}`
  - **Used in**: 73 workflow(s) - agent-performance-analyzer.md, ai-moderator.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-workflow-analyzer.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md
- `safe-outputs-assets`
  - **Paths**: `/opt/gh-aw/safeoutputs/assets/`
  - **Used in**: 14 workflow(s) - copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, github-mcp-structural-analysis.md, org-health-report.md, poem-bot.md, portfolio-analyst.md, python-data-charts.md, stale-repo-identifier.md, technical-doc-writer.md, weekly-issue-summary.md
- `trending-charts`
  - **Paths**: `/tmp/gh-aw/python/charts/*.png`
  - **Used in**: 2 workflow(s) - portfolio-analyst.md, stale-repo-identifier.md
- `trending-source-and-data`
  - **Paths**: `/tmp/gh-aw/python/*.py`, `/tmp/gh-aw/python/data/*`
  - **Used in**: 2 workflow(s) - portfolio-analyst.md, stale-repo-identifier.md

**Artifacts Downloaded:**

- `super-linter-log`
  - **Download paths**: `/tmp/gh-aw/`
  - **Used in**: 1 workflow(s) - super-linter.md

### Job: `conclusion`

**Artifacts Downloaded:**

- `agent-output`
  - **Download paths**: `/opt/gh-aw/safeoutputs/`
  - **Used in**: 73 workflow(s) - agent-performance-analyzer.md, ai-moderator.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-workflow-analyzer.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md

### Job: `detection`

**Artifacts Uploaded:**

- `threat-detection.log`
  - **Paths**: `/tmp/gh-aw/threat-detection/detection.log`
  - **Used in**: 72 workflow(s) - agent-performance-analyzer.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-workflow-analyzer.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md

**Artifacts Downloaded:**

- `agent-artifacts`
  - **Download paths**: `/tmp/gh-aw/threat-detection/`
  - **Used in**: 72 workflow(s) - agent-performance-analyzer.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-workflow-analyzer.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md
- `agent-output`
  - **Download paths**: `/tmp/gh-aw/threat-detection/`
  - **Used in**: 72 workflow(s) - agent-performance-analyzer.md, archie.md, artifacts-summary.md, blog-auditor.md, brave.md, breaking-change-checker.md, campaign-generator.md, changeset.md, ci-coach.md, ci-doctor.md, cli-consistency-checker.md, cloclo.md, commit-changes-analyzer.md, copilot-pr-merged-report.md, copilot-pr-nlp-analysis.md, craft.md, daily-choice-test.md, daily-copilot-token-report.md, daily-fact.md, daily-file-diet.md, daily-issues-report.md, daily-news.md, daily-repo-chronicle.md, deep-report.md, dependabot-go-checker.md, dev-hawk.md, dev.md, dictation-prompt.md, example-workflow-analyzer.md, github-mcp-structural-analysis.md, github-mcp-tools-report.md, glossary-maintainer.md, go-fan.md, go-pattern-detector.md, grumpy-reviewer.md, hourly-ci-cleaner.md, issue-classifier.md, issue-template-optimizer.md, issue-triage-agent.md, layout-spec-maintainer.md, mcp-inspector.md, mergefest.md, notion-issue-summary.md, org-health-report.md, pdf-summary.md, plan.md, playground-org-project-update-issue.md, playground-snapshots-refresh.md, poem-bot.md, portfolio-analyst.md, pr-nitpick-reviewer.md, python-data-charts.md, q.md, release.md, repo-tree-map.md, repository-quality-improver.md, research.md, scout.md, security-compliance.md, slide-deck-maintainer.md, smoke-copilot-playwright.md, smoke-detector.md, smoke-srt.md, stale-repo-identifier.md, super-linter.md, technical-doc-writer.md, tidy.md, typist.md, video-analyzer.md, weekly-issue-summary.md, workflow-generator.md, workflow-health-manager.md

### Job: `post-conclusion`

**Artifacts Downloaded:**

- `agent-artifacts`
  - **Download paths**: `/tmp/gh-aw/`
  - **Used in**: 8 workflow(s) - agent-performance-analyzer.md, copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-news.md, deep-report.md, metrics-collector.md, security-compliance.md, workflow-health-manager.md
- `agent-output`
  - **Download paths**: `/tmp/gh-aw/`
  - **Used in**: 1 workflow(s) - github-mcp-structural-analysis.md
- `safe-output`
  - **Download paths**: `/tmp/gh-aw/`
  - **Used in**: 8 workflow(s) - agent-performance-analyzer.md, copilot-pr-nlp-analysis.md, daily-copilot-token-report.md, daily-news.md, deep-report.md, metrics-collector.md, security-compliance.md, workflow-health-manager.md

### Job: `super-linter`

**Artifacts Uploaded:**

- `super-linter-log`
  - **Paths**: `/tmp/super-linter.log`
  - **Used in**: 1 workflow(s) - super-linter.md
