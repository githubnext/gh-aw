// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import starlightLinksValidator from 'starlight-links-validator';
import starlightGitHubAlerts from 'starlight-github-alerts';
// import starlightChangelogs, { makeChangelogsSidebarLinks } from 'starlight-changelogs';

// NOTE: A previous attempt defined a custom Shiki grammar for `aw` (agentic workflow) but
// Shiki did not register it and builds produced a warning: language "aw" not found.
// For now we alias `aw` -> `markdown` which removes the warning and still gives
// reasonable highlighting for examples that combine frontmatter + markdown.
// If richer highlighting is needed later, implement a proper TextMate grammar
// in a separate JSON file and load it here (ensure required embedded scopes exist).

// https://astro.build/config
export default defineConfig({
	site: 'https://githubnext.github.io/gh-aw/',
	base: '/gh-aw/',
	devToolbar: {
		enabled: false
	},
	integrations: [
		starlight({
			title: 'GitHub Agentic Workflows',
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/githubnext/gh-aw' },
				{ icon: 'rocket', label: 'Instructions', href: 'https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/instructions/github-agentic-workflows.instructions.md' }
			],
			expressiveCode: {
				shiki: {
						langs: /** @type {any[]} */ ([
							"markdown",
							"yaml"
						]),
						langAlias: { aw: "markdown" }
				},
			},
			plugins: [
				// starlightChangelogs(),
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
					label: 'Start Here',
					autogenerate: { directory: 'start-here' },
				},
				{
					label: 'Workflows',
					autogenerate: { directory: 'reference' },
				},
				{
					label: 'Guides',
					autogenerate: { directory: 'guides' },
				},
				{
					label: 'Application Areas',
					autogenerate: { directory: 'samples' },
				},
				{
					label: 'Tools',
					autogenerate: { directory: 'tools' },
				},
				{
					label: 'Troubleshooting',
					autogenerate: { directory: 'troubleshooting' },
				},
				// ...makeChangelogsSidebarLinks([
				// 	{ type: 'all', base: 'changelog', label: 'Changelog' }
				// ]),
				{
					label: 'Status',
					link: '/status/',
				},
			],
		}),
	],
});
