// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import sitemap from '@astrojs/sitemap';
import starlightLlmsTxt from 'starlight-llms-txt';
import starlightLinksValidator from 'starlight-links-validator';
import starlightGitHubAlerts from 'starlight-github-alerts';
import starlightBlog from 'starlight-blog';
import mermaid from 'astro-mermaid';
import { fileURLToPath } from 'node:url';
import remarkStripEmojis from './src/lib/remark/stripEmojis.js';
import remarkTableDataLabels from './src/lib/remark/tableDataLabels.js';
import rehypeTableWrapper from './src/lib/rehype/tableWrapper.js';

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
	site: 'https://github.github.com',
	base: '/gh-aw/',
	trailingSlash: 'always',
	markdown: {
		remarkPlugins: [remarkStripEmojis, remarkTableDataLabels],
		rehypePlugins: [rehypeTableWrapper],
	},
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
	redirects: {
		// Status → Labs → Agent Factory → Agent Factory Status chain
		'/status/': '/gh-aw/agent-factory-status/',
		'/labs/': '/gh-aw/agent-factory-status/',
		'/agent-factory/': '/gh-aw/agent-factory-status/',

		// Blog post date correction
		'/blog/2026-01-12-meet-the-workflows/': '/gh-aw/blog/2026-01-13-meet-the-workflows/',

		// Start-here → Get-started → current paths
		'/start-here/concepts/': '/gh-aw/introduction/how-they-work/',
		'/start-here/quick-start/': '/gh-aw/setup/quick-start/',

		// Get-started → current paths
		'/get-started/concepts/': '/gh-aw/introduction/how-they-work/',
		'/get-started/quick-start/': '/gh-aw/setup/quick-start/',

		// Introduction how-it-works → how-they-work
		'/introduction/how-it-works/': '/gh-aw/introduction/how-they-work/',

		// Tools → Setup renames
		'/tools/cli/': '/gh-aw/setup/cli/',
		'/tools/mcp-server/': '/gh-aw/reference/gh-aw-as-mcp-server/',
		'/tools/agentic-authoring/': '/gh-aw/setup/creating-workflows/',

		// Samples → Examples renames
		'/samples/coding-development/': '/gh-aw/examples/issue-pr-events/coding-development/',
		'/samples/quality-testing/': '/gh-aw/examples/issue-pr-events/quality-testing/',
		'/samples/triage-analysis/': '/gh-aw/examples/issue-pr-events/triage-analysis/',
		'/samples/research-planning/': '/gh-aw/examples/scheduled/research-planning/',

		// Setup renames
		'/setup/agentic-authoring/': '/gh-aw/setup/creating-workflows/',
		'/setup/mcp-server/': '/gh-aw/reference/gh-aw-as-mcp-server/',

		// Reference renames
		'/reference/custom-agents/': '/gh-aw/reference/copilot-custom-agents/',
		'/reference/custom-agent/': '/gh-aw/reference/custom-agent-for-aw/',

		// Organization Practices moved under Guides
		'/organization-practices/': '/gh-aw/guides/organization-practices/',
		'/organization-practices/safe-rollout/': '/gh-aw/guides/organization-practices/safe-rollout/',
		'/organization-practices/sharing-workflows/': '/gh-aw/guides/organization-practices/sharing-workflows/',

		// Guides → Patterns renames
		'/guides/chatops/': '/gh-aw/patterns/chat-ops/',
		'/guides/issueops/': '/gh-aw/patterns/issue-ops/',
		'/guides/labelops/': '/gh-aw/patterns/label-ops/',
		'/guides/dailyops/': '/gh-aw/patterns/daily-ops/',
		'/guides/dispatchops/': '/gh-aw/patterns/dispatch-ops/',
		'/guides/monitoring/': '/gh-aw/experimental/monitoring-with-projects/',
		'/guides/multirepoops/': '/gh-aw/patterns/multi-repo-ops/',
		'/guides/orchestration/': '/gh-aw/patterns/orchestration/',
		'/guides/siderepoops/': '/gh-aw/patterns/side-repo-ops/',
		'/guides/specops/': '/gh-aw/patterns/spec-ops/',
		'/guides/researchplanassign/': '/gh-aw/patterns/research-plan-assign-ops/',
		'/guides/trialops/': '/gh-aw/experimental/trial-ops/',

		// Examples → Patterns renames
		'/examples/comment-triggered/chatops/': '/gh-aw/patterns/chat-ops/',
		'/examples/scheduled/dailyops/': '/gh-aw/patterns/daily-ops/',
		'/examples/issue-pr-events/issueops/': '/gh-aw/patterns/issue-ops/',
		'/examples/issue-pr-events/labelops/': '/gh-aw/patterns/label-ops/',
		'/examples/issue-pr-events/projectops/': '/gh-aw/patterns/project-ops/',

		// Patterns unhyphenated → hyphenated slugs
		'/patterns/centralrepoops/': '/gh-aw/patterns/central-repo-ops/',
		'/patterns/chatops/': '/gh-aw/patterns/chat-ops/',
		'/patterns/dailyops/': '/gh-aw/patterns/daily-ops/',
		'/patterns/dataops/': '/gh-aw/patterns/data-ops/',
		'/patterns/dispatchops/': '/gh-aw/patterns/dispatch-ops/',
		'/patterns/issueops/': '/gh-aw/patterns/issue-ops/',
		'/patterns/labelops/': '/gh-aw/patterns/label-ops/',
		'/patterns/multirepoops/': '/gh-aw/patterns/multi-repo-ops/',
		'/patterns/projectops/': '/gh-aw/patterns/project-ops/',
		'/patterns/siderepoops/': '/gh-aw/patterns/side-repo-ops/',
		'/patterns/specops/': '/gh-aw/patterns/spec-ops/',
		'/patterns/researchplanassignops/': '/gh-aw/patterns/research-plan-assign-ops/',
		'/patterns/batchops/': '/gh-aw/patterns/batch-ops/',
		'/patterns/taskops/': '/gh-aw/patterns/task-ops/',
		'/patterns/trialops/': '/gh-aw/experimental/trial-ops/',
		'/patterns/workqueueops/': '/gh-aw/patterns/workqueue-ops/',

		// Guides → new locations
		'/guides/deterministic-agentic-patterns/': '/gh-aw/patterns/deterministic-ops/',
		'/patterns/deterministic-agentic-patterns/': '/gh-aw/patterns/deterministic-ops/',
		'/guides/custom-otlp-attributes/': '/gh-aw/guides/telemetry/',
		'/guides/audit-with-agents/': '/gh-aw/reference/audit/',
		'/guides/ephemerals/': '/gh-aw/reference/ephemerals/',

		'/reference/awf-reflect/': '/gh-aw/experimental/awf-reflect/',

		// Patterns → Experimental
		'/patterns/correction-ops/': '/gh-aw/experimental/correction-ops/',
		'/patterns/trial-ops/': '/gh-aw/experimental/trial-ops/',
		'/patterns/monitoring/': '/gh-aw/experimental/monitoring-with-projects/',
	},
	integrations: [
		sitemap(),
		mermaid(),
		starlight({
			title: 'GitHub Agentic Workflows',
			description: 'Write agentic workflows in natural language using markdown files and run them as GitHub Actions workflows.',
			favicon: '/favicon.svg',
			logo: {
				src: './src/assets/agentic-workflow.svg',
				replacesTitle: false,
			},
			components: {
				Head: './src/components/CustomHead.astro',
				SkipLink: './src/components/SkipLink.astro',
				SocialIcons: './src/components/CustomHeader.astro',
				ThemeSelect: './src/components/ThemeToggle.astro',
				Footer: './src/components/CustomFooter.astro',
				SiteTitle: './src/components/CustomLogo.astro',
			},
			customCss: [
				'./src/styles/custom.css',
			],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/github/gh-aw' },
			],
			tableOfContents: {
				minHeadingLevel: 2,
				maxHeadingLevel: 4
			},
			pagination: true,
			expressiveCode: {
				frames: {
					showCopyToClipboardButton: true,
				},
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
							name: 'Claude',
							url: 'https://claude.ai',
							picture: '/gh-aw/claude.png',
						},
						'codex': {
							name: 'Codex',
							url: 'https://openai.com/index/openai-codex/',
							picture: '/gh-aw/codex.png',
						},
						'gemini': {
							name: 'Gemini',
							url: 'https://gemini.google.com',
							picture: '/gh-aw/gemini.png',
						},
						'copilot': {
							name: 'Copilot',
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
							url: 'https://github.com/github/gh-aw',
							description: 'Source code and development resources for gh-aw'
						},
						{
							label: 'GitHub CLI Documentation',
							url: 'https://cli.github.com/manual/',
							description: 'Documentation for the GitHub CLI tool'
						}
					],
					customSets: [
						{
							label: "agentic-workflows",
							paths: ['blog/*meet-the-workflows*'],
							description: "A comprehensive blog series documenting workflow patterns, best practices, and real-world examples of agentic workflows created at Peli's Agent Factory"
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
						{ label: 'Creating Workflows', link: '/setup/creating-workflows/' },
						{ label: 'CLI Commands', link: '/setup/cli/' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Agentic Authoring', link: '/guides/agentic-authoring/' },
						{ label: 'Editing Workflows', link: '/guides/editing-workflows/' },
						{ label: 'Getting Started with MCP', link: '/guides/getting-started-mcp/' },
						{ label: 'GitHub Actions Primer', link: '/guides/github-actions-primer/' },
						{ label: 'Maintaining Repositories', link: '/guides/maintaining-repos/' },
						{ label: 'Network Configuration', link: '/guides/network-configuration/' },
						{ label: 'Organization Practices', link: '/guides/organization-practices/' },
						{ label: 'Safe Rollout', link: '/guides/organization-practices/safe-rollout/' },
						{ label: 'Sharing Workflows', link: '/guides/organization-practices/sharing-workflows/' },
						{ label: 'Reusing Workflows', link: '/guides/packaging-imports/' },
						{ label: 'Serena', link: '/guides/serena/' },
						{ label: 'Using Custom MCPs', link: '/guides/mcps/' },
						{ label: 'Upgrading Workflows', link: '/guides/upgrading/' },
						{ label: 'Self-Hosted Runners', link: '/guides/self-hosted-runners/' },
						{ label: 'Web Search', link: '/guides/web-search/' },
						{ label: 'Telemetry', link: '/guides/telemetry/' },
					],
				},
				{
					label: 'Design Patterns',
					items: [
						{ label: 'AgenticOps', link: '/patterns/agentic-ops/' },
						{ label: 'BatchOps', link: '/patterns/batch-ops/' },
						{ label: 'CentralRepoOps', link: '/patterns/central-repo-ops/' },
						{ label: 'ChatOps', link: '/patterns/chat-ops/' },
						{ label: 'DailyOps', link: '/patterns/daily-ops/' },
						{ label: 'DataOps', link: '/patterns/data-ops/' },
						{ label: 'DeterministicOps', link: '/patterns/deterministic-ops/' },
						{ label: 'DispatchOps', link: '/patterns/dispatch-ops/' },
						{ label: 'ExpertOps', link: '/patterns/expert-ops/' },
						{ label: 'IssueOps', link: '/patterns/issue-ops/' },
						{ label: 'LabelOps', link: '/patterns/label-ops/' },
						{ label: 'MemoryOps', link: '/guides/memoryops/' },
						{ label: 'MultiRepoOps', link: '/patterns/multi-repo-ops/' },
						{ label: 'Orchestration', link: '/patterns/orchestration/' },
						{ label: 'ProjectOps', link: '/patterns/project-ops/' },
						{ label: 'ResearchPlanAssignOps', link: '/patterns/research-plan-assign-ops/' },
						{ label: 'SideRepoOps', link: '/patterns/side-repo-ops/' },
						{ label: 'SpecOps', link: '/patterns/spec-ops/' },
						{ label: 'TaskOps', link: '/patterns/task-ops/' },
						{ label: 'WorkQueueOps', link: '/patterns/workqueue-ops/' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'AI Engines', link: '/reference/engines/' },
						{ label: 'Assign to Copilot', link: '/reference/assign-to-copilot/' },
						{ label: 'Artifacts', link: '/reference/artifacts/' },
						{ label: 'Auditing Workflows', link: '/reference/audit/' },
						{ label: 'Authentication', link: '/reference/auth/' },
						{ label: 'Authentication (Projects)', link: '/reference/auth-projects/' },
						{ label: 'aw.yml Package Manifest', link: '/reference/aw-yml-package-manifest/' },
						{ label: 'Cache Memory', link: '/reference/cache-memory/' },
						{ label: 'Command Triggers', link: '/reference/command-triggers/' },
						{ label: 'Compilation Process', link: '/reference/compilation-process/' },
						{ label: 'Concurrency', link: '/reference/concurrency/' },
						{ label: 'Cost Management', link: '/reference/cost-management/' },
						{ label: 'Cost Management (Rate Limiting)', link: '/reference/rate-limiting-controls/' },
						{ label: 'Cost Management (Effective Tokens)', link: '/reference/effective-tokens-specification/' },
						{ label: 'Cost Management (Model Tables)', link: '/reference/model-tables/' },
						{ label: 'Ephemerals', link: '/reference/ephemerals/' },
						{ label: 'Environment Variables', link: '/reference/environment-variables/' },
						{ label: 'FAQ', link: '/reference/faq/' },
						{ label: 'Footers', link: '/reference/footers/' },
						{ label: 'Frontmatter', link: '/reference/frontmatter/' },
						{ label: 'Frontmatter (Hash Specification)', link: '/reference/frontmatter-hash-specification/' },
						{ label: 'Frontmatter (Schedule Syntax)', link: '/reference/schedule-syntax/' },
						{ label: 'Frontmatter (Fuzzy Schedules)', link: '/reference/fuzzy-schedule-specification/' },
						{ label: 'Frontmatter (Full)', link: '/reference/frontmatter-full/' },
						{ label: 'Frontmatter (Triggers)', link: '/reference/triggers/' },
						{ label: 'GH-AW Agent', link: '/reference/custom-agent-for-aw/' },
						{ label: 'GH-AW as MCP Server', link: '/reference/gh-aw-as-mcp-server/' },
						{ label: 'GitHub (Checkout)', link: '/reference/checkout/' },
						{ label: 'GitHub (Read Tools)', link: '/reference/github-tools/' },
						{ label: 'GitHub (Read Permissions)', link: '/reference/permissions/' },
						{ label: 'GitHub (Integrity Filtering)', link: '/reference/integrity/' },
						{ label: 'GitHub (Cross-Repository)', link: '/reference/cross-repository/' },
						{ label: 'GitHub (Fork Support)', link: '/reference/fork-support/' },
						{ label: 'Glossary', link: '/reference/glossary/' },
						{ label: 'Imports', link: '/reference/imports/' },
						{ label: 'Imports (APM)', link: '/reference/dependencies/' },
						{ label: 'Imports (Copilot Agent Files)', link: '/reference/copilot-custom-agents/' },
						{ label: 'Inline Sub-Agents', link: '/reference/inline-sub-agents/' },
						{ label: 'Imports (Dependabot)', link: '/reference/dependabot/' },
						{ label: 'Indexing (QMD)', link: '/reference/qmd/' },
						{ label: 'Markdown', link: '/reference/markdown/' },
						{ label: 'MCP Gateway', link: '/reference/mcp-gateway/' },
						{ label: 'Network Access', link: '/reference/network/' },
						{ label: 'Playwright', link: '/reference/playwright/' },
						{ label: 'Releases and Versioning', link: '/reference/releases/' },
						{ label: 'Repo Memory', link: '/reference/repo-memory/' },
						{ label: 'MCP Scripts', link: '/reference/mcp-scripts/' },
						{ label: 'MCP Scripts (Spec)', link: '/reference/mcp-scripts-specification/' },
						{ label: 'Safe Outputs', link: '/reference/safe-outputs/' },
						{ label: 'Safe Outputs (Custom)', link: '/reference/custom-safe-outputs/' },
						{ label: 'Safe Outputs (Pull Requests)', link: '/reference/safe-outputs-pull-requests/' },
						{ label: 'Safe Outputs (Spec)', link: '/reference/safe-outputs-specification/' },
						{ label: 'Safe Outputs (Staged Mode)', link: '/reference/staged-mode/' },
						{ label: 'Sandbox', link: '/reference/sandbox/' },
						{ label: 'Templating', link: '/reference/templating/' },
						{ label: 'Threat Detection', link: '/reference/threat-detection/' },
						{ label: 'Tools', link: '/reference/tools/' },
						{ label: 'Triggering CI', link: '/reference/triggering-ci/' },
						{ label: 'WASM Compilation', link: '/reference/wasm-compilation/' },
						{ label: 'Workflow Structure', link: '/reference/workflow-structure/' },
					],
				},
				{
					label: 'Troubleshooting',
					autogenerate: { directory: 'troubleshooting' },
				},
				{
					label: 'Experimental',
					items: [
						{ label: 'AWF Reflect Route', link: '/experimental/awf-reflect/' },
					{ label: 'CorrectionOps', link: '/experimental/correction-ops/' },
						{ label: 'Monitoring with Projects', link: '/experimental/monitoring-with-projects/' },
						{ label: 'TrialOps', link: '/experimental/trial-ops/' },
					],
				},
				{ label: 'Agent Factory', link: '/agent-factory-status/' },
				{ label: 'Editors', link: '/reference/editors/' },
			],
		}),
	],
});
