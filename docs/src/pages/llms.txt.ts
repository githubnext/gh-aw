import type { APIRoute } from 'astro';
import { getAwPrompts } from './_aw-prompts.js';

export const prerender = true;

export const GET: APIRoute = () => {
	const prompts = getAwPrompts();

	const lines = [
		'# GitHub Agentic Workflows',
		'',
		'> Official documentation and reference index for GitHub Agentic Workflows (gh-aw).',
		'> Optimized for AI systems to discover key docs, examples, API/reference content, and project context.',
		'',
		'## Docs',
		'',
		'- [Documentation Home](https://github.github.com/gh-aw/)',
		'- [Quick Start](https://github.github.com/gh-aw/setup/quick-start/)',
		'- [Creating Workflows](https://github.github.com/gh-aw/setup/creating-workflows/)',
		'- [Workflow Examples](https://github.github.com/gh-aw/examples/)',
		'- [Workflow Patterns](https://github.github.com/gh-aw/patterns/)',
		'',
		'## Blog',
		'',
		'- [Blog Home](https://github.github.com/gh-aw/blog/)',
		'- [What is GitHub Agentic Workflows?](https://github.github.com/gh-aw/blog/2026-01-13-meet-the-workflows/)',
		'',
		'## API / Reference',
		'',
		'- [CLI Reference](https://github.github.com/gh-aw/reference/cli/)',
		'- [Workflow Syntax Reference](https://github.github.com/gh-aw/reference/workflow-syntax/)',
		'- [Engine Reference](https://github.github.com/gh-aw/reference/engines/)',
		'- [MCP Server Reference](https://github.github.com/gh-aw/reference/gh-aw-as-mcp-server/)',
		'',
		'## About',
		'',
		'- [About gh-aw](https://github.github.com/gh-aw/about/)',
		'- [Contributing](https://github.com/github/gh-aw/blob/main/CONTRIBUTING.md)',
		'',
		'## Agent Prompts',
		'',
		'- [Agent Prompt Index](https://github.github.com/gh-aw/agents.txt)',
		'- [Full Prompt Corpus](https://github.github.com/gh-aw/llms-full.txt)',
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
