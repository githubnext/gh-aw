import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'GitHub Agentic Workflows',
			description: 'Write agentic workflows in natural language using markdown files, and run them as GitHub Actions workflows.',
			social: {
				github: 'https://github.com/githubnext/gh-aw',
			},
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'index' },
						{ label: 'Quick Start', slug: 'quick-start' },
						{ label: 'Commands', slug: 'commands' },
					],
				},
				{
					label: 'Core Concepts',
					items: [
						{ label: 'Concepts', slug: 'concepts' },
						{ label: 'Workflow Structure', slug: 'workflow-structure' },
						{ label: 'Frontmatter', slug: 'frontmatter' },
						{ label: 'Command Triggers', slug: 'command-triggers' },
						{ label: 'Alias Triggers', slug: 'alias-triggers' },
					],
				},
				{
					label: 'Advanced Features',
					items: [
						{ label: 'Include Directives', slug: 'include-directives' },
						{ label: 'Tools', slug: 'tools' },
						{ label: 'MCPs', slug: 'mcps' },
						{ label: 'Safe Outputs', slug: 'safe-outputs' },
					],
				},
				{
					label: 'Security & Configuration',
					items: [
						{ label: 'Secrets', slug: 'secrets' },
						{ label: 'Security Notes', slug: 'security-notes' },
					],
				},
				{
					label: 'Development',
					items: [
						{ label: 'VS Code Extension', slug: 'vscode' },
						{ label: 'Samples', slug: 'samples' },
					],
				},
			],
			customCss: [
				// Relative to the root of the project
				'./src/styles/custom.css',
			],
		}),
	],
});