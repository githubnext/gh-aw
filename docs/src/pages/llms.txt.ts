import type { APIRoute } from 'astro';
import { getLlmsDocPages, LLMS_SITE_BASE_URL } from './_llms-docs.js';

export const prerender = true;

export const GET: APIRoute = async () => {
	const pages = await getLlmsDocPages();
	const sectionKeys = [...new Set(pages.map((page) => page.sectionKey))];

	const lines = [
		'# GitHub Agentic Workflows Documentation',
		'',
		'> Canonical documentation index for GitHub Agentic Workflows (gh-aw), the GitHub CLI extension that compiles markdown workflows into GitHub Actions.',
		`> Base URL: ${LLMS_SITE_BASE_URL}`,
		`> Published pages: ${pages.length}`,
		'',
		'## Sections',
		'',
	];

	for (const sectionKey of sectionKeys) {
		const sectionPages = pages.filter((page) => page.sectionKey === sectionKey);
		lines.push(`### ${sectionPages[0].sectionTitle}`, '');

		for (const page of sectionPages) {
			lines.push(`- [${page.title}](${page.url}): ${page.description}`);
		}

		lines.push('');
	}

	return new Response(lines.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
