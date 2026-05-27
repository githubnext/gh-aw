import { test, expect } from '@playwright/test';

test.describe('Quick Start frontmatter guidance', () => {
  test('introduces frontmatter before optional Step 4 customization', async ({ page }) => {
    await page.goto('/gh-aw/setup/quick-start/');
    await page.waitForLoadState('networkidle');

    const frontmatterLink = page.locator('a[href="/gh-aw/reference/frontmatter/"]').first();
    await expect(frontmatterLink).toBeVisible();

    const appearsBeforeStep4 = await page.evaluate(() => {
      const frontmatter = document.querySelector('a[href="/gh-aw/reference/frontmatter/"]');
      const step4Heading = Array.from(document.querySelectorAll('h3')).find((heading) =>
        heading.textContent?.includes('Step 4 - Customize your workflow'),
      );

      if (!frontmatter || !step4Heading) {
        return false;
      }

      return Boolean(frontmatter.compareDocumentPosition(step4Heading) & Node.DOCUMENT_POSITION_FOLLOWING);
    });

    expect(appearsBeforeStep4).toBe(true);
  });
});
