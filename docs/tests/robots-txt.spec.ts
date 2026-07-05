import { test, expect } from '@playwright/test';

test.describe('robots.txt', () => {
	test('should expose the docs crawler policy and sitemap index', async ({ request }) => {
		const response = await request.get('/gh-aw/robots.txt');
		expect(response.ok()).toBeTruthy();

		const body = await response.text();

		expect(body).toContain('User-agent: *\nAllow: /');
		expect(body).toContain('User-agent: GPTBot\nAllow: /');
		expect(body).toContain('User-agent: OAI-SearchBot\nAllow: /');
		expect(body).toContain('User-agent: ChatGPT-User\nAllow: /');
		expect(body).toContain('User-agent: anthropic-ai\nAllow: /');
		expect(body).toContain('User-agent: ClaudeBot\nAllow: /');
		expect(body).toContain('User-agent: PerplexityBot\nAllow: /');
		expect(body).toContain('User-agent: Perplexity-User\nAllow: /');
		expect(body).toContain('User-agent: Google-Extended\nAllow: /');
		expect(body).toContain('User-agent: Google-CloudVertexBot\nAllow: /');
		expect(body).toContain('Sitemap: https://github.github.com/gh-aw/sitemap-index.xml');
	});
});
