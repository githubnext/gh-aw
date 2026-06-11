/**
 * Shared helper: reads all agent-optimised prompts from .github/aw/*.md and
 * returns metadata needed to build llms.txt / agents.txt.
 */
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

export const RAW_BASE =
	'https://raw.githubusercontent.com/github/gh-aw/main/.github/aw';

export interface AwPrompt {
	file: string;
	description: string;
	rawUrl: string;
}

function parseFrontmatterDescription(content: string): string {
	const match = content.match(/^---[\r\n]+([\s\S]*?)[\r\n]+---/);
	if (!match) return '';
	// Simple key extraction – avoids pulling in a YAML parser at this layer
	const descMatch = match[1].match(/^description:\s*(.+)$/m);
	return descMatch ? descMatch[1].trim() : '';
}

export function getAwPrompts(): AwPrompt[] {
	// process.cwd() is the docs/ directory during `astro build`
	const awDir = join(process.cwd(), '../.github/aw');
	return readdirSync(awDir)
		.filter((f) => f.endsWith('.md'))
		.sort()
		.map((file) => {
			const content = readFileSync(join(awDir, file), 'utf-8');
			return {
				file,
				description: parseFrontmatterDescription(content),
				rawUrl: `${RAW_BASE}/${file}`,
			};
		});
}
