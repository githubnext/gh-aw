import { getCollection } from 'astro:content';

const SITE_BASE_URL = 'https://github.github.com/gh-aw';

const SECTION_TITLES: Record<string, string> = {
	overview: 'Overview',
	introduction: 'Introduction',
	setup: 'Setup',
	guides: 'Guides',
	patterns: 'Patterns',
	examples: 'Examples',
	reference: 'Reference',
	troubleshooting: 'Troubleshooting',
	practices: 'Practices',
	experimental: 'Experimental',
	specs: 'Specifications',
	blog: 'Blog',
	contributing: 'Contributing',
};

const SECTION_ORDER = Object.keys(SECTION_TITLES);

export interface LlmsDocPage {
	id: string;
	title: string;
	description: string;
	url: string;
	sectionKey: string;
	sectionTitle: string;
	body: string;
}

function getSectionKey(id: string): string {
	if (id === 'index' || id === 'index.md' || id === 'index.mdx' || !id.includes('/')) {
		return 'overview';
	}

	return id.split('/')[0];
}

function getSectionTitle(sectionKey: string): string {
	return SECTION_TITLES[sectionKey] ?? sectionKey
		.split(/[-_]/g)
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

function getPageUrl(id: string): string {
	if (id === 'index' || id === 'index.md' || id === 'index.mdx') {
		return `${SITE_BASE_URL}/`;
	}

	const path = id
		.replace(/\/index\.(md|mdx)$/, '')
		.replace(/\.(md|mdx)$/, '');

	return `${SITE_BASE_URL}/${path}/`;
}

function cleanBody(body: string): string {
	return body
		.replace(/^import\s.+$/gm, '')
		.replace(/^export\s.+$/gm, '')
		.replace(/^\s*<\/?[A-Z][^>]*>\s*$/gm, '')
		.replace(/\r\n/g, '\n')
		.replace(/\n{3,}/g, '\n\n')
		.trim();
}

function compareSectionKeys(left: string, right: string): number {
	const leftIndex = SECTION_ORDER.indexOf(left);
	const rightIndex = SECTION_ORDER.indexOf(right);

	if (leftIndex !== -1 || rightIndex !== -1) {
		if (leftIndex === -1) return 1;
		if (rightIndex === -1) return -1;
		if (leftIndex !== rightIndex) return leftIndex - rightIndex;
	}

	return left.localeCompare(right);
}

export async function getLlmsDocPages(): Promise<LlmsDocPage[]> {
	const pages = await getCollection('docs', ({ data }) => !data.draft);

	return pages
		.map((page) => {
			const sectionKey = getSectionKey(page.id);

			return {
				id: page.id,
				title: page.data.title,
				description: page.data.description ?? '',
				url: getPageUrl(page.id),
				sectionKey,
				sectionTitle: getSectionTitle(sectionKey),
				body: cleanBody(page.body),
			};
		})
		.sort((left, right) => {
			const sectionCompare = compareSectionKeys(left.sectionKey, right.sectionKey);
			if (sectionCompare !== 0) return sectionCompare;

			if (left.url === `${SITE_BASE_URL}/`) return -1;
			if (right.url === `${SITE_BASE_URL}/`) return 1;

			return left.url.localeCompare(right.url);
		});
}

export function getLlmsSiteBaseUrl(): string {
	return `${SITE_BASE_URL}/`;
}
