import type { APIRoute } from 'astro';
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';

const BASE_URL = 'https://github.github.com/gh-aw';
const SITE_NAME = 'GitHub Agentic Workflows';
const SITE_DESCRIPTION =
	'Official documentation for GitHub Agentic Workflows (gh-aw), a GitHub CLI extension for authoring markdown workflows, compiling them to GitHub Actions, and running AI-powered repository automation with guardrails.';

const SECTION_LABELS = new Map([
	['blog', 'Blog'],
	['contributing', 'Contributing'],
	['examples', 'Examples'],
	['experimental', 'Experimental'],
	['guides', 'Guides'],
	['introduction', 'Introduction'],
	['patterns', 'Patterns'],
	['practices', 'Practices'],
	['reference', 'Reference'],
	['setup', 'Setup'],
	['specs', 'Specifications'],
	['troubleshooting', 'Troubleshooting'],
]);

const OPTIONAL_SECTIONS = new Set(['Blog']);

interface DocPage {
	title: string;
	description: string;
	url: string;
	section: string;
}

export const prerender = true;

function walkDocs(dir: string): string[] {
	return readdirSync(dir, { withFileTypes: true })
		.sort((a, b) => a.name.localeCompare(b.name))
		.flatMap((entry) => {
			const fullPath = join(dir, entry.name);

			if (entry.isDirectory()) {
				return walkDocs(fullPath);
			}

			return /\.(md|mdx)$/.test(entry.name) ? [fullPath] : [];
		});
}

function stripQuotes(value: string): string {
	return value.replace(/^['"]|['"]$/g, '').trim();
}

function frontmatterValue(frontmatter: string, key: string): string {
	const match = frontmatter.match(new RegExp(`^${key}:\\s*(.+)$`, 'm'));
	return match ? stripQuotes(match[1]) : '';
}

function titleFromSlug(slug: string): string {
	const label = slug.split('/').filter(Boolean).at(-1) ?? 'Home';
	return label
		.replace(/[-_]/g, ' ')
		.replace(/\b\w/g, (char) => char.toUpperCase());
}

function sectionFromSlug(slug: string): string {
	const topLevel = slug.split('/').filter(Boolean)[0];
	return SECTION_LABELS.get(topLevel ?? '') ?? 'Main Pages';
}

function urlFromSlug(slug: string): string {
	if (slug === 'index') {
		return `${BASE_URL}/`;
	}

	const route = slug.endsWith('/index') ? slug.slice(0, -'/index'.length) : slug;
	return `${BASE_URL}/${route}/`;
}

function readDocPage(filePath: string): DocPage {
	const content = readFileSync(filePath, 'utf-8');
	const frontmatter = content.match(/^---[\r\n]+([\s\S]*?)[\r\n]+---/)?.[1] ?? '';
	const slug = relative(join(process.cwd(), 'src', 'content', 'docs'), filePath)
		.replaceAll('\\', '/')
		.replace(/\.(md|mdx)$/, '');
	const title = frontmatterValue(frontmatter, 'title') || titleFromSlug(slug);

	return {
		title,
		description: frontmatterValue(frontmatter, 'description'),
		url: urlFromSlug(slug),
		section: sectionFromSlug(slug),
	};
}

export const GET: APIRoute = () => {
	const docsRoot = join(process.cwd(), 'src', 'content', 'docs');
	const pages = walkDocs(docsRoot)
		.map(readDocPage)
		.sort((a, b) => a.url.localeCompare(b.url));

	const groupedPages = new Map<string, DocPage[]>();

	for (const page of pages) {
		const sectionPages = groupedPages.get(page.section) ?? [];
		sectionPages.push(page);
		groupedPages.set(page.section, sectionPages);
	}

	const orderedSections = [
		'Main Pages',
		'Introduction',
		'Setup',
		'Guides',
		'Patterns',
		'Examples',
		'Practices',
		'Reference',
		'Experimental',
		'Specifications',
		'Troubleshooting',
		'Contributing',
		...Array.from(groupedPages.keys())
			.filter((section) => !OPTIONAL_SECTIONS.has(section))
			.filter((section) => ![
				'Main Pages',
				'Introduction',
				'Setup',
				'Guides',
				'Patterns',
				'Examples',
				'Practices',
				'Reference',
				'Experimental',
				'Specifications',
				'Troubleshooting',
				'Contributing',
			].includes(section))
			.sort((a, b) => a.localeCompare(b)),
	];

	const lines = [
		`# ${SITE_NAME}`,
		'',
		`> ${SITE_DESCRIPTION}`,
		'',
		`The main homepage is available at: [${SITE_NAME}](${BASE_URL}/)`,
		'',
	];

	for (const section of orderedSections) {
		const sectionPages = groupedPages.get(section);
		if (!sectionPages?.length) {
			continue;
		}

		lines.push(`## ${section}`, '');
		for (const page of sectionPages) {
			lines.push(
				page.description
					? `- [${page.title}](${page.url}): ${page.description}`
					: `- [${page.title}](${page.url})`
			);
		}
		lines.push('');
	}

	const optionalPages = Array.from(groupedPages.entries())
		.filter(([section]) => OPTIONAL_SECTIONS.has(section))
		.flatMap(([section, sectionPages]) =>
			sectionPages.map((page) =>
				page.description
					? `- [${page.title}](${page.url}): ${section}. ${page.description}`
					: `- [${page.title}](${page.url}): ${section}`
			)
		);

	if (optionalPages.length) {
		lines.push('## Optional', '', ...optionalPages, '');
	}

	return new Response(lines.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
