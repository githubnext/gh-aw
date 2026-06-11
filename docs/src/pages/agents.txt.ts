import type { APIRoute } from 'astro';
import { getAwPrompts } from './_aw-prompts.js';

export const GET: APIRoute = () => {
	const prompts = getAwPrompts();

	const lines = [
		'# GitHub Agentic Workflows – Agent Prompts',
		'',
		'> Agent-optimised prompt files for GitHub Agentic Workflows (gh-aw).',
		'> Use these files to ground AI agents working with gh-aw workflows.',
		'',
		'## Prompts',
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
