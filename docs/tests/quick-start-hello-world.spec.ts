import { test, expect } from '@playwright/test';

test.describe('Quick Start Hello World Example', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/gh-aw/setup/quick-start/');
    await page.waitForLoadState('networkidle');
  });

  test('should display Hello World workflow example in Step 2', async ({ page }) => {
    // Verify the page contains the Hello World workflow title
    const helloWorldText = page.locator('text=My First Agentic Workflow').first();
    await expect(helloWorldText).toBeVisible();
    
    // Verify it's a simple workflow_dispatch trigger (not cron)
    const workflowDispatch = page.locator('text=workflow_dispatch').first();
    await expect(workflowDispatch).toBeVisible();
    
    // Verify the first workflow example mentions Hello World
    const helloText = page.locator('text=Hello from my first agentic workflow').first();
    await expect(helloText).toBeVisible();
  });

  test('should have minimal configuration in Hello World example', async ({ page }) => {
    // Verify the example uses simple create-discussion safe-output
    const createDiscussion = page.locator('text=create-discussion').first();
    await expect(createDiscussion).toBeVisible();
    
    // Verify it has read-only permissions
    const readPermissions = page.locator('text=contents: read').first();
    await expect(readPermissions).toBeVisible();
  });

  test('should have inline comments explaining configuration', async ({ page }) => {
    // Verify the example includes explanatory comments
    await expect(page.locator('text=Trigger: Run manually from GitHub Actions UI').first()).toBeVisible();
    await expect(page.locator('text=Permissions: Read repository information').first()).toBeVisible();
    await expect(page.locator('text=AI Engine: Use GitHub Copilot').first()).toBeVisible();
    await expect(page.locator('text=Safe Output: Allow creating discussions').first()).toBeVisible();
  });

  test('should reference daily-team-status as a more complex example', async ({ page }) => {
    // Verify the daily-team-status is referenced as an advanced example
    const advancedLink = page.locator('a:has-text("Daily Team Status")');
    await expect(advancedLink).toBeVisible();
    
    // Verify it links to the examples section
    const href = await advancedLink.getAttribute('href');
    expect(href).toContain('/examples/scheduled/daily-team-status');
  });

  test('should have Step 4 reference hello-world workflow', async ({ page }) => {
    // Verify Step 4 uses the hello-world workflow name
    const step4Section = page.locator('h3:has-text("Step 4")');
    await expect(step4Section).toBeVisible();
    
    // Verify the run command uses hello-world
    await expect(page.locator('text=gh aw run hello-world')).toBeVisible();
  });

  test('should explain workflow structure with configuration and instructions sections', async ({ page }) => {
    // Verify the Understanding section explains frontmatter and markdown body
    await expect(page.locator('h3:has-text("Configuration (Frontmatter)")')).toBeVisible();
    await expect(page.locator('h3:has-text("Instructions (Markdown Body)")')).toBeVisible();
    
    // Verify it explains what frontmatter is
    await expect(page.locator('text=What is frontmatter?')).toBeVisible();
  });
});
