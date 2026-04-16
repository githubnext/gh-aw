import { test, expect } from '@playwright/test';

test.describe('Homepage Links', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/gh-aw/');
    await page.waitForLoadState('networkidle');
  });

  test('should have correct Getting Started button link', async ({ page }) => {
    // Locate the Getting Started button
    const gettingStartedButton = page.locator('a.primary:has-text("Getting Started")');
    
    // Verify button exists and is visible
    await expect(gettingStartedButton).toBeVisible();
    
    // Verify the href includes the base path
    const href = await gettingStartedButton.getAttribute('href');
    expect(href).toBe('/gh-aw/setup/quick-start/');
  });

  test('should navigate to quick start page when Getting Started is clicked', async ({ page }) => {
    // Click the Getting Started button
    const gettingStartedButton = page.locator('a.primary:has-text("Getting Started")');
    await gettingStartedButton.click();
    
    // Wait for navigation
    await page.waitForLoadState('networkidle');
    
    // Verify we're on the quick start page
    await expect(page).toHaveURL(/\/gh-aw\/setup\/quick-start\//);
    await expect(page).toHaveTitle(/Quick Start/);
  });

  test('should provide descriptive title attributes on homepage videos', async ({ page }) => {
    const videos = page.locator('video.gh-aw-video-element');
    await expect(videos).toHaveCount(2);

    await expect(videos.nth(0)).toHaveAttribute(
      'title',
      'Install and add workflow in CLI demo video'
    );
    await expect(videos.nth(1)).toHaveAttribute(
      'title',
      'Create workflow on GitHub demo video'
    );
  });
});
