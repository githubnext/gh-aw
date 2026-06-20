import type { APIRoute } from 'astro';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

export const GET: APIRoute = () => {
	// process.cwd() is the docs/ directory during `astro build`
	const repoRoot = join(process.cwd(), '..');
	const awDir = join(repoRoot, '.github', 'aw');

	const files = readdirSync(awDir)
		.filter((f) => f.endsWith('.md'))
		.sort();

	const sections: string[] = [
		'# GitHub Agentic Workflows — Full Corpus',
		'',
		'> Full content of the agent instruction files for GitHub Agentic Workflows (gh-aw).',
		'> This file is intended for AI agents and LLMs that need the complete instruction material.',
		'',
	];

	for (const file of files) {
		const content = readFileSync(join(awDir, file), 'utf-8');
		sections.push(`---`, `## ${file}`, ``, content.trim(), ``);
	}

	return new Response(sections.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
