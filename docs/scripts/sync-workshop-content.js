import { existsSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';

const docsRoot = resolve(import.meta.dirname, '..');
const generatedDir = resolve(docsRoot, 'src/generated');
const markdownOutputDir = join(generatedDir, 'workshop-markdown');
const outputFile = join(generatedDir, 'workshop-content.ts');
const workshopRepo = process.env.GH_AW_WORKSHOP_REPO || 'githubnext/gh-aw-workshop';
const workshopRef = process.env.GH_AW_WORKSHOP_REF || 'main';
const localWorkshopSourceDir = process.env.GH_AW_WORKSHOP_SOURCE_DIR;
const workshopOrgSlug = process.env.GH_AW_WORKSHOP_ORG_SLUG || '2026-07-24-hackathon-blue-bat-18';
const workshopLocalLinkPrefix = `/gh-aw/workshops/${workshopOrgSlug}/?__gh_aw_workshop_local__=`;
const publicWorkshopBase = `https://github.com/${workshopRepo}`;
const publicWorkshopTreeUrl = `${publicWorkshopBase}/tree/${workshopRef}/workshop`;
const publicWorkshopBlobBaseUrl = `${publicWorkshopBase}/blob/${workshopRef}/workshop/`;
const publicWorkshopRawBaseUrl = `https://raw.githubusercontent.com/${workshopRepo}/${workshopRef}/workshop/`;

function getGitHubHeaders() {
	const headers = {
		Accept: 'application/vnd.github+json',
		'User-Agent': 'gh-aw-docs-workshop-sync',
	};

	if (process.env.GITHUB_TOKEN) {
		headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
	}

	return headers;
}

async function fetchJson(url) {
	const response = await fetch(url, { headers: getGitHubHeaders() });
	if (!response.ok) {
		throw new Error(`GET ${url} failed with ${response.status} ${response.statusText}`);
	}
	return response.json();
}

async function fetchText(url) {
	const response = await fetch(url, { headers: getGitHubHeaders() });
	if (!response.ok) {
		throw new Error(`GET ${url} failed with ${response.status} ${response.statusText}`);
	}
	return response.text();
}

async function loadRemoteWorkshopEntries() {
	const contentsUrl = `https://api.github.com/repos/${workshopRepo}/contents/workshop?ref=${encodeURIComponent(workshopRef)}`;
	const contents = await fetchJson(contentsUrl);
	if (!Array.isArray(contents)) {
		throw new Error(`Expected ${contentsUrl} to return a directory listing.`);
	}

	const markdownFiles = contents
		.filter((item) => item.type === 'file' && item.name.endsWith('.md') && item.download_url)
		.sort((left, right) => left.name.localeCompare(right.name, undefined, { numeric: true }));

	return Promise.all(markdownFiles.map(async (item) => ({
		id: item.name,
		body: await fetchText(item.download_url),
	})));
}

function loadLocalWorkshopEntries(sourceDir) {
	if (!existsSync(sourceDir)) {
		throw new Error(`GH_AW_WORKSHOP_SOURCE_DIR does not exist: ${sourceDir}`);
	}

	return readdirSync(sourceDir)
		.filter((name) => name.endsWith('.md'))
		.sort((left, right) => left.localeCompare(right, undefined, { numeric: true }))
		.map((name) => ({
			id: name,
			body: readFileSync(join(sourceDir, name), 'utf8'),
		}));
}

function parseFrontmatter(body) {
	const match = body.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?/u);
	if (!match) return { frontmatter: {}, body };
	const yaml = match[1];
	const frontmatter = /** @type {Record<string, string>} */ ({});
	for (const line of yaml.split('\n')) {
		const colonIdx = line.indexOf(':');
		if (colonIdx > 0) {
			frontmatter[line.slice(0, colonIdx).trim()] = line.slice(colonIdx + 1).trim();
		}
	}
	return { frontmatter, body: body.slice(match[0].length) };
}

function stripMarkdown(value) {
	return String(value)
		.replace(/!\[([^\]]*)\]\([^)]+\)/gu, '$1')
		.replace(/\[([^\]]+)\]\([^)]+\)/gu, '$1')
		.replace(/`([^`]+)`/gu, '$1')
		.replace(/\*\*([^*]+)\*\*/gu, '$1')
		.replace(/_([^_]+)_/gu, '$1')
		.replace(/<[^>]+>/gu, '')
		.trim();
}

function extractTitle(body, fallbackId) {
	const headingMatch = body.match(/^#\s+(.+)$/mu);
	if (headingMatch) return stripMarkdown(headingMatch[1]);

	return normalizeStepId(fallbackId)
		.split('-')
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

function extractSummary(body) {
	const lines = body
		.replace(/^#\s+.+$/mu, '')
		.split('\n')
		.map((line) => line.trim())
		.filter((line) => {
			return Boolean(line)
				&& !line.startsWith('![')
				&& !line.startsWith('>')
				&& !line.startsWith('|')
				&& !line.startsWith('```')
				&& !line.startsWith('- ')
				&& !line.startsWith('* ')
				&& !line.startsWith('## ')
				&& !line.startsWith('### ')
				&& !line.startsWith('_');
		});

	return stripMarkdown(lines[0] ?? 'Continue with this workshop step inside the docs.');
}

function addEntryMetadata(entries) {
	return entries.map((entry) => {
		const { frontmatter, body } = parseFrontmatter(entry.body);
		// Frontmatter is present in fresh remote entries. Cached entries (loaded from the
		// generated file on a transient fetch failure) have already had their frontmatter
		// stripped, so parseFrontmatter returns empty frontmatter — fall back to the
		// existing field value on the entry before falling back to the default.
		return {
			...entry,
			body,
			journey: frontmatter['journey'] || entry.journey || 'all',
			adventure: frontmatter['adventure'] || entry.adventure || 'core',
			title: extractTitle(body, entry.id),
			summary: extractSummary(body),
		};
	});
}

