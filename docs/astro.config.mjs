// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import starlightLinksValidator from 'starlight-links-validator';
import starlightGitHubAlerts from 'starlight-github-alerts';
import starlightBlog from 'starlight-blog';
import mermaid from 'astro-mermaid';
import { fileURLToPath } from 'node:url';

/**
 * Creates blog authors config with GitHub profile pictures
 * @param {Record<string, {name: string, url: string, picture?: string}>} authors
 */
function createAuthors(authors) {
	return Object.fromEntries(
		Object.entries(authors).map(([key, author]) => [
			key,
			{ ...author, picture: author.picture ?? `https://github.com/${key}.png?size=200` }
		])
	);
}

// NOTE: A previous attempt defined a custom Shiki grammar for `aw` (agentic workflow) but
// Shiki did not register it and builds produced a warning: language "aw" not found.
// For now we alias `aw` -> `markdown` which removes the warning and still gives
// reasonable highlighting for examples that combine frontmatter + markdown.
// If richer highlighting is needed later, implement a proper TextMate grammar
// in a separate JSON file and load it here (ensure required embedded scopes exist).

// https://astro.build/config
export default defineConfig({
	site: 'https://githubnext.github.io',
	base: '/gh-aw/',
	vite: {
		server: {
			fs: {
				allow: [
					fileURLToPath(new URL('../', import.meta.url)),
				],
			},
		},
	},
	devToolbar: {
		enabled: false
	},
	experimental: {
		clientPrerender: false
	},
	integrations: [
		mermaid(),
		starlight({
			title: 'GitHub Agentic Workflows',
			logo: {
				src: './src/assets/agentic-workflow.svg',
				replacesTitle: false,
			},
		components: {
				Head: './src/components/CustomHead.astro',
				SocialIcons: './src/components/CustomHeader.astro',
				ThemeSelect: './src/components/ThemeToggle.astro',
				Footer: './src/components/CustomFooter.astro',
				SiteTitle: './src/components/CustomLogo.astro',
			},
			customCss: [
				'./src/styles/custom.css',
			],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/githubnext/gh-aw' },
			],
			tableOfContents: { 
			minHeadingLevel: 2, 
			maxHeadingLevel: 4 
		},
			pagination: true,
			expressiveCode: {
				frames: false,
				shiki: {
						langs: /** @type {any[]} */ ([
							"markdown",
							"yaml"
						]),
						langAlias: { aw: "markdown" }
				},
			},
			plugins: [
				starlightBlog({
					recentPostCount: 12,
					authors: createAuthors({
						'githubnext': {
							name: 'GitHub Next',
							url: 'https://githubnext.com/',
						},
						'dsyme': {
							name: 'Don Syme',
							url: 'https://dsyme.net/',
						},
						'pelikhan': {
							name: 'Peli de Halleux',
							url: 'https://www.microsoft.com/research/people/jhalleux/',
						},
						'mnkiefer': {
							name: 'Mara Kiefer',
							url: 'https://github.com/mnkiefer',
						},
						'claude': {
							name: 'Claude AI',
							url: 'https://claude.ai',
						},
						'copilot': {
							name: 'GitHub Copilot',
							url: 'https://github.com/features/copilot',
							picture: 'https://avatars.githubusercontent.com/in/1143301?s=64&amp;v=4',
						},
					}),
				}),
				starlightGitHubAlerts(),
				starlightLinksValidator({
					errorOnRelativeLinks: true,
					errorOnLocalLinks: true,
				}),
				starlightLlmsTxt({
					description: 'GitHub Agentic Workflows (gh-aw) is a Go-based GitHub CLI extension that enables writing agentic workflows in natural language using markdown files, and running them as GitHub Actions workflows.',
					optionalLinks: [
						{
							label: 'GitHub Repository',
							url: 'https://github.com/githubnext/gh-aw',
							description: 'Source code and development resources for gh-aw'
						},
						{
							label: 'GitHub CLI Documentation',
							url: 'https://cli.github.com/manual/',
							description: 'Documentation for the GitHub CLI tool'
						}
					]
				})
			],
			sidebar: [
				{
					label: 'Introduction',
					autogenerate: { directory: 'introduction' },
				},
				{
					label: 'Setup',
					items: [
						{ label: 'Quick Start', link: '/setup/quick-start/' },
						{ label: 'Creating Workflows', link: '/setup/agentic-authoring/' },
						{ label: 'CLI Commands', link: '/setup/cli/' },
						{ label: 'VS Code Integration', link: '/setup/vscode/' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Packaging & Distribution', link: '/guides/packaging-imports/' },
						{ label: 'Security Best Practices', link: '/guides/security/' },
						{ label: 'Using MCPs', link: '/guides/mcps/' },
						{ label: 'Custom Safe Outputs', link: '/guides/custom-safe-outputs/' },
						{ label: 'Threat Detection', link: '/guides/threat-detection/' },
						{ label: 'Web Search', link: '/guides/web-search/' },
						{ label: 'Ephemerals', link: '/guides/ephemerals/' },
					],
				},
				{
					label: 'Design Patterns',
					items: [
						{ label: 'ChatOps', link: '/examples/comment-triggered/chatops/' },
						{ label: 'DailyOps', link: '/examples/scheduled/dailyops/' },
						{ label: 'IssueOps', link: '/examples/issue-pr-events/issueops/' },
						{ label: 'LabelOps', link: '/examples/issue-pr-events/labelops/' },
						{ label: 'ProjectOps', link: '/examples/issue-pr-events/projectops/' },
						{ label: 'ResearchPlanAssign', link: '/guides/researchplanassign/' },
						{ label: 'MultiRepoOps', link: '/guides/multirepoops/' },
						{ label: 'SideRepoOps', link: '/guides/siderepoops/' },
						{ label: 'TrialOps', link: '/guides/trialops/' },
						{ label: 'AgenticImport', link: '/guides/agentic-import/' },
					],
				},
				{
					label: 'Examples',
					items: [
						{ label: 'Research & Planning', link: '/examples/scheduled/research-planning/' },
						{ label: 'Triage & Analysis', link: '/examples/issue-pr-events/triage-analysis/' },
						{ label: 'Coding & Development', link: '/examples/issue-pr-events/coding-development/' },
						{ label: 'Quality & Testing', link: '/examples/issue-pr-events/quality-testing/' },
					],
				},
				{
					label: 'Campaigns',
					items: [
						{ label: 'Overview', link: '/guides/campaigns/' },
						{ label: 'Campaign Specs', link: '/guides/campaigns/specs/' },
						{ label: 'Getting Started', link: '/guides/campaigns/getting-started/' },
						{ label: 'Project Management', link: '/guides/campaigns/project-management/' },
						{ label: 'CLI Commands', link: '/guides/campaigns/cli-commands/' },
						{ label: 'vs GitHub Security Campaigns', link: '/guides/campaigns/comparison-github-campaigns/' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Glossary', link: '/reference/glossary/' },
						{ label: 'Workflow Structure', link: '/reference/workflow-structure/' },
						{ label: 'Frontmatter', link: '/reference/frontmatter/' },
						{ label: 'Frontmatter (Full)', link: '/reference/frontmatter-full/' },
						{ label: 'Triggers', link: '/reference/triggers/' },
						{ label: 'Schedule Syntax', link: '/reference/schedule-syntax/' },
						{ label: 'Command Triggers', link: '/reference/command-triggers/' },
						{ label: 'Permissions', link: '/reference/permissions/' },
						{ label: 'AI Engines', link: '/reference/engines/' },
						{ label: 'Tools', link: '/reference/tools/' },
						{ label: 'Safe Outputs', link: '/reference/safe-outputs/' },
						{ label: 'Safe Inputs', link: '/reference/safe-inputs/' },
						{ label: 'Custom Safe Outputs', link: '/guides/custom-safe-outputs/' },
						{ label: 'Imports', link: '/reference/imports/' },
						{ label: 'Templating', link: '/reference/templating/' },
						{ label: 'Network Access', link: '/reference/network/' },
						{ label: 'Cache & Memory', link: '/reference/memory/' },
						{ label: 'Concurrency', link: '/reference/concurrency/' },
						{ label: 'Markdown', link: '/reference/markdown/' },
						{ label: 'Custom Agents', link: '/reference/custom-agents/' },
						{ label: 'GH-AW as MCP Server', link: '/setup/mcp-server/' },
						{ label: 'MCP Gateway', link: '/reference/mcp-gateway/' },
					],
				},
				{
					label: 'Troubleshooting',
					autogenerate: { directory: 'troubleshooting' },
				},
				{
					label: 'Agent Factory',
					link: '/agent-factory/',
				},
			],
		}),
	],
});
