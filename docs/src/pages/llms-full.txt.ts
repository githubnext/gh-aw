import type { APIRoute } from 'astro';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { getAwDir, getAwPromptFiles } from './_aw-prompts.js';

export const prerender = true;

export const GET: APIRoute = () => {
	const awDir = getAwDir();
	const files = getAwPromptFiles();

	const sections: string[] = [
		'# GitHub Agentic Workflows — Full Corpus',
		'',
		'> Full content of the agent instruction files for GitHub Agentic Workflows (gh-aw).',
		'> This file is intended for AI agents and LLMs that need the complete instruction material.',
		'',
	];

	if (!awDir || files.length === 0) {
		sections.push('(No content available.)');
	} else {
		for (const file of files) {
			const content = readFileSync(join(awDir, file), 'utf-8');
			sections.push(`<!-- file: ${file} -->`, ``, content, ``);
		}
	}

	return new Response(sections.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
