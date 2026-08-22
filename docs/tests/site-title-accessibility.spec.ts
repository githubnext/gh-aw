import { expect, test } from "@playwright/test";

test("mobile home logo link has an accessible name", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/gh-aw/");
  await page.waitForLoadState("networkidle");

  await expect(page.locator("a.site-title").first()).toHaveAccessibleName("GitHub Agentic Workflows home");
});
