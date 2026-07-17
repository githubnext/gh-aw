import { test, expect } from '@playwright/test';

test.describe('llms.txt', () => {
  test('should be accessible and contain expected documentation sections', async ({ request }) => {
    const response = await request.get('/gh-aw/llms.txt');
    expect(response.ok()).toBeTruthy();
    expect(response.headers()['content-type']).toContain('text/plain');

    const body = (await response.text()).replace(/\r\n/g, '\n');

    // Should start with the project title heading.
    expect(body).toContain('# GitHub Agentic Workflows');

    // Should contain a project description blockquote.
    expect(body).toMatch(/^> .+/m);

    // Should include at least one documentation section.
    expect(body).toMatch(/^## (Introduction|Setup|Design Patterns|Guides|Reference|Experimental|Specifications|Troubleshooting)$/m);

    // Should contain documentation page links pointing to the live docs site.
    expect(body).toContain('https://github.github.com/gh-aw/');

    // Each link line should follow the llms.txt link format: - [Title](url)
    const linkLines = body.split('\n').filter((l) => l.startsWith('- ['));
    expect(linkLines.length).toBeGreaterThan(0);
    for (const line of linkLines) {
      expect(line).toMatch(/^- \[.+\]\(https?:\/\/.+\)/);
    }
  });
});
