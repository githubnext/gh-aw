import { workshopContent } from '../../generated/workshop-content';
import { createWorkshopRoutes, type WorkshopRouteId } from './routes';

export const workshopRoutes = createWorkshopRoutes(workshopContent);

export type WorkshopJourneyId = WorkshopRouteId;
export type WorkshopScenarioId = keyof typeof workshopRoutes.scenarios;

export type WorkshopJourney = {
	id: WorkshopJourneyId;
	label: string;
	icon: string;
	kicker: string;
	summary: string;
	accent: string;
	/** Content journey values from workshop frontmatter that match this manifest journey. */
	contentJourneyIds: string[];
};

export type WorkshopEntryPath = {
	id: string;
	journeyId: WorkshopJourneyId;
	label: string;
	icon: string;
	kicker: string;
	summary: string;
	fit: string;
};

export type WorkshopScenario = {
	id: WorkshopScenarioId;
	label: string;
	icon: string;
	kicker: string;
	summary: string;
};

const scenarioDisplay = {
	'daily-status': {
		icon: 'repo',
		kicker: 'Repository pulse',
	},
	'daily-docs': {
		icon: 'book',
		kicker: 'Docs drift control',
	},
	'pr-reviewer': {
		icon: 'code-review',
		kicker: 'Review queue assist',
	},
} satisfies Record<WorkshopScenarioId, Pick<WorkshopScenario, 'icon' | 'kicker'>>;

export const workshopJourneys: WorkshopJourney[] = [
	{
		id: 'github',
		label: 'GitHub.com',
		icon: 'browser',
		kicker: 'Browser only',
		summary: 'Use the web editor and Actions tab.',
		accent: 'var(--sl-color-accent-high)',
		contentJourneyIds: ['ui'],
	},
	{
		id: 'terminal',
		label: 'Terminal',
		icon: 'terminal',
		kicker: 'CLI workflow',
		summary: 'Use your editor, repo clone, and shell.',
		accent: 'var(--sl-color-accent)',
		contentJourneyIds: ['terminal', 'local'],
	},
	{
		id: 'vscode',
		label: 'VS Code',
		icon: 'device-desktop',
		kicker: 'Local editor',
		summary: 'Stay in VS Code with a local repository and terminal.',
		accent: 'var(--sl-color-accent-high)',
		contentJourneyIds: ['local', 'terminal'],
	},
	{
		id: 'copilot',
		label: 'GitHub Copilot',
		icon: 'sparkle-fill',
		kicker: 'Agent assisted',
		summary: 'Use Copilot to draft, compile, and land the workflow.',
		accent: 'var(--sl-color-accent-high)',
		contentJourneyIds: ['copilot', 'ui'],
	},
];

export const workshopEntryPaths: WorkshopEntryPath[] = [
	{
		id: 'ui-learner',
		journeyId: 'github',
		label: 'UI learner',
		icon: 'browser',
		kicker: 'GitHub web UI',
		summary: 'Little or no terminal experience.',
		fit: 'Stay in the browser without terminal setup.',
	},
	{
		id: 'cli-user',
		journeyId: 'terminal',
		label: 'CLI user',
		icon: 'terminal',
		kicker: 'Comfortable in a terminal',
		summary: 'Use your existing local workflow and tools.',
		fit: 'Best when you want direct compiler feedback and shell control.',
	},
	{
		id: 'vscode-user',
		journeyId: 'vscode',
		label: 'VS Code user',
		icon: 'device-desktop',
		kicker: 'Editor-first workflow',
		summary: 'Keep working in VS Code with your local repository.',
		fit: 'Follow the local path, but stay anchored in your editor.',
	},
	{
		id: 'copilot-app-user',
		journeyId: 'copilot',
		label: 'GitHub Copilot app user',
		icon: 'device-desktop',
		kicker: 'Desktop app',
		summary: 'Open your repository in the app and steer an agent.',
		fit: 'Best when you want Copilot to build the workflow and open the PR.',
	},
	{
		id: 'copilot-agents-user',
		journeyId: 'copilot',
		label: 'GitHub Copilot user with the Agents tab enabled',
		icon: 'hubot',
		kicker: 'Browser agent session',
		summary: 'Start a browser session, paste a prompt, and merge a PR.',
		fit: 'No local install needed before the Copilot build path.',
	},
];

