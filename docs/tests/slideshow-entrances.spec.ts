import { test, expect, type Page } from "@playwright/test";

const dialog = (page: Page) => page.getByRole("dialog", { name: "GitHub Agentic Workflows presentation" });
const hero = (page: Page) => dialog(page).locator('.aw-hero [data-variant="a"]');

async function pauseEntrances(page: Page) {
  await page.evaluate(() => {
    Object.defineProperty(document, "startViewTransition", { value: undefined, configurable: true });
    const animate = Element.prototype.animate;
    Element.prototype.animate = function (...args) {
      const animation = animate.apply(this, args);
      if (this.hasAttribute("data-slideshow-entrance")) {
        this.setAttribute("data-entrance-count", String(Number(this.getAttribute("data-entrance-count") ?? 0) + 1));
        animation.pause();
      }
      return animation;
    };
  });
}

async function open(page: Page) {
  await page.getByRole("button", { name: "Start slideshow" }).click();
  await expect(dialog(page)).toBeVisible();
}

test.beforeEach(async ({ page }) => {
  await page.goto("/gh-aw/");
});

test("keeps normal-page content visible and annotates every slide without nested entrances", async ({ page }) => {
  const deck = page.locator("[data-slideshow-slides]");
  await expect(deck.locator("[data-slideshow-entrance] [data-slideshow-entrance]")).toHaveCount(0);
  expect(await deck.locator(":scope > section").evaluateAll(slides => slides.every(slide => slide.querySelector("[data-slideshow-entrance]")))).toBe(true);
  const heading = page.locator('.aw-hero [data-variant="a"] h1');
  await expect(heading).toBeVisible();
  await expect(heading).toHaveCSS("opacity", "1");
  expect(await heading.evaluate(element => element.getAnimations().length)).toBe(0);
});

test("uses restrained per-item keyframes, DOM order and a bounded total duration", async ({ page }) => {
  await pauseEntrances(page);
  await open(page);
  await page.keyboard.press("ArrowRight");
  const items = hero(page).locator("[data-slideshow-entrance]");
  await expect(items).toHaveCount(4);
  const timing = await items.evaluateAll(elements =>
    elements.map(element => {
      const effect = element.getAnimations()[0].effect as KeyframeEffect;
      const { duration, delay, easing } = effect.getTiming();
      return { duration, delay, easing, translate: effect.getKeyframes()[0].translate };
    })
  );
  expect(timing).toEqual(
    [0, 80, 160, 240].map((delay, index) => ({
      duration: 480,
      delay,
      easing: "cubic-bezier(0.16, 1, 0.3, 1)",
      translate: index === 3 ? "none" : "0px 12px",
    }))
  );
  await expect(items.first()).toHaveCSS("opacity", "0");
  await page.keyboard.press("End");
  await page.keyboard.press("PageUp");
  const totals = await dialog(page)
    .locator("#learn [data-slideshow-entrance]")
    .evaluateAll(elements =>
      elements.map(element => {
        const timing = element.getAnimations()[0].effect!.getTiming();
        return Number(timing.duration) + timing.delay;
      })
    );
  expect(totals).toHaveLength(7);
  expect(Math.max(...totals)).toBe(800);
});

test("reveals items permanently after finishing, rapid navigation, exit and reopening", async ({ page }) => {
  await pauseEntrances(page);
  await open(page);
  await page.keyboard.press("ArrowRight");
  const items = hero(page).locator("[data-slideshow-entrance]");
  await expect(items.first()).toHaveAttribute("data-entrance-count", "1");
  await items.first().evaluate(element => element.getAnimations().forEach(animation => animation.finish()));
  await expect(items.first()).toHaveCSS("opacity", "1");
  expect(await items.first().evaluate(element => element.getAnimations().length)).toBe(0);
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("ArrowLeft");
  for (const item of await items.all()) {
    await expect(item).toHaveCSS("opacity", "1");
    await expect(item).toHaveAttribute("data-entrance-count", "1");
  }
  await page.keyboard.press("Escape");
  expect(await page.locator(".aw-hero [data-slideshow-entrance]").evaluateAll(elements => elements.every(element => element.getAnimations().length === 0))).toBe(true);
  await open(page);
  await page.keyboard.press("ArrowRight");
  for (const item of await items.all()) {
    await expect(item).toHaveCSS("opacity", "1");
    await expect(item).toHaveAttribute("data-entrance-count", "1");
  }
});

test("Tab reveals an entering control group immediately without consuming navigation keys", async ({ page }) => {
  await pauseEntrances(page);
  await open(page);
  await page.keyboard.press("ArrowRight");
  const cta = hero(page).locator(".aw-hero-cta");
  await expect(cta).toHaveCSS("opacity", "0");
  await hero(page)
    .locator("h1")
    .evaluate(element => {
      element.tabIndex = -1;
      element.focus();
    });
  await page.keyboard.press("Tab");
  await expect(hero(page).getByRole("link", { name: "Create a workflow", exact: true })).toBeFocused();
  await expect(cta).toHaveCSS("opacity", "1");
  expect(await cta.evaluate(element => element.getAnimations().length)).toBe(0);
  await page.keyboard.press("Tab");
  await expect(hero(page).getByRole("button", { name: "Copy installation prompt" })).toBeFocused();
  await page.keyboard.press("PageDown");
  await expect(dialog(page).locator("[data-slideshow-status]")).toContainText("3 / 12");
});

test("skips entrances for reduced motion and immediately finishes them when the preference changes", async ({ page }) => {
  await pauseEntrances(page);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await open(page);
  await page.keyboard.press("ArrowRight");
  const heading = hero(page).locator("h1");
  await expect(heading).toHaveCSS("opacity", "1");
  await expect(heading).not.toHaveAttribute("data-entrance-count");
  await page.emulateMedia({ reducedMotion: "no-preference" });
  await page.keyboard.press("ArrowRight");
  const galleryHeading = dialog(page).locator("#workflows-title");
  await expect(galleryHeading).toHaveCSS("opacity", "0");
  await page.emulateMedia({ reducedMotion: "reduce" });
  await expect(galleryHeading).toHaveCSS("opacity", "1");
  expect(await galleryHeading.evaluate(element => element.getAnimations().length)).toBe(0);
  await page.emulateMedia({ reducedMotion: "no-preference" });
  await page.keyboard.press("ArrowLeft");
  await expect(heading).not.toHaveAttribute("data-entrance-count");
});

test("holds entrances through the slide transition and cleans up after fast navigation", async ({ page }) => {
  await open(page);
  await page.keyboard.press("ArrowRight");
  await expect(dialog(page).locator("[data-slideshow-status]")).toContainText("2 / 12");
  await expect(page.locator("html")).not.toHaveAttribute("data-slideshow-direction");
  await expect(hero(page).locator("h1")).toHaveCSS("opacity", "1");
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("ArrowLeft");
  await expect(dialog(page).locator("[data-slideshow-status]")).toContainText("3 / 12");
  await expect(page.locator("html")).not.toHaveAttribute("data-slideshow-direction");
  await page.keyboard.press("Escape");
  expect(await page.locator("[data-slideshow-entrance]").evaluateAll(elements => elements.every(element => element.getAnimations().length === 0))).toBe(true);
});
