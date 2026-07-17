import type { APIRoute } from 'astro';
import { getCollection } from 'astro:content';
import { getAwPrompts } from './_aw-prompts.js';

const BASE_URL = 'https://github.github.com/gh-aw';

/** Human-readable label for each docs section directory. */
const SECTION_LABELS: Record<string, string> = {
	introduction: 'Introduction',
	setup: 'Setup',
	patterns: 'Design Patterns',
	guides: 'Guides',
	reference: 'Reference',
	experimental: 'Experimental',
	specs: 'Specifications',
	troubleshooting: 'Troubleshooting',
};

/** Sections to include, in display order. */
const SECTION_ORDER = [
	'introduction',
	'setup',
	'patterns',
	'guides',
	'reference',
	'experimental',
	'specs',
	'troubleshooting',
];

interface DocPageData {
	title: string;
	description?: string;
}

export const GET: APIRoute = async () => {
	const allDocs = await getCollection('docs');
	const prompts = getAwPrompts();

	// Group docs by first path segment, excluding blog and contributing pages.
	const bySection = new Map<string, Array<{ id: string; title: string; description?: string }>>();
	for (const doc of allDocs) {
		const parts = doc.id.split('/');
		const section = parts.length > 1 ? parts[0] : 'root';
		if (!(section in SECTION_LABELS)) continue;

		if (!bySection.has(section)) bySection.set(section, []);
		const data = doc.data as DocPageData;
		bySection.get(section)!.push({
			id: doc.id,
			title: data.title,
			description: data.description,
		});
	}

	// Sort pages within each section by their id (preserves sidebar order roughly).
	for (const pages of bySection.values()) {
		pages.sort((a, b) => a.id.localeCompare(b.id));
	}

	const lines: string[] = [
		'# GitHub Agentic Workflows',
		'',
		'> Write agentic AI workflows in natural language markdown, run as GitHub Actions.',
		'> GitHub Agentic Workflows (gh-aw) is a GitHub CLI extension for AI-powered workflow automation.',
		'',
	];

	for (const section of SECTION_ORDER) {
		const pages = bySection.get(section);
		if (!pages || pages.length === 0) continue;

		lines.push(`## ${SECTION_LABELS[section]}`, '');
		for (const { id, title, description } of pages) {
			const url = `${BASE_URL}/${id}/`;
			lines.push(description ? `- [${title}](${url}): ${description}` : `- [${title}](${url})`);
		}
		lines.push('');
	}

	if (prompts.length > 0) {
		lines.push('## Agent Prompts', '');
		for (const { file, description, rawUrl } of prompts) {
			const label = file.replace(/\.md$/, '');
			lines.push(description ? `- [${label}](${rawUrl}): ${description}` : `- [${label}](${rawUrl})`);
		}
		lines.push('');
	}

	return new Response(lines.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
