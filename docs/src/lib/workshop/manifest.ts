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
	},
	{
		id: 'terminal',
		label: 'Terminal',
		icon: 'terminal',
		kicker: 'CLI workflow',
		summary: 'Use your editor, repo clone, and shell.',
		accent: 'var(--sl-color-accent)',
	},
	{
		id: 'vscode',
		label: 'VS Code',
		icon: 'device-desktop',
		kicker: 'Local editor',
		summary: 'Stay in VS Code with a local repository and terminal.',
		accent: 'var(--sl-color-accent-high)',
	},
	{
		id: 'copilot',
		label: 'GitHub Copilot',
		icon: 'sparkle-fill',
		kicker: 'Agent assisted',
		summary: 'Use Copilot to draft, compile, and land the workflow.',
		accent: 'var(--sl-color-accent-high)',
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

export const workshopDefaults = {
	journeyId: 'github' as WorkshopJourneyId,
	scenarioId: 'daily-status' as WorkshopScenarioId,
};

export function buildWorkshopFlow(
	journeyId: WorkshopJourneyId,
	scenarioId: WorkshopScenarioId,
): string[] {
	const journey = workshopJourneys.find((item) => item.id === journeyId) ?? workshopJourneys[0];
	const scenario = workshopScenarios.find((item) => item.id === scenarioId) ?? workshopScenarios[0];
	const journeyRoute = workshopRoutes.workspaces[journey.id];
	const scenarioRoute = workshopRoutes.scenarios[scenario.id];

	return [...new Set([
		...journeyRoute.prelude,
		scenarioRoute.designStep,
		scenarioRoute.buildStepByWorkspace[journey.id],
		...journeyRoute.postBuild,
		...workshopRoutes.preSchedule,
		journeyRoute.scheduleStep,
		...workshopRoutes.wrapUp,
	])];
}