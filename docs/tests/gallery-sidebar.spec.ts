import { test, expect, type Page } from "@playwright/test";

const previewLabels = ["Issue Triage", "Pull Request Review", "Documentation Maintenance", "Repository Maintenance", "Dependabot Rollout", "More..."];

function galleryGroup(page: Page) {
  return page.locator("#starlight__sidebar details").filter({
    has: page.locator("summary", { hasText: /^Gallery$/ }),
  });
}

test("Gallery generates a More button after its five featured entries outside gallery pages", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");

  const group = galleryGroup(page);
  await expect(group.locator("a")).toHaveText(previewLabels.slice(0, -1));
  await expect(group.getByRole("button", { name: "More...", exact: true })).toHaveAttribute("aria-expanded", "false");
});

test("gallery overview exposes every catalog page in the sidebar", async ({ page }) => {
  await page.goto("/gh-aw/gallery/");

  const group = galleryGroup(page);
  await expect(group).toHaveAttribute("open", "");
  const catalogLinks = await page.locator(".sl-markdown-content table a").evaluateAll(links => links.map(link => link.getAttribute("href")));
  expect(catalogLinks).toHaveLength(17);
  for (const href of catalogLinks) {
    await expect(group.locator(`a[href="${href}"]`)).toHaveCount(1);
  }
  await expect(group.locator("a")).toHaveCount(19);
  await expect(group.getByRole("link", { name: "Multi-Repository Examples", exact: true })).toHaveAttribute("href", "/gh-aw/gallery/multi-repo/");
  await expect(group.getByRole("link", { name: "All workflows", exact: true })).toHaveAttribute("aria-current", "page");
  await expect(group.getByRole("link", { name: "More...", exact: true })).toHaveCount(0);
});

for (const route of ["ai-issue-triage", "multi-repo", "multi-repo/feature-sync"]) {
  test(`gallery/${route} has the full sidebar and marks its current page`, async ({ page }) => {
    await page.goto(`/gh-aw/gallery/${route}/`);

    const group = galleryGroup(page);
    await expect(group.locator("a")).toHaveCount(19);
    await expect(group.locator('a[aria-current="page"]')).toHaveAttribute("href", `/gh-aw/gallery/${route}/`);
    await expect(group).toHaveAttribute("open", "");
  });
}

test("More expands Gallery across navigation but resets on reload", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  const url = page.url();
  const historyLength = await page.evaluate(() => history.length);
  await group.getByRole("button", { name: "More...", exact: true }).click();
  await expect(page).toHaveURL(url);
  expect(await page.evaluate(() => history.length)).toBe(historyLength);
  await expect(group.locator("a")).toHaveCount(19);
  await expect(group.getByRole("button", { name: "More...", exact: true })).toHaveCount(0);
  await expect(group.locator("a:focus")).toHaveCount(1);
  await expect(group.getByRole("link", { name: "All workflows", exact: true })).toHaveAttribute("href", "/gh-aw/gallery/");

  await group.getByRole("link", { name: "Feature Synchronization", exact: true }).click();
  await expect(page).toHaveURL(/\/gh-aw\/gallery\/multi-repo\/feature-sync\/$/);
  await expect(group.locator("a")).toHaveCount(19);

  await page.locator('#starlight__sidebar .top-level a[href="/gh-aw/introduction/overview/"]').click();
  await expect(group.locator("a")).toHaveCount(19);
  await page.reload();
  await expect(group.locator("a")).toHaveText(previewLabels.slice(0, -1));
  await expect(group.getByRole("button", { name: "More...", exact: true })).toHaveCount(1);
});

test("Gallery expands in place on mobile without expanding other catalogs", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/gh-aw/introduction/overview/");
  await page.getByRole("button", { name: "Menu", exact: true }).click();
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  const url = page.url();
  await group.getByRole("button", { name: "More...", exact: true }).click();
  await expect(page).toHaveURL(url);
  await expect(group.locator("a")).toHaveCount(19);
  await expect(group.locator("a:focus")).toBeVisible();
  await expect(page.getByRole("button", { name: "More...", exact: true, includeHidden: true })).toHaveCount(3);
  await page.reload();
  await page.getByRole("button", { name: "Menu", exact: true }).click();
  await expect(group.locator("a")).toHaveText(previewLabels.slice(0, -1));
});

