/**
 * Shared helper: reads all agent-optimised prompts from .github/aw/*.md and
 * returns metadata needed to build llms.txt / agents.txt.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs';
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

export function getAwDir(): string | null {
	const candidates = [
		join(process.cwd(), '.github', 'aw'),
		join(process.cwd(), '..', '.github', 'aw'),
	];

	for (const candidate of candidates) {
		if (existsSync(candidate)) {
			return candidate;
		}
	}

	return null;
}

export function getAwPromptFiles(): string[] {
	const awDir = getAwDir();
	if (!awDir) {
		return [];
	}

	try {
		return readdirSync(awDir)
			.filter((f) => f.endsWith('.md'))
			.sort();
	} catch {
		return [];
	}
}

export function getAwPrompts(): AwPrompt[] {
	const awDir = getAwDir();
	if (!awDir) {
		return [];
	}

	return getAwPromptFiles().map((file) => {
		const content = readFileSync(join(awDir, file), 'utf-8');
		return {
			file,
			description: parseFrontmatterDescription(content),
			rawUrl: `${RAW_BASE}/${file}`,
		};
	});
}
