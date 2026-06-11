import type { APIRoute } from 'astro';
import { getAwPrompts } from './_aw-prompts.js';

export const GET: APIRoute = () => {
	const prompts = getAwPrompts();

	const lines = [
		'# GitHub Agentic Workflows',
		'',
		'> Agent-optimised prompt files for GitHub Agentic Workflows (gh-aw).',
		'> These markdown files are designed for AI agents and LLMs, not human readers.',
		'',
		'## Agent Prompts',
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
