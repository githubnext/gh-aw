import type { APIRoute } from 'astro';
import { getAwPrompts } from './_aw-prompts.js';

export const prerender = true;

const BASE_URL = 'https://github.github.com/gh-aw';

export const GET: APIRoute = () => {
	const prompts = getAwPrompts();

	const lines = [
		'# GitHub Agentic Workflows',
		'',
		'> GitHub CLI extension that compiles natural language markdown workflows into GitHub Actions with sandboxed AI agent execution.',
		'',
		'GitHub Agentic Workflows (gh-aw) lets you write repository automation in plain Markdown and run AI coding agents — GitHub Copilot, Claude Code, OpenAI Codex, or Google Gemini — inside GitHub Actions, with sandboxed execution, read-only defaults, and safe outputs for writes.',
		'',
		'- Install: `gh extension install github/gh-aw`',
		'- Source: https://github.com/github/gh-aw',
		'- License: MIT',
		'',
		'## Getting Started',
		'',
		`- [Quick Start](${BASE_URL}/setup/quick-start/): Install the extension, pick an AI engine, and run your first workflow in minutes.`,
		`- [Creating Workflows](${BASE_URL}/setup/creating-workflows/): Author agentic workflows in natural language markdown.`,
		`- [CLI Commands](${BASE_URL}/setup/cli/): Reference for all gh aw CLI commands.`,
		`- [How They Work](${BASE_URL}/introduction/how-they-work/): Concepts and architecture of agentic workflows.`,
		`- [Security Architecture](${BASE_URL}/introduction/architecture/): Sandboxed execution, safe outputs, and threat modeling.`,
		'',
		'## AI Engines',
		'',
		`- [GitHub Copilot](${BASE_URL}/engines/copilot/): Run workflows using GitHub Copilot as the AI engine.`,
		`- [Claude Code](${BASE_URL}/engines/claude/): Run workflows using Anthropic Claude as the AI engine.`,
		`- [OpenAI Codex](${BASE_URL}/engines/codex/): Run workflows using OpenAI Codex as the AI engine.`,
		`- [Google Gemini](${BASE_URL}/engines/gemini/): Run workflows using Google Gemini as the AI engine.`,
		'',
		'## Guides',
		'',
		`- [AI Issue Triage](${BASE_URL}/guides/ai-issue-triage/): Automatically triage and label GitHub issues with AI.`,
		`- [Automated PR Review](${BASE_URL}/guides/automated-pr-review/): AI-powered code review for pull requests.`,
		`- [AI Release Notes](${BASE_URL}/guides/ai-release-notes/): Generate release notes automatically with AI.`,
		`- [Docs Automation](${BASE_URL}/guides/docs-automation/): Automate documentation tasks with AI agents.`,
		`- [Using MCPs](${BASE_URL}/guides/mcps/): Integrate Model Context Protocol servers into workflows.`,
		`- [Reusing Workflows](${BASE_URL}/guides/reusing-workflows/): Package and share workflows across repositories.`,
		`- [Governance](${BASE_URL}/guides/governance/): Manage permissions, approvals, and access controls.`,
		`- [Using at Scale](${BASE_URL}/guides/using-at-scale/): Deploy agentic workflows across organizations.`,
		'',
		'## Design Patterns',
		'',
		`- [IssueOps](${BASE_URL}/patterns/issue-ops/): Trigger workflows from GitHub issue events.`,
		`- [ChatOps](${BASE_URL}/patterns/chat-ops/): Trigger workflows from issue and PR comments.`,
		`- [BatchOps](${BASE_URL}/patterns/batch-ops/): Process multiple items in parallel.`,
		`- [DispatchOps](${BASE_URL}/patterns/dispatch-ops/): Manually trigger workflows via workflow_dispatch.`,
		`- [OrchestratorOps](${BASE_URL}/patterns/orchestrator-ops/): Coordinate multiple sub-agents.`,
		`- [MemoryOps](${BASE_URL}/patterns/memory-ops/): Persist context across workflow runs with cache memory.`,
		`- [MultiRepoOps](${BASE_URL}/patterns/multi-repo-ops/): Coordinate automation across multiple repositories.`,
		`- [MonitorOps](${BASE_URL}/patterns/monitor-ops/): Schedule regular monitoring and health-check workflows.`,
		'',
		'## Reference',
		'',
		`- [Frontmatter Reference](${BASE_URL}/reference/frontmatter/): All supported frontmatter fields for workflows.`,
		`- [Safe Outputs](${BASE_URL}/reference/safe-outputs/): How write operations are sanitized and applied safely.`,
		`- [Sandbox and Network](${BASE_URL}/reference/sandbox/): Sandboxed execution environment and network controls.`,
		`- [FAQ](${BASE_URL}/reference/faq/): Frequently asked questions.`,
		`- [Troubleshooting](${BASE_URL}/troubleshooting/common-issues/): Common issues and how to resolve them.`,
		`- [About](${BASE_URL}/about/): About GitHub Agentic Workflows.`,
		'',
		'## Agent Prompt Files',
		'',
		'> Raw instruction files for AI agents working with gh-aw. These are intended for LLM consumption.',
		'',
		...prompts.map(({ file, description, rawUrl }) => {
			const label = file.replace(/\.md$/, '');
			return description
				? `- [${label}](${rawUrl}): ${description}`
				: `- [${label}](${rawUrl})`;
		}),
	];

	return new Response(lines.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
