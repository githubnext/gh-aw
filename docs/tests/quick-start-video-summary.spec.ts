import { test, expect } from '@playwright/test';

test.describe('Quick Start video summary', () => {
  test('should provide a text summary for the quick start demo video', async ({ page }) => {
    await page.goto('/gh-aw/setup/quick-start/');
    await page.waitForLoadState('networkidle');

    const quickStartSummary = page.locator('details').filter({
      has: page.getByText('Prefer text? Read the Quick Start demo summary'),
    });

    await expect(quickStartSummary).toBeVisible();
    await expect(quickStartSummary).toContainText('gh extension install github/gh-aw');
    await expect(quickStartSummary).toContainText(
      'gh aw add-wizard githubnext/agentics/daily-repo-status'
    );
    await expect(quickStartSummary).toContainText(
      'check Issues for the generated daily report'
    );
  });
});
