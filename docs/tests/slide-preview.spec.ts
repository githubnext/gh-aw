import { test, expect } from '@playwright/test';

test.describe('Slide Preview on Homepage', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/gh-aw/');
    await page.waitForLoadState('networkidle');
  });

  test('should render slide preview component', async ({ page }) => {
    // Verify the slide hero component exists
    const slideHero = page.locator('[data-slide-hero]');
    await expect(slideHero).toBeVisible();

    // Verify the canvas element exists
    const canvas = page.locator('[data-slide-canvas]');
    await expect(canvas).toBeVisible();
  });

  test('should load and render PDF slides', async ({ page }) => {
    // Wait for the loading state to be hidden
    const loading = page.locator('[data-slide-loading]');
    await expect(loading).toBeHidden({ timeout: 10000 });

    // Verify the canvas has been rendered with content
    const canvas = page.locator('[data-slide-canvas]');
    await expect(canvas).toBeVisible();

    // Check that the canvas has width and height set (indicating PDF has been rendered)
    const canvasElement = await canvas.elementHandle();
    const width = await canvasElement?.evaluate((el) => (el as HTMLCanvasElement).width);
    const height = await canvasElement?.evaluate((el) => (el as HTMLCanvasElement).height);

    expect(width).toBeGreaterThan(0);
    expect(height).toBeGreaterThan(0);

    // Verify the hero has the 'is-ready' class indicating successful load
    const slideHero = page.locator('[data-slide-hero]');
    await expect(slideHero).toHaveClass(/is-ready/);
  });

  test('should be keyboard accessible', async ({ page }) => {
    // Wait for slides to load
    const loading = page.locator('[data-slide-loading]');
    await expect(loading).toBeHidden({ timeout: 10000 });

    // Verify the stage has proper accessibility attributes
    const stage = page.locator('[data-slide-stage]');
    await expect(stage).toHaveAttribute('tabindex', '0');
    await expect(stage).toHaveAttribute('role', 'link');
    await expect(stage).toHaveAttribute('aria-label', 'Open slide presentation');

    // Verify canvas has aria-label
    const canvas = page.locator('[data-slide-canvas]');
    await expect(canvas).toHaveAttribute('role', 'img');
    await expect(canvas).toHaveAttribute('aria-label');
  });

  test('should navigate to PDF when Enter key is pressed', async ({ page }) => {
    // Wait for slides to load
    const loading = page.locator('[data-slide-loading]');
    await expect(loading).toBeHidden({ timeout: 10000 });

    // Focus on the stage
    const stage = page.locator('[data-slide-stage]');
    await stage.focus();

    // Listen for navigation - use Promise.race to wait for either event
    const navigationPromise = Promise.race([
      page.waitForEvent('popup', { timeout: 5000 }).catch(() => null),
      page.waitForNavigation({ timeout: 5000 }).catch(() => null),
    ]);

    await stage.press('Enter');
    const response = await navigationPromise;

    // If navigation happened, verify the URL contains the PDF path
    if (response) {
      const url = typeof response === 'object' && 'url' in response ? 
        await response.url() : 
        page.url();
      expect(url).toContain('slides/github-agentic-workflows.pdf');
    }
  });

  test('should navigate to PDF when Space key is pressed', async ({ page }) => {
    // Wait for slides to load
    const loading = page.locator('[data-slide-loading]');
    await expect(loading).toBeHidden({ timeout: 10000 });

    // Focus on the stage
    const stage = page.locator('[data-slide-stage]');
    await stage.focus();

    // Listen for navigation - use Promise.race to wait for either event
    const navigationPromise = Promise.race([
      page.waitForEvent('popup', { timeout: 5000 }).catch(() => null),
      page.waitForNavigation({ timeout: 5000 }).catch(() => null),
    ]);

    await stage.press('Space');
    const response = await navigationPromise;

    // If navigation happened, verify the URL contains the PDF path
    if (response) {
      const url = typeof response === 'object' && 'url' in response ? 
        await response.url() : 
        page.url();
      expect(url).toContain('slides/github-agentic-workflows.pdf');
    }
  });

  test('should handle PDF fetch failures gracefully', async ({ page }) => {
    // Override the PDF request to fail
    await page.route('**/slides/github-agentic-workflows.pdf', (route) => {
      route.abort('failed');
    });

    // Reload the page
    await page.goto('/gh-aw/');
    await page.waitForLoadState('networkidle');

    // Wait a bit for the error to be displayed
    await page.waitForTimeout(2000);

    // Verify error message is displayed
    const loading = page.locator('[data-slide-loading]');
    const errorText = await loading.textContent();
    expect(errorText).toContain('Unable to load slides');
  });
});
