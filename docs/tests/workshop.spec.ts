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
				const workshopRoot = document.querySelector('.aw-workshop');
				const panelHeader = document.querySelector('.aw-workshop-panel-header');
				const panelFooter = document.querySelector('.aw-workshop-panel-footer');
				const stepContentRect = stepContent?.getBoundingClientRect() ?? null;
				const panelHeaderRect = panelHeader?.getBoundingClientRect() ?? null;
				const panelFooterRect = panelFooter?.getBoundingClientRect() ?? null;
				const selectors = [
					'.aw-workshop',
					'.aw-workshop-panel-shell',
					'.aw-workshop-panel-header',
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
					workshopRootStyle: workshopRoot ? {
						marginTop: window.getComputedStyle(workshopRoot).marginTop,
					} : null,
					bounds,
					panelAlignment: stepContentRect ? {
						stepContentLeft: stepContentRect.left,
						panelHeaderLeft: panelHeaderRect?.left ?? 0,
						panelFooterLeft: panelFooterRect?.left ?? 0,
					} : null,
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
				expect(layout.workshopRootStyle).toEqual({ marginTop: '0px' });
				const panelShell = layout.bounds.find((bound) => bound.selector === '.aw-workshop-panel-shell');
				expect(panelShell?.left).toBeLessThanOrEqual(PIXEL_TOLERANCE);
				expect(panelShell?.right).toBeGreaterThanOrEqual(layout.availableWidth - PIXEL_TOLERANCE);
				expect(panelShell?.width).toBeGreaterThanOrEqual(layout.availableWidth - PIXEL_TOLERANCE);
				expect(layout.panelAlignment).not.toBeNull();
				expect(Math.abs((layout.panelAlignment?.panelHeaderLeft ?? 0) - (layout.panelAlignment?.stepContentLeft ?? 0))).toBeLessThanOrEqual(PIXEL_TOLERANCE);
				expect(Math.abs((layout.panelAlignment?.panelFooterLeft ?? 0) - (layout.panelAlignment?.stepContentLeft ?? 0))).toBeLessThanOrEqual(PIXEL_TOLERANCE);
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

test.describe('Workshop URL hash navigation', () => {
	test('encodes journey and scenario in the URL hash after setup', async ({ page }) => {
		await page.goto('/gh-aw/workshop/');
		await page.waitForLoadState('networkidle');

		await page.locator('[data-workshop-journey="github"]').click();
		expect(page.url()).toMatch(/#j=github$/);

		await page.locator('[data-workshop-scenario="daily-status"]').click();
		await expect(page.locator('[data-workshop-tutorial]')).toBeVisible();
		expect(page.url()).toMatch(/#j=github&s=daily-status&t=.+$/);
	});

	test('encodes current step in the URL hash when navigating steps', async ({ page }) => {
		await startWorkshop(page);

		const initialUrl = page.url();
		expect(initialUrl).toContain('#j=github&s=daily-status&t=');

		await page.getByRole('button', { name: /Next step/i }).click();
		const nextUrl = page.url();
		expect(nextUrl).toContain('#j=github&s=daily-status&t=');
		expect(nextUrl).not.toBe(initialUrl);
	});

	test('restores tutorial step from URL hash on direct navigation', async ({ page }) => {
		await startWorkshop(page);

		await page.getByRole('button', { name: /Next step/i }).click();
		const tutorialUrl = page.url();
		// Capture which step is currently displayed so we can assert the same step is restored.
		const stepPosition = await page.locator('[data-workshop-step-position]').textContent();

		// Navigate away so storage would otherwise default back to step 1.
		await page.goto('/gh-aw/workshop/');
		await page.waitForLoadState('networkidle');
		// Clear session storage so the only source of truth for the step is the URL hash.
		await page.evaluate(() => sessionStorage.clear());

		// Navigate directly to the captured URL — hash must take precedence over (empty) storage.
		await page.goto(tutorialUrl);
		await page.waitForLoadState('networkidle');
		await expect(page.locator('[data-workshop-tutorial]')).toBeVisible();
		expect(page.url()).toBe(tutorialUrl);
		// Assert the specific step is displayed, not merely some tutorial state.
		await expect(page.locator('[data-workshop-step-position]')).toHaveText(stepPosition || '');
	});

	test('supports browser back navigation from tutorial to setup', async ({ page }) => {
		await page.goto('/gh-aw/workshop/');
		await page.waitForLoadState('networkidle');

		await page.locator('[data-workshop-journey="github"]').click();
		await page.locator('[data-workshop-scenario="daily-status"]').click();
		await expect(page.locator('[data-workshop-tutorial]')).toBeVisible();

		await page.locator('[data-workshop-change]').click();
		await expect(page.locator('[data-workshop-setup]')).toBeVisible();

		await page.goBack();
		await expect(page.locator('[data-workshop-tutorial]')).toBeVisible();
	});

	test('supports browser back navigation from scenario picker to workspace picker', async ({ page }) => {
		await page.goto('/gh-aw/workshop/');
		await page.waitForLoadState('networkidle');

		await page.locator('[data-workshop-journey="github"]').click();
		expect(page.url()).toMatch(/#j=github$/);
		await expect(page.locator('[data-workshop-setup-step="scenario"]')).toBeVisible();

		await page.goBack();
		await expect(page.locator('[data-workshop-setup-step="workspace"]')).toBeVisible();
		expect(page.url()).not.toContain('#');
	});
});
