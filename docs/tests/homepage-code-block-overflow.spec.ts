import { test, expect } from '@playwright/test';

test.describe('Homepage code block overflow', () => {
  const mobileWidths = [360, 393, 428];

  for (const width of mobileWidths) {
    test(`keeps landing page code block accessible at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 852 });
      await page.goto('/gh-aw/');
      await page.waitForLoadState('networkidle');

      const metrics = await page.evaluate(() => {
        const frame = document.querySelector<HTMLElement>('.expressive-code .frame');
        if (!frame) {
          return null;
        }

        const frameStyle = getComputedStyle(frame);
        const hasHorizontalOverflow = frame.scrollWidth > frame.clientWidth + 1;

        if (hasHorizontalOverflow) {
          frame.scrollLeft = frame.scrollWidth;
        }

        return {
          bodyScrollWidth: document.body.scrollWidth,
          bodyClientWidth: document.body.clientWidth,
          frameOverflowX: frameStyle.overflowX,
          frameScrollWidth: frame.scrollWidth,
          frameClientWidth: frame.clientWidth,
          frameScrolled: frame.scrollLeft > 0,
        };
      });

      expect(metrics).not.toBeNull();
      expect(metrics!.bodyScrollWidth).toBeLessThanOrEqual(metrics!.bodyClientWidth + 1);
      expect(['auto', 'scroll']).toContain(metrics!.frameOverflowX);

      if (metrics!.frameScrollWidth > metrics!.frameClientWidth + 1) {
        expect(metrics!.frameScrolled).toBeTruthy();
      }
    });
  }
});