export const workshopScenarios: WorkshopScenario[] = [
	...(Object.entries(workshopRoutes.scenarioOptions) as Array<[WorkshopScenarioId, { label: string; summary: string }]>).map(([id, option]) => ({
		id,
		label: option.label,
		summary: option.summary,
		...scenarioDisplay[id],
	})),
];

/**
 * Maps manifest scenario IDs to the corresponding adventure frontmatter value
 * used in workshop content files.
 */
export const workshopScenarioAdventures: Record<WorkshopScenarioId, string> = {
	'daily-status': 'scenario-a',
	'daily-docs': 'scenario-b',
	'pr-reviewer': 'scenario-c',
};

export const workshopDefaults = {
	journeyId: 'github' as WorkshopJourneyId,
	scenarioId: 'daily-status' as WorkshopScenarioId,
};

function normalizeStepId(fileName: string) {
	return fileName.replace(/\.md$/u, '');
}

export function buildWorkshopFlow(
	journeyId: WorkshopJourneyId,
	scenarioId: WorkshopScenarioId,
): string[] {
	const journey = workshopJourneys.find((item) => item.id === journeyId) ?? workshopJourneys[0];
	const scenarioAdventure = workshopScenarioAdventures[scenarioId] ?? '';
	const { contentJourneyIds } = journey;
	const isCopilot = contentJourneyIds.includes('copilot');

	const includedAdventures = new Set(['core', 'setup', 'advanced']);
	if (scenarioAdventure) includedAdventures.add(scenarioAdventure);
	if (isCopilot) includedAdventures.add('scenario-d');

	// Filter entries by journey and adventure; exclude side-quests from the main flow.
	const candidates = workshopContent.filter((entry) => {
		// Copilot journey uses the UI path for setup/scheduling but not for
		// journey-specific scenario build steps; those are covered by scenario-d (11d).
		if (isCopilot && entry.journey === 'ui' && !['core', 'setup', 'advanced', 'scenario-d'].includes(entry.adventure)) {
			return false;
		}
		const journeyMatch = entry.journey === 'all' || contentJourneyIds.includes(entry.journey);
		const adventureMatch = includedAdventures.has(entry.adventure);
		return journeyMatch && adventureMatch && entry.adventure !== 'side-quest';
	});

	// Hub detection: use full content filtered only by journey ID membership (no adventure or
	// copilot-exclusion filter). This correctly identifies hub pages even when the copilot
	// filter removes some journey-specific variants from the candidates set.
	const hubPrefixes = new Set(
		workshopContent
			.filter((e) => contentJourneyIds.includes(e.journey))
			.map((e) => normalizeStepId(e.id).split('-')[0]),
	);

	// Exclude hub/overview pages: a journey:all entry is a hub page when:
	// 1. Its exact prefix matches a journey-specific entry (e.g. '11a' when '11a-build-*-ui.md' exists), OR
	// 2. Its numeric-only prefix (e.g. '06') has letter-suffixed journey-specific variants ('06a', '06b').
	return candidates
		.filter((entry) => {
			if (entry.journey !== 'all') return true;
			const keyPrefix = normalizeStepId(entry.id).split('-')[0];
			// Case 1: exact prefix match (e.g. '11a' hub when '11a-build-*-ui' exists for this journey).
			if (hubPrefixes.has(keyPrefix)) return false;
			// Case 2: numeric-only prefix (e.g. '06'): hub if letter-variant specific entries exist ('06a', '06b').
			const numericOnly = keyPrefix.match(/^(\d+)$/u);
			if (numericOnly) {
				const hubRe = new RegExp(`^${numericOnly[1]}[a-z]`, 'u');
				return ![...hubPrefixes].some((p) => hubRe.test(p));
			}
			return true;
		})
		.map((entry) => normalizeStepId(entry.id))
		.filter((key) => key !== 'README');
}