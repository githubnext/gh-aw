import { expect, test, type Page } from '@playwright/test';

const PIXEL_TOLERANCE = 1;
const ZEN_MODE_MOBILE_BREAKPOINT = 800;

const workshopDevices = [
	{ name: 'Galaxy S21', width: 360, height: 800 },
	{ name: 'iPhone 12', width: 390, height: 844 },
	{ name: 'iPad', width: 768, height: 1024 },
	{ name: 'iPad Pro 11', width: 834, height: 1194 },
	{ name: 'HD Desktop', width: 1366, height: 768 },
];

async function startWorkshop(page: Page) {
	await page.goto('/gh-aw/workshop/');
	await page.waitForLoadState('networkidle');
	await page.locator('[data-workshop-entry-path="ui-learner"]').click();
	await page.locator('[data-workshop-scenario="daily-status"]').click();
	await expect(page.locator('[data-workshop-tutorial]')).toBeVisible();
}

test.describe('Workshop tutorial', () => {
	test('progress rail follows the active step instead of saved completion history', async ({ page }) => {
		await startWorkshop(page);

		const bubbles = page.locator('[data-workshop-step-bubbles] .aw-workshop-step-bubble');
		const bubbleCount = await bubbles.count();
		const firstStepPercent = bubbleCount <= 1 ? 100 : 0;
		const thirdStepPercent = bubbleCount <= 1 ? 100 : Math.round((2 / (bubbleCount - 1)) * 100);

		await expect(page.locator('[data-workshop-step-position]')).toHaveText(`Step 1 of ${bubbleCount}`);
		await expect(page.locator('[data-workshop-lesson-percent]')).toHaveText(`${firstStepPercent}%`);
		await expect(bubbles.nth(0)).toHaveClass(/is-active/);
		await expect(bubbles.nth(0)).not.toHaveClass(/is-complete/);

		await page.getByRole('button', { name: /Next step/i }).click();
		await page.getByRole('button', { name: /Next step/i }).click();

		await expect(page.locator('[data-workshop-step-position]')).toHaveText(`Step 3 of ${bubbleCount}`);
		await expect(page.locator('[data-workshop-lesson-percent]')).toHaveText(`${thirdStepPercent}%`);
		await expect(bubbles.nth(0)).toHaveClass(/is-complete/);
		await expect(bubbles.nth(1)).toHaveClass(/is-complete/);
		await expect(bubbles.nth(2)).toHaveClass(/is-active/);

		await bubbles.nth(0).click();

		await expect(page.locator('[data-workshop-step-position]')).toHaveText(`Step 1 of ${bubbleCount}`);
		await expect(page.locator('[data-workshop-lesson-percent]')).toHaveText(`${firstStepPercent}%`);
		await expect(bubbles.nth(0)).toHaveClass(/is-active/);
		await expect(bubbles.nth(0)).not.toHaveClass(/is-complete/);
		await expect(bubbles.nth(1)).not.toHaveClass(/is-complete/);
	});

	test('switching entry path clears previous scenario and restarts the flow', async ({ page }) => {
		await startWorkshop(page);

		await page.getByRole('button', { name: /Next step/i }).click();
		await expect(page.locator('[data-workshop-step-position]')).toHaveText(/Step 2 of/);

		await page.getByRole('button', { name: /Change route/i }).click();
		await page.locator('[data-workshop-entry-path="cli-user"]').click();

		await expect(page.locator('[data-workshop-setup-step="scenario"]')).toBeVisible();
		await expect(page.locator('[data-workshop-scenario][aria-pressed="true"]')).toHaveCount(0);

		const stateAfterPathChange = await page.evaluate(() => {
			return window.sessionStorage.getItem('gh-aw-docs-workshop-state');
		});
		expect(stateAfterPathChange).toContain('"journeyId":"terminal"');
		expect(stateAfterPathChange).toContain('"scenarioId":""');
		expect(stateAfterPathChange).toContain('"stepKey":"00-welcome"');

		await page.locator('[data-workshop-scenario="daily-docs"]').click();
		await expect(page.locator('[data-workshop-step-position]')).toHaveText(/Step 1 of/);

		await page.getByRole('button', { name: /Home/i }).click();
		await expect(page.locator('[data-workshop-setup-step="workspace"]')).toBeVisible();

		const stateAfterHome = await page.evaluate(() => {
			return window.sessionStorage.getItem('gh-aw-docs-workshop-state');
		});
		expect(stateAfterHome).toBeNull();
	});

	for (const device of workshopDevices) {
		test(`renders the workshop flow cleanly on ${device.name}`, async ({ page }) => {
			await page.setViewportSize({ width: device.width, height: device.height });
			await startWorkshop(page);
			const isZenMobileViewport = device.width <= ZEN_MODE_MOBILE_BREAKPOINT;

			await expect(page.locator('.aw-workshop-panel-shell')).toBeVisible();
			await expect(page.locator('.aw-workshop-step-content')).toBeVisible();
			await expect(page.getByRole('button', { name: /Next step|Finish workshop/i })).toBeVisible();
			if (isZenMobileViewport) {
				await expect(page.locator('.aw-workshop-flow-header')).toBeHidden();
				await expect(page.locator('.aw-workshop-progress-card')).toBeHidden();
				await expect(page.locator('.aw-workshop-panel-summary')).toBeHidden();
				await expect(page.locator('.aw-workshop-panel-actions')).toBeHidden();
			} else {
				await expect(page.locator('.aw-workshop-flow-header')).toBeVisible();
				await expect(page.locator('.aw-workshop-progress-card')).toBeVisible();
				await expect(page.locator('.aw-workshop-panel-summary')).toBeVisible();
				await expect(page.locator('.aw-workshop-panel-actions')).toBeVisible();
			}

			const layout = await page.evaluate(() => {
				const stepContent = document.querySelector('.aw-workshop-step-content');
				const stepContentStyle = stepContent ? window.getComputedStyle(stepContent) : null;
				const selectors = [
					'.aw-workshop-panel-shell',
					'.aw-workshop-progress-card',
					'.aw-workshop-step-content',
					'.aw-workshop-panel-footer',
				];

				const bounds = selectors.map((selector) => {
					const element = document.querySelector(selector);
					if (!element) return { selector, exists: false, left: 0, right: 0, width: 0 };
					const rect = element.getBoundingClientRect();
					return {
						selector,
						exists: true,
						left: rect.left,
						right: rect.right,
						width: rect.width,
					};
				});

				return {
					viewportWidth: window.innerWidth,
					availableWidth: document.body.getBoundingClientRect().width,
					scrollWidth: document.scrollingElement?.scrollWidth ?? document.documentElement.scrollWidth,
					clientWidth: document.scrollingElement?.clientWidth ?? document.documentElement.clientWidth,
					bounds,
					stepContentStyle: stepContentStyle ? {
						borderWidth: stepContentStyle.borderWidth,
						borderRadius: stepContentStyle.borderRadius,
						backgroundImage: stepContentStyle.backgroundImage,
						backgroundColor: stepContentStyle.backgroundColor,
						boxShadow: stepContentStyle.boxShadow,
					} : null,
				};
			});

			expect(layout.scrollWidth).toBeLessThanOrEqual(layout.clientWidth + PIXEL_TOLERANCE);
			for (const bound of layout.bounds) {
				expect(bound.exists).toBe(true);
				if (!bound.exists) continue;
				expect(bound.left).toBeGreaterThanOrEqual(-PIXEL_TOLERANCE);
				expect(bound.right).toBeLessThanOrEqual(layout.viewportWidth + PIXEL_TOLERANCE);
			}
			if (isZenMobileViewport) {
				const panelShell = layout.bounds.find((bound) => bound.selector === '.aw-workshop-panel-shell');
				expect(panelShell?.left).toBeLessThanOrEqual(PIXEL_TOLERANCE);
				expect(panelShell?.right).toBeGreaterThanOrEqual(layout.availableWidth - PIXEL_TOLERANCE);
				expect(panelShell?.width).toBeGreaterThanOrEqual(layout.availableWidth - PIXEL_TOLERANCE);
				expect(layout.stepContentStyle).toEqual({
					borderWidth: '0px',
					borderRadius: '0px',
					backgroundImage: 'none',
					backgroundColor: 'rgba(0, 0, 0, 0)',
					boxShadow: 'none',
				});
			}
		});
	}
});
