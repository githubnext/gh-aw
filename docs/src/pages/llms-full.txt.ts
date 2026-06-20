import type { APIRoute } from 'astro';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

// Root-level instruction files for AI agents working with gh-aw.
// These files guide agents on how to create, install, debug, optimize, and package workflows.
const INSTRUCTION_FILES = ['create.md', 'install.md', 'debug.md', 'optimize.md', 'package.md'];

export const GET: APIRoute = () => {
	// process.cwd() is the docs/ directory during `astro build`
	const repoRoot = join(process.cwd(), '..');

	const sections: string[] = [
		'# GitHub Agentic Workflows — Full Corpus',
		'',
		'> Full content of the agent instruction files for GitHub Agentic Workflows (gh-aw).',
		'> This file is intended for AI agents and LLMs that need the complete instruction material.',
		'',
	];

	for (const file of INSTRUCTION_FILES) {
		const content = readFileSync(join(repoRoot, file), 'utf-8');
		sections.push(`---`, `## ${file}`, ``, content.trim(), ``);
	}

	return new Response(sections.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
