import { expect, test } from '@playwright/test';

test.describe('GitHub emoji rendering', () => {
	test('renders GitHub-style emoji shortcodes and loads octicon CSS', async ({ page }) => {
		await page.goto('/gh-aw/reference/markdown/');
		await page.waitForLoadState('networkidle');

		const shipitEmoji = page.locator('img.emoji[title=":shipit:"]');
		await expect(shipitEmoji).toBeVisible();
		await expect(shipitEmoji).toHaveAttribute('src', /\/images\/icons\/emoji\/shipit\.png$/);

		const octiconStyles = await page.evaluate(() => {
			const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
			svg.setAttribute('class', 'octicon');
			svg.style.color = 'rgb(12, 34, 56)';
			document.body.appendChild(svg);

			const computed = window.getComputedStyle(svg);
			const styles = {
				display: computed.display,
				verticalAlign: computed.verticalAlign,
				fill: computed.fill,
			};

			svg.remove();
			return styles;
		});

		expect(octiconStyles.display).toBe('inline-block');
		expect(octiconStyles.verticalAlign).toBe('text-top');
		expect(octiconStyles.fill).toBe('rgb(12, 34, 56)');
	});
});
