import { test, expect } from '@playwright/test';

test.describe('Mobile and Responsive Layout', () => {
  const formFactors = [
    { name: 'iPhone 16 (Mobile)', width: 393, height: 852 },
    { name: 'iPad (768px)', width: 768, height: 1024 },
    { name: 'iPad Pro 11 (834px)', width: 834, height: 1194 },
    { name: 'iPad Landscape (1024px)', width: 1024, height: 768 },
    { name: 'Desktop Portrait', width: 1080, height: 1920 },
    { name: 'Desktop Landscape', width: 1920, height: 1080 },
  ];

  const pages = [
    { url: '/gh-aw/', name: 'home page' },
    { url: '/gh-aw/introduction/overview/', name: 'content page' },
  ];

  test('should include markdown table data-label attributes without JavaScript', async ({ browser }) => {
    const context = await browser.newContext({
      javaScriptEnabled: false,
      viewport: { width: 393, height: 852 },
    });
    const page = await context.newPage();

    await page.goto('/gh-aw/reference/engines/');
    await page.waitForLoadState('domcontentloaded');

    const firstTableCell = page.locator('.sl-markdown-content table tbody td').first();
    await expect(firstTableCell).toBeVisible();
    await expect(firstTableCell).toHaveAttribute('data-label', 'Engine');

    await context.close();
  });

  for (const formFactor of formFactors) {
    test.describe(`${formFactor.name}`, () => {
      test.beforeEach(async ({ page }) => {
        await page.setViewportSize({ 
          width: formFactor.width, 
          height: formFactor.height 
        });
      });

      for (const testPage of pages) {
        test(`should render ${testPage.name} correctly`, async ({ page }) => {
          await page.goto(testPage.url);
          await page.waitForLoadState('networkidle');

          // Verify page loads
          await expect(page).toHaveTitle(/GitHub Agentic Workflows/);

          // Verify header is visible
          const header = page.locator('header');
          await expect(header).toBeVisible();

          // Verify main content is visible
          const main = page.locator('main');
          await expect(main).toBeVisible();

          // Check for horizontal scrollbar (should not exist)
          const bodyScrollWidth = await page.evaluate(() => document.body.scrollWidth);
          const bodyClientWidth = await page.evaluate(() => document.body.clientWidth);
          expect(bodyScrollWidth).toBeLessThanOrEqual(bodyClientWidth + 1); // Allow 1px tolerance
        });
      }

      test('should have proper content spacing on mobile', async ({ page }) => {
        if (formFactor.width < 768) {
          await page.goto('/gh-aw/introduction/overview/');
          await page.waitForLoadState('networkidle');

          // Content should have proper padding
          const contentPanel = page.locator('.content-panel').first();
          await expect(contentPanel).toBeVisible();

          // Sidebar should be hidden on mobile (below 768px)
          const sidebar = page.locator('.sidebar');
          await expect(sidebar).not.toBeVisible();
        }
      });

      test('should show persistent sidebar on tablet (WCAG W2)', async ({ page }) => {
        if (formFactor.width >= 768) {
          await page.goto('/gh-aw/introduction/overview/');
          await page.waitForLoadState('networkidle');

          // Sidebar should be persistently visible on tablet and desktop (768px+)
          const sidebar = page.locator('.sidebar');
          await expect(sidebar).toBeVisible();
        }
      });
    });
  }
});
