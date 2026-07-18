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
		kicker: 'Local tools',
		summary: 'Use your editor, repo clone, and shell.',
		accent: 'var(--sl-color-accent)',
	},
	{
		id: 'vscode',
		label: 'Codespaces',
		icon: 'device-desktop',
		kicker: 'Cloud IDE',
		summary: 'Use VS Code in a GitHub Codespace.',
		accent: 'var(--sl-color-accent-high)',
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