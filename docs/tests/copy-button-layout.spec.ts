import { test, expect } from "@playwright/test";

const pages = [
  { route: "/gh-aw/setup/working-with-workflows/", count: 2 },
  { route: "/gh-aw/reference/custom-agent-for-aw/", count: 3 },
];

for (const width of [390, 1280, 1920]) {
  for (const { route, count } of pages) {
    test(`copy controls stay inside the prose column at ${width}px on ${route}`, async ({ page }) => {
      await page.setViewportSize({ width, height: 1080 });
      await page.goto(route);

      const buttons = page.locator(".sl-markdown-content .copy-entire-file-btn");
      await expect(buttons).toHaveCount(count);
      const layout = await buttons.evaluateAll(buttons => {
        const heading = document.querySelector(".sl-markdown-content h2");
        if (!heading) throw new Error("Missing prose heading");
        const prose = heading.getBoundingClientRect();
        return buttons.map(button => {
          const box = button.getBoundingClientRect();
          return {
            left: box.left,
            right: box.right,
            proseLeft: prose.left,
            proseRight: prose.right,
            inParagraph: button.parentElement?.tagName === "P",
          };
        });
      });
      for (const button of layout) {
        expect(button.inParagraph).toBe(true);
        expect(button.left).toBeGreaterThanOrEqual(button.proseLeft - 1);
        expect(button.right).toBeLessThanOrEqual(button.proseRight + 1);
      }

      const debugButton = buttons.filter({ hasText: "Copy debug instructions" });
      const debugBox = await debugButton.boundingBox();
      expect(debugBox).not.toBeNull();
      expect(Math.abs(debugBox!.x - layout[0].proseLeft)).toBeLessThanOrEqual(1);
    });
  }
}

test("aligned debug button still copies the fetched instructions", async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const instructions = "# Debug instructions\nInspect the failed workflow run.";
  await page.route("https://raw.githubusercontent.com/github/gh-aw/main/debug.md", route => route.fulfill({ status: 200, contentType: "text/plain", body: instructions }));
  await page.goto("/gh-aw/setup/working-with-workflows/");

  const button = page.locator('.copy-entire-file-btn[data-filepath$="/debug.md"]');
  await button.click();
  await expect(button).toContainText("Copied!");
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(instructions);
});
