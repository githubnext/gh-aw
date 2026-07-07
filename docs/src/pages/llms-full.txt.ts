import type { APIRoute } from 'astro';
import { getLlmsDocPages, LLMS_SITE_BASE_URL } from './_llms-docs.js';

export const prerender = true;

export const GET: APIRoute = async () => {
	const pages = await getLlmsDocPages();

	const sections: string[] = [
		'# GitHub Agentic Workflows Documentation — Full Corpus',
		'',
		'> Full text index of the published GitHub Agentic Workflows (gh-aw) documentation site.',
		`> Base URL: ${LLMS_SITE_BASE_URL}`,
		`> Published pages: ${pages.length}`,
		'',
	];

	if (pages.length === 0) {
		sections.push('(No content available.)');
	} else {
		for (const page of pages) {
			sections.push(
				`## ${page.title}`,
				'',
				`URL: ${page.url}`,
				`Description: ${page.description}`,
				`Section: ${page.sectionTitle}`,
				'',
				page.body || '(No page body content available.)',
				'',
			);
		}
	}

	return new Response(sections.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
