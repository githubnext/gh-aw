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

    expect(graphTypes).toContain('WebSite');
    expect(graphTypes).toContain('FAQPage');
  });

  test('adds BlogPosting schema to blog posts', async ({ page }) => {
    await page.goto('/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/');
    await page.waitForLoadState('networkidle');

    const schema = await getJsonLd(page);

    expect(schema['@type']).toBe('BlogPosting');
    expect(schema.headline).toBe('Meet the Workflows: Testing & Validation');
    expect(schema.description).toBe(
      'A curated tour of testing workflows that keep everything running smoothly'
    );
    expect(schema.url).toBe(
      'https://github.github.com/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/'
    );
    expect(schema.datePublished).toBe('2026-01-13T11:00:00.000Z');
    expect(schema.dateModified).toBe('2026-01-13T11:00:00.000Z');
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
    expect(schema.name).toBe('Quick Start');
    expect(schema.description).toContain('Get your first agentic workflow running in minutes.');
    expect(schema.url).toBe('https://github.github.com/gh-aw/setup/quick-start/');
    expect(schema.isPartOf).toEqual({
      '@id': 'https://github.github.com/gh-aw/#website',
    });
  });
});
