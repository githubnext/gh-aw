import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const testDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(testDir, "../..");

test.describe("Hidden text cloaking guard", () => {
  test("README does not include hidden setup guidance", async () => {
    const readme = await readFile(resolve(repoRoot, "README.md"), "utf8");

    expect(readme).not.toMatch(/<!--[\s\S]*?(fellow agent|agentic workflows)[\s\S]*?-->/i);
    expect(readme).not.toMatch(/<details[\s>][\s\S]*agentic workflows[\s\S]*<\/details>/i);
  });

  test("responsive header does not ship hidden dropdown link text while closed", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 });
    await page.goto("/gh-aw/");
    await page.waitForLoadState("networkidle");

    const menuButton = page.locator(".hamburger-btn");
    const dropdown = page.locator(".tablet-dropdown");

    await expect(menuButton).toBeVisible();
    await expect(dropdown).toBeHidden();
    await expect(dropdown).toHaveText("");

    await menuButton.click();
    await expect(dropdown.locator(".dropdown-link")).toHaveCount(7);
    await expect(dropdown.locator(".dropdown-link", { hasText: "Quick Start" })).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(menuButton).toHaveAttribute("aria-expanded", "false");
    await expect(dropdown).toBeHidden();
    await expect(dropdown).toHaveText("");
  });
});
