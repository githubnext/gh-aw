import type { Page } from '@playwright/test';
import { expect, test } from '@playwright/test';

async function getJsonLd(page: Page) {
  const jsonLd = page.locator('script[type="application/ld+json"]');
  await expect(jsonLd).toHaveCount(1);

  const raw = await jsonLd.first().textContent();
  if (!raw) {
    throw new Error('Expected JSON-LD script content');
  }

  return JSON.parse(raw);
}

test.describe('Schema JSON-LD', () => {
  test('keeps homepage website schema intact', async ({ page }) => {
    await page.goto('/gh-aw/');
    await page.waitForLoadState('networkidle');

    const schema = await getJsonLd(page);
    const graphTypes = schema['@graph'].map((item: { '@type': string }) => item['@type']);
    const website = schema['@graph'].find((item: { '@type': string }) => item['@type'] === 'WebSite');
    const organization = schema['@graph'].find((item: { '@type': string }) => item['@type'] === 'Organization');

    expect(graphTypes).toContain('WebSite');
    expect(graphTypes).toContain('Organization');
    expect(graphTypes).toContain('FAQPage');
    expect(website).toBeDefined();
    expect(organization).toBeDefined();
    expect(website.name).toBe('gh-aw — GitHub Agentic Workflows');
    expect(website.url).toBe('https://github.github.com/gh-aw/');
    expect(organization.name).toBe('GitHub');
    expect(organization.url).toBe('https://github.com/github/gh-aw');
    expect(organization.logo).toBe(
      'https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png'
    );
    expect(organization.sameAs).toEqual(
      expect.arrayContaining(['https://github.com/github/gh-aw'])
    );
  });

  test('adds BlogPosting schema to blog posts', async ({ page }) => {
    await page.goto('/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/');
    await page.waitForLoadState('networkidle');

    const schema = await getJsonLd(page);

    expect(schema['@type']).toBe('BlogPosting');
    expect(typeof schema.headline).toBe('string');
    expect(schema.headline.length).toBeGreaterThan(0);
    expect(typeof schema.description).toBe('string');
    expect(schema.description.length).toBeGreaterThan(0);
    expect(typeof schema.url).toBe('string');
    expect(schema.url).toContain(
      '/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/'
    );
    expect(schema.datePublished).toMatch(/^\d{4}-\d{2}-\d{2}T/);
    expect(schema.dateModified).toMatch(/^\d{4}-\d{2}-\d{2}T/);
    expect(schema.author).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ '@type': 'Person' }),
      ])
    );
    expect(schema.publisher).toEqual({
      '@id': 'https://github.github.com/gh-aw/#organization',
    });
  });

  test('adds TechArticle schema to inner docs pages', async ({ page }) => {
    await page.goto('/gh-aw/setup/quick-start/');
    await page.waitForLoadState('networkidle');

    const schema = await getJsonLd(page);

    expect(schema['@type']).toBe('TechArticle');
    expect(schema.headline).toBe('Quick Start');
    expect(schema.description).toContain('Get your first agentic workflow running in minutes.');
    expect(schema.url).toBe('https://github.github.com/gh-aw/setup/quick-start/');
    expect(schema.isPartOf).toEqual({
      '@id': 'https://github.github.com/gh-aw/#website',
    });
  });
});