function rewriteWorkshopMarkdownForAstro(body, rawBaseUrl = publicWorkshopRawBaseUrl) {
	return body
		.replace(/\((images\/[^)\s]+)\)/gu, (_match, assetPath) => {
			return `(${new URL(assetPath, rawBaseUrl).toString()})`;
		})
		.replace(/\(([^)\s]+\.md(?:#[^)]+)?)\)/gu, (_match, linkPath) => {
			return `(${workshopLocalLinkPrefix}${encodeURIComponent(linkPath)})`;
		});
}

function loadExistingGeneratedWorkshopEntries() {
	const content = readFileSync(outputFile, 'utf8');
	const serializedEntries = content.match(/export const workshopContent: WorkshopContentEntry\[\] = (\[[\s\S]*\]);\s*$/mu)?.[1];
	if (!serializedEntries) {
		throw new Error(`Could not parse existing generated workshop content from ${outputFile}`);
	}

	// Parse the cached workshopSource block so fallback generation uses the same
	// repo/ref/URLs the content was originally fetched from, rather than the
	// currently configured env-var values (which may differ on a transient failure).
	const sourceBlock = content.match(/export const workshopSource = \{([\s\S]*?)\};/mu)?.[1] ?? '';
	const extractStr = (key) => {
		const escapedKey = key.replace(/[.*+?^${}()|[\]\\]/gu, '\\$&');
		const m = sourceBlock.match(new RegExp(`\\b${escapedKey}:\\s*("(?:[^"\\\\]|\\\\.)*")`, 'u'));
		return m ? JSON.parse(m[1]) : null;
	};
	const cachedRepo = extractStr('repo') ?? workshopRepo;
	const cachedRef = extractStr('ref') ?? workshopRef;
	const cachedTreeUrl = extractStr('treeUrl') ?? publicWorkshopTreeUrl;
	const cachedGithubBaseUrl = extractStr('githubBaseUrl') ?? publicWorkshopBlobBaseUrl;
	const cachedRawBaseUrl = extractStr('rawBaseUrl') ?? publicWorkshopRawBaseUrl;

	return {
		source: cachedTreeUrl,
		sourceMetadata: {
			repo: cachedRepo,
			ref: cachedRef,
			treeUrl: cachedTreeUrl,
			githubBaseUrl: cachedGithubBaseUrl,
			rawBaseUrl: cachedRawBaseUrl,
		},
		entries: JSON.parse(serializedEntries),
	};
}

async function loadWorkshopEntries() {
	if (localWorkshopSourceDir) {
		const sourceDir = resolve(localWorkshopSourceDir);
		return {
			source: sourceDir,
			entries: loadLocalWorkshopEntries(sourceDir),
		};
	}

	try {
		return {
			source: publicWorkshopTreeUrl,
			entries: await loadRemoteWorkshopEntries(),
		};
	} catch (error) {
		if (existsSync(outputFile)) {
			console.warn(`Could not fetch ${workshopRepo}@${workshopRef}; using existing generated content. ${error.message}`);
			return loadExistingGeneratedWorkshopEntries();
		}

		throw error;
	}
}

const { source, sourceMetadata, entries } = await loadWorkshopEntries();
const workshopEntries = addEntryMetadata(entries);

// When falling back to cached content, use the source metadata embedded in that cache
// so that image URLs and workshopSource point to the ref the content actually came from,
// not the currently configured env-var values (which may differ on a transient fetch failure).
const effectiveRepo = sourceMetadata?.repo ?? workshopRepo;
const effectiveRef = sourceMetadata?.ref ?? workshopRef;
const effectiveTreeUrl = sourceMetadata?.treeUrl ?? publicWorkshopTreeUrl;
const effectiveGithubBaseUrl = sourceMetadata?.githubBaseUrl ?? publicWorkshopBlobBaseUrl;
const effectiveRawBaseUrl = sourceMetadata?.rawBaseUrl ?? publicWorkshopRawBaseUrl;

mkdirSync(generatedDir, { recursive: true });
rmSync(markdownOutputDir, { recursive: true, force: true });
mkdirSync(markdownOutputDir, { recursive: true });

for (const entry of entries) {
	const { body } = parseFrontmatter(entry.body);
	writeFileSync(join(markdownOutputDir, entry.id), rewriteWorkshopMarkdownForAstro(body, effectiveRawBaseUrl), 'utf8');
}

const output = `${[
	'// Generated by docs/scripts/sync-workshop-content.js',
	`// Source: ${source}`,
	'// Do not edit by hand.',
	'',
	'export const workshopSource = {',
	`\trepo: ${JSON.stringify(effectiveRepo)},`,
	`\tref: ${JSON.stringify(effectiveRef)},`,
	`\ttreeUrl: ${JSON.stringify(effectiveTreeUrl)},`,
	`\tgithubBaseUrl: ${JSON.stringify(effectiveGithubBaseUrl)},`,
	`\trawBaseUrl: ${JSON.stringify(effectiveRawBaseUrl)},`,
	'};',
	'',
	'export type WorkshopContentEntry = {',
	"\tid: string;",
	"\tjourney: string;",
	"\tadventure: string;",
	"\ttitle: string;",
	"\tsummary: string;",
	"\tbody: string;",
	'};',
	'',
	`export const workshopContent: WorkshopContentEntry[] = ${JSON.stringify(workshopEntries, null, '\t')};`,
	'',
].join('\n')}`;

writeFileSync(outputFile, output, 'utf8');
console.log(`Generated workshop content from ${source}: ${outputFile}`);