import { test, expect } from "@playwright/test";

for (const example of [
  { slug: "daily-report-portfolio", title: "Daily Report Portfolio" },
  { slug: "linter-factory", title: "Linter Factory" },
]) {
  test(`${example.title} is a concise Gallery example with source links`, async ({ page }) => {
    const response = await page.goto(`/gh-aw/gallery/${example.slug}/`);
    expect(response?.status()).toBe(200);
    await expect(page.getByRole("heading", { name: example.title, exact: true })).toBeVisible();
    const content = page.locator(".sl-markdown-content");
    expect((await content.innerText()).split(/\s+/).length).toBeLessThan(850);
    await expect(content.locator('a[href*="github.com/github/gh-aw/blob/main/.github/workflows/"]').first()).toBeVisible();
    await expect(content.locator('a[href="/gh-aw/reference/work-queue/"]').first()).toBeVisible();
    await expect(
      page
        .locator("#starlight__sidebar details")
        .filter({
          has: page.locator("summary", { hasText: /^Gallery$/ }),
        })
        .getByRole("link", { name: example.title, exact: true })
    ).toHaveAttribute("aria-current", "page");
    await expect(
      page
        .locator("#starlight__sidebar details")
        .filter({
          has: page.locator("summary", { hasText: /^Design Patterns$/ }),
        })
        .getByRole("link", { name: example.title, exact: true, includeHidden: true })
    ).toHaveCount(0);

    const diagram = content.locator("pre.mermaid svg");
    await expect(diagram).toHaveCount(1);
    await expect(diagram).toBeVisible();
    await expect(content.locator("pre.mermaid .error-text")).toHaveCount(0);
  });

  test(`${example.title} retains its old URL as a redirect`, async ({ page }) => {
    await page.goto(`/gh-aw/patterns/${example.slug}/`);
    await expect(page).toHaveURL(new RegExp(`/gh-aw/gallery/${example.slug}/$`));
    await expect(page.getByRole("heading", { name: example.title, exact: true })).toBeVisible();
  });
}

test("queue reference owns memory configuration and diagnostic details", async ({ page }) => {
  await page.goto("/gh-aw/reference/work-queue/");
  const content = page.locator(".sl-markdown-content");
  await expect(content.locator("#claim-scoped-memory")).toBeVisible();
  await expect(content.locator("#diagnostic-artifacts")).toBeVisible();
  await expect(content).toContainText("262144");
  await expect(content).toContainText("safe-output-items.jsonl");
  await expect(content).toContainText("Redaction does not anonymize");
});
