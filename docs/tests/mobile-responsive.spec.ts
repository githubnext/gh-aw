import { test, expect } from '@playwright/test';

test.describe('Mobile and Responsive Layout', () => {
  const formFactors = [
    { name: '360px Mobile', width: 360, height: 800 },
    { name: 'iPhone 16 (Mobile)', width: 393, height: 852 },
    { name: '412px Mobile', width: 412, height: 915 },
    { name: '428px Mobile', width: 428, height: 926 },
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

  test('should wrap markdown tables in a scroll wrapper without JavaScript', async ({ browser }) => {
    const context = await browser.newContext({
      javaScriptEnabled: false,
      viewport: { width: 768, height: 1024 },
    });
    const page = await context.newPage();

    await page.goto('/gh-aw/reference/engines/');
    await page.waitForLoadState('domcontentloaded');

    // The rehype plugin should have added the wrapper div at build time
    const wrapper = page.locator('.sl-markdown-content .table-scroll-wrapper').first();
    await expect(wrapper).toBeVisible();

    // The table must be a direct child of the wrapper
    const tableInWrapper = page.locator('.sl-markdown-content .table-scroll-wrapper > table').first();
    await expect(tableInWrapper).toBeVisible();

    await context.close();
  });

  test('should wrap ALL markdown tables in a scroll wrapper on the engines reference page', async ({ browser }) => {
    const context = await browser.newContext({
      javaScriptEnabled: false,
      viewport: { width: 768, height: 1024 },
    });
    const page = await context.newPage();

    await page.goto('/gh-aw/reference/engines/');
    await page.waitForLoadState('domcontentloaded');

    // Count all tables in markdown content area
    const tableCount = await page.locator('.sl-markdown-content table').count();
    expect(tableCount).toBeGreaterThan(0);

    // Count tables that are direct children of .table-scroll-wrapper
    const wrappedTableCount = await page.locator('.sl-markdown-content .table-scroll-wrapper > table').count();

    // Every table must have a scroll wrapper for consistent horizontal scrolling on all viewports
    expect(wrappedTableCount).toBe(tableCount);

    await context.close();
  });

  test('should have WCAG 2.5.5-compliant touch target size for mobile table cells', async ({ browser }) => {
    const context = await browser.newContext({
      javaScriptEnabled: true,
      viewport: { width: 390, height: 844 },
    });
    const page = await context.newPage();

    await page.goto('/gh-aw/reference/engines/');
    await page.waitForLoadState('networkidle');

    // On mobile (<=640px), table cells are rendered as stacked cards.
    // Each cell must meet the WCAG 2.5.5 AAA minimum touch target of 44 px (2.75 rem).
    const tdMinHeight = await page.evaluate(() => {
      const td = document.querySelector('.sl-markdown-content table tbody td');
      if (!td) return 0;
      return parseFloat(getComputedStyle(td).minHeight);
    });

    expect(tdMinHeight).toBeGreaterThanOrEqual(44);

    await context.close();
  });

  test('should expose a functional home page skip link target', async ({ page }) => {
    await page.setViewportSize({ width: 360, height: 800 });
    await page.goto('/gh-aw/');
    await page.waitForLoadState('networkidle');

    const skipLink = page.locator('a[href="#starlight__main"]');
    await expect(skipLink).toHaveCount(1);
    await expect(page.locator('main#starlight__main')).toBeVisible();
    await expect(page.locator('#main-content')).toHaveCount(1);
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
          const scrollMetrics = await page.evaluate(() => ({
            scrollWidth: document.scrollingElement?.scrollWidth ?? document.documentElement.scrollWidth,
            clientWidth: document.scrollingElement?.clientWidth ?? document.documentElement.clientWidth,
          }));
          expect(scrollMetrics.scrollWidth).toBeLessThanOrEqual(scrollMetrics.clientWidth + 1); // Allow 1px tolerance
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

  // Regression test for https://github.com/github/gh-aw/issues/29545
  // Verify the navigation dropdown is fully within the viewport when large
  // user fonts cause header elements to shift on Android Chrome.
  test('hamburger dropdown stays within viewport with large user fonts', async ({ browser }) => {
    const VIEWPORT_WIDTH = 393;
    const context = await browser.newContext({
      // Simulate Android Chrome with the user's accessibility font size set to
      // "Large" — typically 1.3× the default, so override the page root font-size.
      viewport: { width: VIEWPORT_WIDTH, height: 852 },
      javaScriptEnabled: true,
    });
    const page = await context.newPage();

    await page.goto('/gh-aw/introduction/overview/');
    await page.waitForLoadState('networkidle');

    // Simulate large OS-level font scaling by overriding the root font size.
    // Done after navigation so the document exists and the style tag can attach.
    await page.addStyleTag({ content: 'html { font-size: 20px !important; }' });

    // The hamburger wrapper should be visible on a narrow mobile viewport.
    const hamburgerBtn = page.locator('.hamburger-btn');
    await expect(hamburgerBtn).toBeVisible();

    // Click the hamburger to open the dropdown.
    await hamburgerBtn.click();

    const dropdown = page.locator('.tablet-dropdown');
    await expect(dropdown).toBeVisible();

    // The dropdown must be fully within the viewport horizontally.
    const dropdownBox = await dropdown.boundingBox();
    expect(dropdownBox).not.toBeNull();
    if (dropdownBox) {
      expect(dropdownBox.x).toBeGreaterThanOrEqual(0);
      expect(dropdownBox.x + dropdownBox.width).toBeLessThanOrEqual(VIEWPORT_WIDTH + 1); // 1px tolerance
    }

    await context.close();
  });

  // Verify mobile navigation toggle: hamburger menu nav links become visible on narrow viewports.
  // Addresses the manual verification recommendation from the 2026-06-24 multi-device docs test report.
  test('hamburger menu toggles navigation visibility on mobile viewport', async ({ browser }) => {
    const context = await browser.newContext({
      viewport: { width: 390, height: 844 },
      javaScriptEnabled: true,
    });
    const page = await context.newPage();

    await page.goto('/gh-aw/introduction/overview/');
    await page.waitForLoadState('networkidle');

    // The hamburger button must be present and focusable on a narrow mobile viewport.
    const hamburgerBtn = page.locator('.hamburger-btn');
    await expect(hamburgerBtn).toBeVisible();
    await expect(hamburgerBtn).toHaveAttribute('aria-expanded', 'false');

    // The dropdown must be hidden before the button is clicked.
    const dropdown = page.locator('.tablet-dropdown');
    await expect(dropdown).toBeHidden();

    // Click the button; the dropdown must become visible and contain nav links.
    await hamburgerBtn.click();
    await expect(hamburgerBtn).toHaveAttribute('aria-expanded', 'true');
    await expect(dropdown).toBeVisible();

    const navLinks = dropdown.locator('.dropdown-link');
    const linkCount = await navLinks.count();
    expect(linkCount).toBeGreaterThan(0);
    for (const link of await navLinks.all()) {
      await expect(link).toBeVisible();
    }

    // A second click must close the dropdown.
    await hamburgerBtn.click();
    await expect(hamburgerBtn).toHaveAttribute('aria-expanded', 'false');
    await expect(dropdown).toBeHidden();

    await context.close();
  });
});
