import { test, expect } from '@playwright/test';

test.describe('Quick Start frontmatter guidance', () => {
  test('introduces frontmatter before optional Step 4 customization', async ({ page }) => {
    await page.goto('/gh-aw/setup/quick-start/');
    await page.waitForLoadState('networkidle');

    const frontmatterLink = page.locator('a[href="/gh-aw/reference/frontmatter/"]').first();
    await expect(frontmatterLink).toBeVisible();

    const positions = await page.evaluate(() => {
      const content = (document.querySelector('main') ?? document.body).textContent ?? '';
      return {
        frontmatterIndex: content.toLowerCase().indexOf('frontmatter'),
        step4Index: content.indexOf('Step 4 - Customize your workflow'),
      };
    });

    expect(positions.frontmatterIndex).toBeGreaterThan(-1);
    expect(positions.step4Index).toBeGreaterThan(positions.frontmatterIndex);
  });
});
