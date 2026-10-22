import { test, expect, type Page } from "@playwright/test";

const previewLabels = ["Issue Triage", "Pull Request Review", "Documentation Maintenance", "Repository Maintenance", "Dependabot Rollout", "More..."];

function galleryGroup(page: Page) {
  return page.locator("#starlight__sidebar details").filter({
    has: page.locator("summary", { hasText: /^Gallery$/ }),
  });
}

test("Gallery keeps its five featured entries and More link outside gallery pages", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");

  const group = galleryGroup(page);
  await expect(group.locator("a")).toHaveText(previewLabels);
  await expect(group.getByRole("link", { name: "More...", exact: true })).toHaveAttribute("href", "/gh-aw/gallery/");
});

test("gallery overview exposes every catalog page in the sidebar", async ({ page }) => {
  await page.goto("/gh-aw/gallery/");

  const group = galleryGroup(page);
  await expect(group).toHaveAttribute("open", "");
  const catalogLinks = await page.locator(".sl-markdown-content table a").evaluateAll(links => links.map(link => link.getAttribute("href")));
  expect(catalogLinks).toHaveLength(15);
  for (const href of catalogLinks) {
    await expect(group.locator(`a[href="${href}"]`)).toHaveCount(1);
  }
  await expect(group.locator("a")).toHaveCount(17);
  await expect(group.getByRole("link", { name: "Multi-Repository Examples", exact: true })).toHaveAttribute("href", "/gh-aw/gallery/multi-repo/");
  await expect(group.getByRole("link", { name: "All workflows", exact: true })).toHaveAttribute("aria-current", "page");
  await expect(group.getByRole("link", { name: "More...", exact: true })).toHaveCount(0);
});

for (const route of ["ai-issue-triage", "multi-repo", "multi-repo/feature-sync"]) {
  test(`gallery/${route} has the full sidebar and marks its current page`, async ({ page }) => {
    await page.goto(`/gh-aw/gallery/${route}/`);

    const group = galleryGroup(page);
    await expect(group.locator("a")).toHaveCount(17);
    await expect(group.locator('a[aria-current="page"]')).toHaveAttribute("href", `/gh-aw/gallery/${route}/`);
    await expect(group).toHaveAttribute("open", "");
  });
}

test("More expands the catalog on navigation and leaving the gallery restores the preview", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  await group.getByRole("link", { name: "More...", exact: true }).click();
  await expect(page).toHaveURL(/\/gh-aw\/gallery\/$/);
  await expect(group.locator("a")).toHaveCount(17);

  await group.getByRole("link", { name: "Feature Synchronization", exact: true }).click();
  await expect(page).toHaveURL(/\/gh-aw\/gallery\/multi-repo\/feature-sync\/$/);
  await expect(group.locator("a")).toHaveCount(17);

  await page.locator('#starlight__sidebar .top-level a[href="/gh-aw/introduction/overview/"]').click();
  await expect(group.locator("a")).toHaveText(previewLabels);
});

test("mobile gallery navigation exposes the full catalog", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/gh-aw/gallery/multi-repo/feature-sync/");
  await page.getByRole("button", { name: "Menu", exact: true }).click();
  await expect(galleryGroup(page).getByRole("link", { name: "Feature Synchronization", exact: true })).toBeVisible();
  await expect(galleryGroup(page).locator("a")).toHaveCount(17);
});

test.describe("without JavaScript", () => {
  test.use({ javaScriptEnabled: false });

  test("gallery expansion is rendered on the server", async ({ page }) => {
    await page.goto("/gh-aw/gallery/ai-issue-triage/");
    await expect(galleryGroup(page).locator("a")).toHaveCount(17);
    await page.goto("/gh-aw/introduction/overview/");
    await expect(galleryGroup(page).locator("a")).toHaveText(previewLabels);
  });
});
