import type { APIRoute } from 'astro';
import { getAwPrompts } from './_aw-prompts.js';

const BASE_URL = 'https://github.github.com/gh-aw';

const DOCS_SECTIONS = [
	{
		heading: '## Introduction',
		links: [
			{ title: 'About Workflows', path: '/introduction/overview/', description: 'Understanding how GitHub Agentic Workflows transforms natural language into automated AI-powered workflows' },
			{ title: 'How They Work', path: '/introduction/how-they-work/', description: 'Understanding the core concepts and architecture, from compilation to execution' },
			{ title: 'Security Architecture', path: '/introduction/architecture/', description: 'Defense-in-depth mechanisms including sandboxing, firewall, and safe-outputs' },
		],
	},
	{
		heading: '## Getting Started',
		links: [
			{ title: 'Quick Start', path: '/setup/quick-start/', description: 'Get your first agentic workflow running in minutes' },
			{ title: 'Creating Workflows', path: '/setup/creating-workflows/', description: 'Author powerful automation workflows in natural language with interactive guidance' },
			{ title: 'CLI Commands', path: '/setup/cli/', description: 'Complete guide to all available CLI commands for managing agentic workflows' },
		],
	},
	{
		heading: '## Guides',
		links: [
			{ title: 'Editing Workflows', path: '/guides/editing-workflows/', description: 'When and how to edit workflows directly or recompile them' },
			{ title: 'Network Configuration', path: '/guides/network-configuration/', description: 'Configure network access for package registries, CDNs, and development tools' },
			{ title: 'Using at Scale', path: '/guides/using-at-scale/', description: 'Adopt, share, and govern workflows across teams and repositories' },
			{ title: 'Reusing Workflows', path: '/guides/reusing-workflows/', description: 'Share and reuse workflow definitions across repositories' },
			{ title: 'MCP Servers', path: '/guides/mcps/', description: 'Configure and use Model Context Protocol servers in your workflows' },
		],
	},
	{
		heading: '## Reference',
		links: [
			{ title: 'Authentication', path: '/reference/auth/', description: 'GitHub Actions secrets, GitHub tokens and GitHub Apps in gh-aw' },
			{ title: 'Tools', path: '/reference/tools/', description: 'Configure GitHub API tools, browser automation, and AI capabilities' },
			{ title: 'Compilation Process', path: '/reference/compilation-process/', description: 'How markdown workflows are compiled to GitHub Actions YAML' },
			{ title: 'Safe Outputs', path: '/reference/safe-outputs/', description: 'Produce structured outputs that pass through the network firewall safely' },
			{ title: 'Sandbox', path: '/reference/sandbox/', description: 'Isolated execution environment for agentic workflow steps' },
			{ title: 'Billing', path: '/reference/billing/', description: 'Understand and manage costs for running agentic workflows' },
		],
	},
];

export const GET: APIRoute = () => {
	const prompts = getAwPrompts();

	const lines: string[] = [
		'# GitHub Agentic Workflows',
		'',
		'> GitHub Agentic Workflows (gh-aw) is a GitHub CLI extension that compiles markdown-based workflow definitions into GitHub Actions YAML. It enables teams to build, run, and manage AI-powered agentic workflows directly on GitHub using natural language.',
		'',
		'> Documentation: https://github.github.com/gh-aw/',
		'',
	];

	for (const { heading, links } of DOCS_SECTIONS) {
		lines.push(heading, '');
		for (const { title, path, description } of links) {
			lines.push(`- [${title}](${BASE_URL}${path}): ${description}`);
		}
		lines.push('');
	}

	if (prompts.length > 0) {
		lines.push('## Agent Prompts', '');
		for (const { file, description, rawUrl } of prompts) {
			const label = file.replace(/\.md$/, '');
			lines.push(description ? `- [${label}](${rawUrl}): ${description}` : `- [${label}](${rawUrl})`);
		}
	}

	return new Response(lines.join('\n'), {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
