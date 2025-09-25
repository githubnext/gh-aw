// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import starlightLinksValidator from 'starlight-links-validator';
import starlightGitHubAlerts from 'starlight-github-alerts';
// import starlightChangelogs, { makeChangelogsSidebarLinks } from 'starlight-changelogs';

// Define custom language for agentic workflows (frontmatter + markdown)
const awLanguageDefinition = {
	id: "aw",
	scopeName: "source.aw",
	aliases: ["agentic-workflow"],
	grammar: {
		name: "Agentic Workflow",
		scopeName: "source.aw",
		fileTypes: ["aw"],
		patterns: [
			{
				// Match YAML frontmatter block
				name: "meta.frontmatter.yaml",
				begin: "\\A(---)\\s*$",
				end: "^(---)\\s*$",
				beginCaptures: {
					"1": { name: "punctuation.definition.tag.begin.yaml" }
				},
				endCaptures: {
					"1": { name: "punctuation.definition.tag.end.yaml" }
				},
				contentName: "source.yaml",
				patterns: [
					{ include: "source.yaml" }
				]
			},
			{
				// Match everything after frontmatter as markdown
				name: "text.html.markdown",
				begin: "(?<=^---\\s*$\\s)",
				end: "\\z",
				patterns: [
					{ include: "text.html.markdown" }
				]
			}
		]
	}
};

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
					langs: [
						"markdown",
						"yaml",
						awLanguageDefinition
					],
				},
			},
			plugins: [
				// starlightChangelogs(), // Temporarily disabled for testing
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
