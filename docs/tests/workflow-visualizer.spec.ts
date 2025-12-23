import { test, expect } from '@playwright/test';

test.describe('Workflow Hero Playground', () => {
  test('renders a mermaid SVG for the selected workflow', async ({ page }) => {
    await page.goto('/gh-aw/playground/');
    await page.waitForLoadState('networkidle');

    const hero = page.locator('[data-hero-playground]');
    await expect(hero).toBeVisible();

    const select = hero.locator('[data-hero-select]');
    await expect(select).toBeVisible();

    const diagram = hero.locator('[data-hero-graph-canvas]');
    await expect(diagram).toBeVisible();

    // Mermaid should inject an SVG element on success.
    await expect(diagram.locator('svg')).toBeVisible({ timeout: 10_000 });

    // Should have at least one node group.
    const gCount = await diagram.locator('svg g').count();
    expect(gCount).toBeGreaterThan(0);
  });
});
