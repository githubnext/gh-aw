import { test, expect } from '@playwright/test';

const EXPECTED_ROBOTS_TXT = [
  'User-agent: *',
  'Allow: /',
  '',
  'User-agent: GPTBot',
  'Allow: /',
  '',
  'User-agent: OAI-SearchBot',
  'Allow: /',
  '',
  'User-agent: ChatGPT-User',
  'Allow: /',
  '',
  'User-agent: anthropic-ai',
  'Allow: /',
  '',
  'User-agent: ClaudeBot',
  'Allow: /',
  '',
  'User-agent: Claude-SearchBot',
  'Allow: /',
  '',
  'User-agent: PerplexityBot',
  'Allow: /',
  '',
  'User-agent: Googlebot',
  'Allow: /',
  '',
  'User-agent: Perplexity-User',
  'Allow: /',
  '',
  'User-agent: Google-Extended',
  'Allow: /',
  '',
  'User-agent: Google-CloudVertexBot',
  'Allow: /',
  '',
  'User-agent: Bingbot',
  'Allow: /',
  '',
  'User-agent: cohere-ai',
  'Allow: /',
  '',
  'User-agent: DuckAssistBot',
  'Allow: /',
  '',
  'User-agent: xAI-Bot',
  'Allow: /',
  '',
  'User-agent: Amazonbot',
  'Allow: /',
  '',
  'User-agent: AI2Bot',
  'Allow: /',
  '',
  'User-agent: YouBot',
  'Allow: /',
  '',
  'User-agent: CCBot',
  'Allow: /',
  '',
  'Sitemap: https://github.github.com/gh-aw/sitemap-index.xml',
  '',
].join('\n');

test.describe('robots.txt', () => {
  test('should contain only the expected AI crawler directives and sitemap index', async ({ request }) => {
    const response = await request.get('/gh-aw/robots.txt');
    expect(response.ok()).toBeTruthy();

    const body = (await response.text()).replace(/\r\n/g, '\n');

    expect(body).toBe(EXPECTED_ROBOTS_TXT);
  });
});