test("blocked storage still allows in-place expansion and reports the persistence failure", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, "sessionStorage", {
      get() {
        throw new DOMException("Storage is blocked", "SecurityError");
      },
    });
  });
  const warnings: string[] = [];
  page.on("console", message => {
    if (message.type() === "warning") warnings.push(message.text());
  });
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  const url = page.url();
  await group.getByRole("button", { name: "More...", exact: true }).click();
  await expect(page).toHaveURL(url);
  await expect(group.locator("a")).toHaveCount(19);
  await group.getByRole("link", { name: "All workflows", exact: true }).click();
  await expect(page).toHaveURL(/\/gh-aw\/gallery\/$/);
  expect(warnings.some(warning => warning.includes("Unable to restore sidebar navigation state."))).toBe(true);
});

test("new tabs do not inherit expansion, even when they retain their opener", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  await group.getByRole("button", { name: "More...", exact: true }).click();
  const [popup] = await Promise.all([page.waitForEvent("popup"), page.evaluate(() => window.open(location.href, "_blank"))]);
  await expect(galleryGroup(popup).locator("a")).toHaveText(previewLabels.slice(0, -1));
  await expect(group.locator("a")).toHaveCount(19);
  await popup.close();
});

test("navigation handoffs are consumed, leaving no expansion state to restore later", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  await group.getByRole("button", { name: "More...", exact: true }).click();
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem("gh-aw:sidebar:navigation"))).toBeNull();
  await group.getByRole("link", { name: "All workflows", exact: true }).click();
  await page.locator('#starlight__sidebar .top-level a[href="/gh-aw/introduction/overview/"]').click();
  await expect(group.locator("a")).toHaveCount(19);
  expect(await page.evaluate(() => sessionStorage.getItem("gh-aw:sidebar:navigation"))).toBeNull();
});

test("back and forward navigation keep the expansion within the current tab", async ({ page }) => {
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  await group.getByRole("button", { name: "More...", exact: true }).click();
  await group.getByRole("link", { name: "All workflows", exact: true }).click();
  await page.waitForLoadState("load");
  await page.goBack();
  await expect(group.locator("a")).toHaveCount(19);
  await page.goForward();
  await page.locator('#starlight__sidebar .top-level a[href="/gh-aw/introduction/overview/"]').click();
  await expect(group.locator("a")).toHaveCount(19);
});

test("same-tab link navigation also persists without the Navigation API", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, "navigation", { value: undefined });
  });
  await page.goto("/gh-aw/introduction/overview/");
  const group = galleryGroup(page);
  if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
  await group.getByRole("button", { name: "More...", exact: true }).click();
  await group.getByRole("link", { name: "All workflows", exact: true }).click();
  await page.locator('#starlight__sidebar .top-level a[href="/gh-aw/introduction/overview/"]').click();
  await expect(group.locator("a")).toHaveCount(19);
  await page.reload();
  await expect(group.locator("a")).toHaveText(previewLabels.slice(0, -1));
});

test("old saved preferences and a stale handoff cannot restore expansion on a fresh visit", async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("gh-aw:sidebar:expanded:gallery", "true");
    sessionStorage.setItem(
      "gh-aw:sidebar:navigation",
      JSON.stringify({
        destination: `${location.origin}/gh-aw/gallery/`,
        expanded: ["gallery"],
      })
    );
  });
  await page.goto("/gh-aw/introduction/overview/");
  await expect(galleryGroup(page).locator("a")).toHaveText(previewLabels.slice(0, -1));
});

test("mobile gallery navigation exposes the full catalog", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/gh-aw/gallery/multi-repo/feature-sync/");
  await page.getByRole("button", { name: "Menu", exact: true }).click();
  await expect(galleryGroup(page).getByRole("link", { name: "Feature Synchronization", exact: true })).toBeVisible();
  await expect(galleryGroup(page).locator("a")).toHaveCount(19);
});

test.describe("without JavaScript", () => {
  test.use({ javaScriptEnabled: false });

  test("gallery expansion is rendered on the server", async ({ page }) => {
    await page.goto("/gh-aw/gallery/ai-issue-triage/");
    await expect(galleryGroup(page).locator("a")).toHaveCount(19);
    await page.goto("/gh-aw/introduction/overview/");
    await expect(galleryGroup(page).locator("a")).toHaveText(previewLabels);
    await expect(galleryGroup(page).getByRole("link", { name: "More...", exact: true })).toHaveAttribute("href", "/gh-aw/gallery/");
  });
});
