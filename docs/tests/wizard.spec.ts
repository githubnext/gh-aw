import { expect, test } from '@playwright/test';

test.describe('AW wizard page', () => {
  test('renders the client-side wizard shell and advances through data-driven steps', async ({ page }) => {
    await page.goto('/gh-aw/wizard/');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Build an agentic workflow prompt step by step' })).toBeVisible();
    await expect(page.getByText('Step 1 of 3')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Choose a workflow goal' })).toBeVisible();

    await page.getByLabel('Generate a recurring report').check();
    await page.getByRole('button', { name: 'Next' }).click();

    await expect(page.getByRole('heading', { name: 'Choose a trigger' })).toBeVisible();
    await expect(page.getByLabel('Every day')).toBeVisible();
    await expect(page.getByLabel('Every week')).toBeVisible();

    await page.getByRole('button', { name: 'Next' }).click();

    await expect(page.getByRole('heading', { name: 'Review the inferred output destination' })).toBeVisible();
    await expect(page.locator('[data-wizard-summary]')).toContainText('Generate a recurring report');
    await expect(page.locator('[data-wizard-preview]')).toContainText('Destination: Post a discussion');
    await expect(page.getByRole('button', { name: 'Review selections' })).toBeVisible();
  });

  test('infers a comment destination for the issue automation goal', async ({ page }) => {
    await page.goto('/gh-aw/wizard/');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Step 1 of 3')).toBeVisible();
    await page.getByLabel('Automate issue triage or replies').check();
    await page.getByRole('button', { name: 'Next' }).click();

    await expect(page.getByRole('heading', { name: 'Choose a trigger' })).toBeVisible();
    await expect(page.getByLabel('When an issue is opened')).toBeVisible();
    await expect(page.getByLabel('When someone comments on an issue')).toBeVisible();

    await page.getByRole('button', { name: 'Next' }).click();

    await expect(page.getByRole('heading', { name: 'Review the inferred output destination' })).toBeVisible();
    await expect(page.locator('[data-wizard-summary]')).toContainText('Automate issue triage or replies');
    await expect(page.locator('[data-wizard-preview]')).toContainText('Destination: Post a comment');
    await expect(page.getByRole('button', { name: 'Review selections' })).toBeVisible();
  });
});
