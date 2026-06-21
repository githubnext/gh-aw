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

  test('should use descriptive fallback links and captions tracks on homepage videos', async ({ page }) => {
    const fallbackLinks = page.locator('video.gh-aw-video-element p a');
    await expect(fallbackLinks).toHaveCount(2);

    await expect(fallbackLinks.nth(0)).toHaveText('Download Install and add workflow in CLI demo video');
    await expect(fallbackLinks.nth(1)).toHaveText('Download Create workflow on GitHub demo video');

    const captionTracks = page.locator('video.gh-aw-video-element track[kind="captions"]');
    await expect(captionTracks).toHaveCount(2);

    await expect(captionTracks.nth(0)).toHaveAttribute(
      'src',
      '/gh-aw/videos/install-and-add-workflow-in-cli.vtt'
    );
    await expect(captionTracks.nth(1)).toHaveAttribute(
      'src',
      '/gh-aw/videos/create-workflow-on-github.vtt'
    );
  });

  test('should expose VideoObject schema for homepage videos', async ({ page }) => {
    const videoSchemas = await page.locator('script[type="application/ld+json"]').evaluateAll((scripts) =>
      scripts
        .map((script) => script.textContent ?? '')
        .map((text) => {
          try {
            return JSON.parse(text);
          } catch {
            return null;
          }
        })
        .filter((schema) => schema?.['@type'] === 'VideoObject')
    );

    expect(videoSchemas).toHaveLength(2);
    expect(videoSchemas).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          '@context': 'https://schema.org',
          '@type': 'VideoObject',
          name: 'Install and add workflow in CLI demo video',
          description: 'Install and add workflow in CLI demo video',
          thumbnailUrl: 'https://github.github.com/gh-aw/videos/install-and-add-workflow-in-cli.png',
          uploadDate: '2026-06-21',
          contentUrl: 'https://github.github.com/gh-aw/videos/install-and-add-workflow-in-cli.mp4',
        }),
        expect.objectContaining({
          '@context': 'https://schema.org',
          '@type': 'VideoObject',
          name: 'Create workflow on GitHub demo video',
          description: 'Create workflow on GitHub demo video',
          thumbnailUrl: 'https://github.github.com/gh-aw/videos/create-workflow-on-github.png',
          uploadDate: '2026-06-21',
          contentUrl: 'https://github.github.com/gh-aw/videos/create-workflow-on-github.mp4',
        }),
      ])
    );
  });
});
