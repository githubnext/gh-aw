import { test, expect } from '@playwright/test';

test.describe('Homepage Public Preview callout', () => {
  test('shows a caution-style notice above the fold', async ({ page }) => {
    await page.goto('/gh-aw/');
    await page.waitForLoadState('networkidle');

    const callout = page
      .locator('.hero-preview-callout')
      .filter({ hasText: 'GitHub Agentic Workflows is in Public Preview and may change significantly.' });

    await expect(callout).toBeVisible();

    const borderColor = await callout.evaluate((el) => getComputedStyle(el).borderLeftColor);
    expect(borderColor).toBe('rgb(219, 168, 43)');

    const box = await callout.boundingBox();
    const viewport = page.viewportSize();
    expect(box).not.toBeNull();
    expect(viewport).not.toBeNull();

    if (box && viewport) {
      expect(box.y).toBeGreaterThanOrEqual(0);
      expect(box.y + box.height).toBeLessThanOrEqual(viewport.height);
    }
  });
});
