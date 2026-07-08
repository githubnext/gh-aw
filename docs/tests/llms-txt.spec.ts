import { test, expect } from '@playwright/test';

test.describe('llms.txt', () => {
  test('should expose docs-site sections and links for AI indexing', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    expect(response.ok()).toBeTruthy();

    const body = (await response.text()).replace(/\r\n/g, '\n');

    expect(body).toContain('# GitHub Agentic Workflows');
    expect(body).toContain('> Official documentation for GitHub Agentic Workflows (gh-aw)');
    expect(body).toContain(
      'The main homepage is available at: [GitHub Agentic Workflows](https://github.github.com/gh-aw/)'
    );
    expect(body).toContain('## Setup');
    expect(body).toContain('## Reference');
    expect(body).toContain('## Optional');
    expect(body).toContain('[Quick Start](https://github.github.com/gh-aw/setup/quick-start/)');
    expect(body).toContain('[Tools](https://github.github.com/gh-aw/reference/tools/)');
    expect(body).not.toContain('raw.githubusercontent.com/github/gh-aw/main/.github/aw/');
    expect(body.split(/\s+/).length).toBeGreaterThan(1000);
  });
});
