import { test, expect } from '@playwright/test';

test.describe('llms.txt', () => {
	test('should expose a docs index instead of the agent prompt corpus', async ({ request }) => {
		const response = await request.get('/gh-aw/llms.txt');
		expect(response.ok()).toBeTruthy();

		const body = (await response.text()).replace(/\r\n/g, '\n');

		expect(body).toContain('# GitHub Agentic Workflows Documentation');
		expect(body).toContain('## Sections');
		expect(body).toContain('### Setup');
		expect(body).toContain('[Quick Start](https://github.github.com/gh-aw/setup/quick-start/)');
		expect(body).toContain('[Frontmatter](https://github.github.com/gh-aw/reference/frontmatter/)');
		expect(body).not.toContain('## Agent Prompts');
		expect(body).not.toContain('/.github/aw/');
	});

	test('should expose the full docs corpus for published pages', async ({ request }) => {
		const response = await request.get('/gh-aw/llms-full.txt');
		expect(response.ok()).toBeTruthy();

		const body = (await response.text()).replace(/\r\n/g, '\n');

		expect(body).toContain('# GitHub Agentic Workflows Documentation — Full Corpus');
		expect(body).toContain('URL: https://github.github.com/gh-aw/setup/quick-start/');
		expect(body).toContain('Description: Get your first agentic workflow running in minutes.');
		expect(body).toContain('## Quick Start');
		expect(body).toContain('## Frontmatter');
		expect(body).not.toContain('<!-- file: agentic-chat.md -->');
	});
});
