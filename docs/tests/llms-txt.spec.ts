import { test, expect } from '@playwright/test';

test.describe('llms.txt', () => {
  test('should be published with llms.txt core sections', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    expect(response.ok()).toBeTruthy();

    const body = (await response.text()).replace(/\r\n/g, '\n');

    expect(body).toContain('# GitHub Agentic Workflows');
    expect(body).toContain('> Agent-optimised prompt files for GitHub Agentic Workflows (gh-aw).');
    expect(body).toContain('## Agent Prompts');
    expect(body).toMatch(/- \[[^\]]+\]\(https:\/\/raw\.githubusercontent\.com\/github\/gh-aw\/main\/\.github\/aw\/[^)]+\)/);
  });
});
