import type { WorkshopContentEntry } from '../../generated/workshop-content';

export type WorkshopRouteId = 'github' | 'terminal' | 'vscode' | 'copilot';
export type WorkshopScenarioRoute = {
	designStep: string;
	buildStepByWorkspace: Record<WorkshopRouteId, string>;
};
export type WorkshopRoutes = {
	curriculum: CurriculumRow[];
	preSchedule: string[];
	wrapUp: string[];
	workspaces: Record<WorkshopRouteId, {
		prelude: string[];
		postBuild: string[];
		scheduleStep: string;
	}>;
	scenarios: Record<string, WorkshopScenarioRoute>;
	scenarioOptions: Record<string, {
		label: string;
		summary: string;
	}>;
};

type CurriculumRow = {
	order: string;
	title: string;
	links: Array<{
		label: string;
		file: string;
		id: string;
	}>;
};

function normalizeStepId(fileName: string) {
	return fileName.replace(/\.md$/u, '');
}

function stripMarkdown(value: string) {
	return String(value)
		.replace(/!\[([^\]]*)\]\([^)]+\)/gu, '$1')
		.replace(/\[([^\]]+)\]\([^)]+\)/gu, '$1')
		.replace(/`([^`]+)`/gu, '$1')
		.replace(/\*\*([^*]+)\*\*/gu, '$1')
		.replace(/_([^_]+)_/gu, '$1')
		.replace(/<[^>]+>/gu, '')
		.trim();
}

function parseCurriculumRows(readmeBody: string): CurriculumRow[] {
	return readmeBody
		.split('\n')
		.map((line) => line.trim())
		.filter((line) => line.startsWith('|') && !/^\|\s*-+/u.test(line))
		.map((line) => line.slice(1, -1).split('|').map((cell) => cell.trim()))
		.filter((cells) => cells.length >= 3 && cells[0] !== '#')
		.map(([order, fileCell, title]) => {
			const links = [...fileCell.matchAll(/\[([^\]]+)\]\(([^)#]+\.md)(?:#[^)]+)?\)/gu)]
				.map((match) => ({
					label: stripMarkdown(match[1]),
					file: match[2],
					id: normalizeStepId(match[2]),
				}));

			return { order, title: stripMarkdown(title), links };
		})
		.filter((row) => row.links.length > 0);
}

export function createWorkshopRoutes(entries: WorkshopContentEntry[]): WorkshopRoutes {
	const entryById = new Map(entries.map((entry) => [normalizeStepId(entry.id), entry]));
	const readme = entries.find((entry) => entry.id === 'README.md');
	if (!readme) {
		throw new Error('Workshop route sync requires workshop/README.md curriculum metadata.');
	}

	const curriculum = parseCurriculumRows(readme.body);
	const rowsByOrder = new Map<string, CurriculumRow[]>();
	for (const row of curriculum) {
		const rows = rowsByOrder.get(row.order) ?? [];
		rows.push(row);
		rowsByOrder.set(row.order, rows);
	}

	const existing = (stepId: string) => {
		if (!entryById.has(stepId)) {
			throw new Error(`Workshop route references missing step from curriculum: ${stepId}`);
		}
		return stepId;
	};
	const linksForOrder = (order: string) => rowsByOrder.get(order)?.flatMap((row) => row.links) ?? [];
	const onlyLink = (order: string) => {
		const links = linksForOrder(order);
		if (links.length !== 1) throw new Error(`Expected exactly one curriculum link for row ${order}, found ${links.length}.`);
		return existing(links[0].id);
	};
	const linkMatching = (order: string, predicate: (link: CurriculumRow['links'][number]) => boolean, description: string) => {
		const match = linksForOrder(order).find(predicate);
		if (!match) throw new Error(`Could not find ${description} in curriculum row ${order}.`);
		return existing(match.id);
	};
	const isCoreOrder = (order: string) => /^\d+$/u.test(order);
	const orderNumber = (order: string) => Number(order.match(/^\d+/u)?.[0] ?? Number.NaN);
	const routeWith = (parts: Array<string | (() => string)>) => parts.map((part) => typeof part === 'function' ? part() : onlyLink(part));
	const preSchedule = curriculum
		.filter((row) => isCoreOrder(row.order) && orderNumber(row.order) === 12)
		.map((row) => {
			if (row.links.length !== 1) throw new Error(`Expected one pre-schedule link for curriculum row ${row.order}.`);
			return existing(row.links[0].id);
		});
	const wrapUp = curriculum
		.filter((row) => isCoreOrder(row.order) && orderNumber(row.order) >= 14)
		.map((row) => {
			if (row.links.length !== 1) throw new Error(`Expected one wrap-up link for curriculum row ${row.order}.`);
			return existing(row.links[0].id);
		});
	const designRows = curriculum.filter((row) => /^10[a-z]$/u.test(row.order));
	const scenarios = Object.fromEntries(designRows.map((row) => {
		const design = row.links[0];
		const scenarioId = design.id.replace(/^\d+[a-z]-design-/u, '');
		const buildOrder = row.order.replace(/^10/u, '11');
		const buildLinks = linksForOrder(buildOrder).filter((link) => link.id.includes(scenarioId));
		const terminalBuild = buildLinks.find((link) => /terminal/u.test(link.id));
		const githubBuild = buildLinks.find((link) => /-ui$/u.test(link.id));
		if (!terminalBuild || !githubBuild) {
			throw new Error(`Could not derive build paths for scenario ${scenarioId} from curriculum row ${buildOrder}.`);
		}

		return [scenarioId, {
			designStep: existing(design.id),
			buildStepByWorkspace: {
				github: existing(githubBuild.id),
				terminal: existing(terminalBuild.id),
				vscode: existing(terminalBuild.id),
				copilot: linkMatching('11d', () => true, 'GitHub Copilot build step'),
			},
		}];
	}));

	return {
		curriculum,
		preSchedule,
		wrapUp,
		workspaces: {
			github: {
				prelude: routeWith([
					'0',
					'1',
					() => linkMatching('3b', () => true, 'GitHub.com repository setup step'),
					'4',
					'5',
					() => linkMatching('6c', () => true, 'GitHub.com install step'),
					() => linkMatching('7b', () => true, 'GitHub.com first workflow step'),
					'7d',
					'8',
					'8b',
					'9',
					'10',
				]),
				postBuild: [],
				scheduleStep: linkMatching('13b', () => true, 'GitHub.com schedule step'),
			},
			terminal: {
				prelude: routeWith([
					'0',
					'1',
					() => linkMatching('2', (link) => /local/u.test(link.id), 'local setup step'),
					() => linkMatching('3a', () => true, 'terminal repository setup step'),
					'4',
					'5',
					() => linkMatching('6b', () => true, 'local install step'),
					() => linkMatching('7a', (link) => !/part2/u.test(link.id), 'terminal first workflow step'),
					'7d',
					'8',
					'8b',
					'9',
					'10',
				]),
				postBuild: [],
				scheduleStep: linkMatching('13a', () => true, 'terminal schedule step'),
			},
			vscode: {
				prelude: routeWith([
					'0',
					'1',
					() => linkMatching('2', (link) => /local/u.test(link.id), 'local setup step'),
					() => linkMatching('3a', () => true, 'terminal repository setup step'),
					'4',
					'5',
					() => linkMatching('6b', () => true, 'local install step'),
					() => linkMatching('7a', (link) => !/part2/u.test(link.id), 'terminal first workflow step'),
					'7d',
					'8',
					'8b',
					'9',
					'10',
				]),
				postBuild: [],
				scheduleStep: linkMatching('13a', () => true, 'terminal schedule step'),
			},
			copilot: {
				prelude: routeWith([
					'0',
					'1',
					() => linkMatching('3b', () => true, 'GitHub.com repository setup step'),
					'4',
					'5',
					() => linkMatching('6c', () => true, 'GitHub.com install step'),
					() => linkMatching('7c', () => true, 'GitHub Copilot first workflow step'),
					'7d',
					'8',
					'8b',
					'9',
					'10',
				]),
				postBuild: [],
				scheduleStep: linkMatching('13b', () => true, 'GitHub.com schedule step'),
			},
		},
		scenarios,
		scenarioOptions: Object.fromEntries(designRows.map((row) => {
			const design = row.links[0];
			const step = entryById.get(design.id);
			const title = row.title
				.replace(/^.*?:\s*/u, '')
				.replace(/^Design\s+[—-]\s*/iu, '')
				.trim();

			return [design.id.replace(/^\d+[a-z]-design-/u, ''), {
				label: title,
				summary: step?.summary ?? '',
			}];
		})),
	};
}
