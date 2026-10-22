import { test, expect, type Page } from "@playwright/test";

const sections = [
  {
    label: "Guides",
    preview: ["GitHub Actions Primer", "Using at Scale in Organizations", "More..."],
    more: "/gh-aw/practices/safe-rollout/",
    other: "/gh-aw/guides/mcps/",
    count: 8,
  },
  {
    label: "Design Patterns",
    preview: ["BatchOps", "CentralRepoOps", "More..."],
    more: "/gh-aw/patterns/chat-ops/",
    other: "/gh-aw/patterns/workqueue-ops/",
    count: 18,
  },
  {
    label: "Reference",
    preview: ["FAQ", "AI Engines", "Cost Management", "Safe Outputs", "Workflows", "More..."],
    more: "/gh-aw/reference/workflow-structure/",
    other: "/gh-aw/reference/engines/copilot/",
    count: 77,
  },
];

function sidebarGroup(page: Page, label: string) {
  return page.locator("#starlight__sidebar details").filter({
    has: page.locator("summary", { hasText: new RegExp(`^${label}$`) }),
  });
}

for (const section of sections) {
  test(`${section.label} shows its preview and More outside its section`, async ({ page }) => {
    await page.goto("/gh-aw/introduction/overview/");
    const group = sidebarGroup(page, section.label);
    await expect(group.locator("a")).toHaveText(section.preview);
    await expect(group.getByRole("link", { name: "More...", exact: true, includeHidden: true })).toHaveAttribute("href", section.more);
    if (section.label === "Reference") await expect(group).toHaveAttribute("open", "");
  });

  for (const route of [section.more, section.other]) {
    test(`${route} expands ${section.label} and highlights the current page`, async ({ page }) => {
      await page.goto(route);
      const group = sidebarGroup(page, section.label);
      await expect(group).toHaveAttribute("open", "");
      await expect(group.locator("a")).toHaveCount(section.count);
      await expect(group.locator('a[aria-current="page"]')).toHaveAttribute("href", route);
      await expect(group.getByRole("link", { name: "More...", exact: true })).toHaveCount(0);
    });
  }

  test(`${section.label} expands via More and returns to its preview on exit`, async ({ page }) => {
    await page.goto("/gh-aw/introduction/overview/");
    const group = sidebarGroup(page, section.label);
    if ((await group.getAttribute("open")) === null) await group.locator("summary").click();
    await group.getByRole("link", { name: "More...", exact: true }).click();
    await expect(page).toHaveURL(new RegExp(`${section.more}$`));
    await expect(group.locator("a")).toHaveCount(section.count);
    await expect(group).toHaveAttribute("open", "");

    await page.locator('#starlight__sidebar .top-level a[href="/gh-aw/introduction/overview/"]').click();
    await expect(group.locator("a")).toHaveText(section.preview);
  });

  test(`${section.label} has the full catalog on mobile`, async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(section.more);
    await page.getByRole("button", { name: "Menu", exact: true }).click();
    const group = sidebarGroup(page, section.label);
    await expect(group.locator('a[aria-current="page"]')).toBeVisible();
    await expect(group.locator("a")).toHaveCount(section.count);
  });
}

test("guide pages listed in other sidebar groups still expand Guides", async ({ page }) => {
  await page.goto("/gh-aw/guides/working-with-workflows/");
  const group = sidebarGroup(page, "Guides");
  await expect(group.locator("a")).toHaveCount(8);
  await expect(group).toHaveAttribute("open", "");
});

test("expanding one section keeps the other sections compact", async ({ page }) => {
  await page.goto("/gh-aw/patterns/chat-ops/");
  await expect(sidebarGroup(page, "Guides").locator("a")).toHaveText(sections[0].preview);
  await expect(sidebarGroup(page, "Gallery").locator("a")).toHaveCount(6);
  await expect(sidebarGroup(page, "Reference").locator("a")).toHaveText(sections[2].preview);

  await page.goto("/gh-aw/practices/safe-rollout/");
  await expect(sidebarGroup(page, "Design Patterns").locator("a")).toHaveText(sections[1].preview);
});

test.describe("without JavaScript", () => {
  test.use({ javaScriptEnabled: false });

  for (const section of sections) {
    test(`${section.label} is expanded on the server`, async ({ page }) => {
      await page.goto(section.more);
      await expect(sidebarGroup(page, section.label).locator("a")).toHaveCount(section.count);
      await page.goto("/gh-aw/introduction/overview/");
      await expect(sidebarGroup(page, section.label).locator("a")).toHaveText(section.preview);
    });
  }
});
