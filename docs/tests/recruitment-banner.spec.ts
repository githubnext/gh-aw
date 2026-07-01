import { test, expect } from '@playwright/test';

test.describe('Recruitment banner', () => {
  test('is not rendered when feature is disabled by config', async ({ page }) => {
    await page.goto('/gh-aw/?recruit=gh-aw-docs-research-2026q3&uid=12345');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-recruitment-banner]')).toHaveCount(0);
  });
});
