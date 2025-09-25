// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import starlightLinksValidator from 'starlight-links-validator';
import starlightGitHubAlerts from 'starlight-github-alerts';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
// import starlightChangelogs, { makeChangelogsSidebarLinks } from 'starlight-changelogs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Load the custom TextMate grammar for agentic workflows
const agenticWorkflowGrammar = JSON.parse(
	readFileSync(join(__dirname, '../grammars/agentic-workflow.tmLanguage.json'), 'utf-8')
);

// https://astro.build/config
export default defineConfig({
	site: 'https://githubnext.github.io/gh-aw/',
	base: '/gh-aw/',
	integrations: [
		starlight({
			title: 'GitHub Agentic Workflows',
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/githubnext/gh-aw' },
				{ icon: 'rocket', label: 'Instructions', href: 'https://raw.githubusercontent.com/githubnext/gh-aw/main/pkg/cli/templates/instructions.md' }
			],
			expressiveCode: {
				shiki: {
						langs: /** @type {any[]} */ ([
							"markdown",
							"yaml",
							agenticWorkflowGrammar
						]),
						langAlias: { aw: "agentic-workflow" }
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
					label: 'Tools',
					autogenerate: { directory: 'tools' },
				},
				{
					label: 'Guides',
					autogenerate: { directory: 'guides' },
				},
				{
					label: 'Application Areas',
					autogenerate: { directory: 'samples' },
				},
				// ...makeChangelogsSidebarLinks([
				// 	{ type: 'all', base: 'changelog', label: 'Changelog' }
				// ]),
			],
		}),
	],
});
