import { test, expect } from '@playwright/test';

const BASE_URL = 'https://github.github.com/gh-aw';

test.describe('llms.txt', () => {
  test('should be served as plain text', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    expect(response.ok()).toBeTruthy();
    expect(response.headers()['content-type']).toContain('text/plain');
  });

  test('should contain H1 project name', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    const body = await response.text();
    expect(body).toContain('# GitHub Agentic Workflows');
  });

  test('should contain project description', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    const body = await response.text();
    expect(body).toContain('> GitHub Agentic Workflows (gh-aw)');
  });

  test('should contain links to key documentation pages', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    const body = await response.text();

    const expectedLinks = [
      `${BASE_URL}/introduction/overview/`,
      `${BASE_URL}/introduction/how-they-work/`,
      `${BASE_URL}/setup/quick-start/`,
      `${BASE_URL}/setup/creating-workflows/`,
      `${BASE_URL}/setup/cli/`,
      `${BASE_URL}/guides/editing-workflows/`,
      `${BASE_URL}/reference/auth/`,
      `${BASE_URL}/reference/tools/`,
    ];

    for (const link of expectedLinks) {
      expect(body).toContain(link);
    }
  });

  test('should contain Introduction, Getting Started, Guides, and Reference sections', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    const body = await response.text();

    expect(body).toContain('## Introduction');
    expect(body).toContain('## Getting Started');
    expect(body).toContain('## Guides');
    expect(body).toContain('## Reference');
  });
});
