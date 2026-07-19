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

test.describe('Workshop Astro rendering contract', () => {
	test('step content renders Astro-compiled HTML with block-level elements', async ({ page }) => {
		await startWorkshop(page);

		const stepContent = page.locator('[data-workshop-step-content]');
		await expect(stepContent).toBeVisible();

		// Astro-compiled markdown always produces block-level HTML elements. If the
		// pipeline were broken and raw markdown text were embedded instead, none of
		// these tags would appear.
		const html = await stepContent.innerHTML();
		expect(html).toMatch(/<(?:p|h[1-6]|ul|ol|pre|table)\b/i);
	});

	test('welcome step renders five entry cards with navigable links and updated checklist text', async ({ page }) => {
		await startWorkshop(page);

		const welcomeContract = await page.evaluate(() => {
			const node = document.getElementById('aw-workshop-step-data');
			if (!node) return { cardCount: 0, linkTargets: [] as string[], hasChecklistText: false, hasNavigableTargets: false };
			const steps = JSON.parse(node.textContent ?? '[]') as Array<{ key?: string; file?: string; html?: string }>;
			const welcomeStep = steps.find((step) => step.key === '00-welcome' || step.file === '00-welcome.md');
			const welcomeHtml = welcomeStep?.html ?? '';
			const cardBodies = [...welcomeHtml.matchAll(/<article class="aw-workshop-entry-card">([\s\S]*?)<\/article>/g)].map((match) => match[1]);
			const cardCount = cardBodies.length;
			const recommendedLinkTargets = cardBodies.flatMap((cardHtml) => {
				const nextSection = cardHtml.match(/<div class="aw-workshop-entry-card-next">([\s\S]*?)<\/div>/)?.[1] ?? '';
				return [...nextSection.matchAll(/data-workshop-local-link="([^"]+)"/g)].map((match) => decodeURIComponent(match[1]));
			});
			const knownTargets = new Set(
				steps.flatMap((step) => [step.key, step.file?.replace(/\.md$/u, '')]).filter((value): value is string => value != null),
			);
			return {
				cardCount,
				recommendedLinkTargets,
				hasChecklistText: welcomeHtml.includes('I picked the entry path above that best matches how I want to work today'),
				hasNavigableTargets: recommendedLinkTargets.every((target) => {
					const key = target.replace(/\.md(?:#.*)?$/u, '');
					return knownTargets.has(key);
				}),
			};
		});

		expect(welcomeContract.cardCount).toBe(5);
		expect(welcomeContract.recommendedLinkTargets).toHaveLength(5);
		expect(welcomeContract.hasChecklistText).toBe(true);
		expect(welcomeContract.hasNavigableTargets).toBe(true);
	});

	test('non-entry tables remain rendered as tables inside aw-workshop-table-wrap', async ({ page }) => {
		await startWorkshop(page);

		const wrappedTableCount = await page.evaluate(() => {
			const node = document.getElementById('aw-workshop-step-data');
			if (!node) return 0;
			const steps = JSON.parse(node.textContent ?? '[]') as Array<{ key?: string; html?: string }>;
			return steps.filter((step) => {
				const html = step.html ?? '';
				if (step.key === '00-welcome') return false;
				return html.includes('<div class="aw-workshop-table-wrap">')
					&& html.includes('<table>')
					&& !html.includes('aw-workshop-entry-grid');
			}).length;
		});

		expect(wrappedTableCount).toBeGreaterThan(0);
	});

	test('workshop images embedded in step data resolve to absolute URLs', async ({ page }) => {
		await startWorkshop(page);

		// The image URLs are rewritten to absolute raw.githubusercontent.com paths at
		// build time (rewriteWorkshopMarkdownForAstro + rewriteWorkshopHtml). Check
		// every img src in the embedded step-data JSON to verify no relative paths slipped
		// through. If there are no images in this workshop build the test passes vacuously.
		const imageSrcs = await page.evaluate(() => {
			const node = document.getElementById('aw-workshop-step-data');
			if (!node) return [] as string[];
			const steps = JSON.parse(node.textContent ?? '[]') as Array<{ html: string }>;
			const srcs: string[] = [];
			for (const step of steps) {
				for (const [, src] of step.html.matchAll(/<img[^>]+src="([^"]+)"/gi)) {
					srcs.push(src);
				}
			}
			return srcs;
		});

		for (const src of imageSrcs) {
			expect(src, `Image src "${src}" should be an absolute URL`).toMatch(/^https?:\/\//);
		}
	});

	test('clicking an in-content workshop link navigates to the linked step', async ({ page }) => {
		await startWorkshop(page);

		// Locate the first step in the visible flow that contains a data-workshop-local-link,
		// navigating forward until one is found or the flow ends.
		const bubbles = page.locator('[data-workshop-step-bubbles] .aw-workshop-step-bubble');
		const flowLength = await bubbles.count();

		let localLink = page.locator('[data-workshop-step-content] [data-workshop-local-link]').first();
		let found = await localLink.isVisible();

		for (let step = 1; step < flowLength && !found; step++) {
			await page.getByRole('button', { name: /Next step/i }).click();
			localLink = page.locator('[data-workshop-step-content] [data-workshop-local-link]').first();
			found = await localLink.isVisible();
		}

		if (!found) {
			// Confirm via the embedded data whether any step carries local links at all.
			// If none exist, the test passes vacuously (the content simply has no links).
			// If they exist but are not rendered, that is a bug and the test should fail.
			const hasLocalLinks = await page.evaluate(() => {
				const node = document.getElementById('aw-workshop-step-data');
				if (!node) return false;
				const steps = JSON.parse(node.textContent ?? '[]') as Array<{ html: string }>;
				return steps.some((s) => s.html.includes('data-workshop-local-link'));
			});
			expect(hasLocalLinks).toBe(false);
			return;
		}

		const positionBefore = await page.locator('[data-workshop-step-position]').textContent();
		await localLink.click();
		await expect(page.locator('[data-workshop-step-position]')).not.toHaveText(positionBefore ?? '');
	});

	test('GFM task list items in step data are rendered as styled checklists, not raw bullet points', async ({ page }) => {
		await startWorkshop(page);

		// Check the embedded step-data JSON for GFM task list items. If any step's HTML
		// still contains raw "[ ]" or "[x]" text inside <li> elements it means
		// rewriteGfmTaskLists did not run or failed to match. If the content has no
		// GFM task lists the test passes vacuously.
		const result = await page.evaluate(() => {
			const node = document.getElementById('aw-workshop-step-data');
			if (!node) return { hasTaskLists: false, hasRawMarkers: false, firstTaskListStepKey: null as string | null };
			const steps = JSON.parse(node.textContent?.trim() || '[]') as Array<{ key: string; html: string }>;
			// Detect raw task-list markers that should have been transformed.
			const rawMarkerPattern = /class="task-list-item"|class="contains-task-list"/i;
			const checklistPattern = /class="aw-workshop-checklist"/i;
			const hasRawMarkers = steps.some((s) => rawMarkerPattern.test(s.html));
			const checklistStep = steps.find((s) => checklistPattern.test(s.html));
			return {
				hasTaskLists: !!checklistStep,
				hasRawMarkers,
				firstTaskListStepKey: checklistStep?.key ?? null,
			};
		});

		// Raw remark-gfm classes must not appear in any step HTML — they should have been rewritten.
		expect(result.hasRawMarkers).toBe(false);

		// If the workshop content includes task lists, navigate to the step and verify the
		// checklist is rendered with the expected workshop class. The bubble list reflects
		// only the current route's visibleFlow, so the step may not be present; in that
		// case the UI assertion is skipped rather than failing on missing navigation.
		if (result.hasTaskLists && result.firstTaskListStepKey) {
			const bubble = page.locator(
				`[data-workshop-step-bubbles] .aw-workshop-step-bubble[data-workshop-step-link="${result.firstTaskListStepKey}"]`
			);
			if ((await bubble.count()) > 0) {
				await bubble.first().click();
				const checklist = page.locator('[data-workshop-step-content] ul.aw-workshop-checklist').first();
				await expect(checklist).toBeVisible();
			}
		}
	});

	test('GFM alerts in step data are rendered as aside elements, not raw blockquotes', async ({ page }) => {
		await startWorkshop(page);

		// Check the embedded step-data JSON for GFM alert markers. If any step's HTML
		// contains raw [!NOTE]/[!TIP]/etc. text it means rewriteGfmAlerts did not run
		// or failed to match. If the content has no GFM alerts the test passes vacuously.
		const result = await page.evaluate(() => {
			const node = document.getElementById('aw-workshop-step-data');
			if (!node) return { hasAlerts: false, hasRawMarkers: false, firstAlertStepKey: null as string | null };
			const steps = JSON.parse(node.textContent?.trim() || '[]') as Array<{ key: string; html: string }>;
			const alertPattern = /\[!(NOTE|TIP|WARNING|IMPORTANT|CAUTION)\]/i;
			const asidePattern = /class="aw-workshop-admonition-(?:note|tip|warning|important|caution)"/i;
			const hasRawMarkers = steps.some((s) => alertPattern.test(s.html));
			const alertStep = steps.find((s) => asidePattern.test(s.html));
			return { hasAlerts: !!alertStep, hasRawMarkers, firstAlertStepKey: alertStep?.key ?? null };
		});

		// Raw [!TYPE] markers must not appear in any step HTML.
		expect(result.hasRawMarkers).toBe(false);

		// If the workshop content includes GFM alerts, navigate to the step by its
		// key via the data-workshop-step-link attribute. The bubble list reflects
		// only the current route's visibleFlow, so the step may not be present; in
		// that case the UI assertion is skipped rather than clicking the wrong bubble.
		if (result.hasAlerts && result.firstAlertStepKey) {
			const bubble = page.locator(
				`[data-workshop-step-bubbles] .aw-workshop-step-bubble[data-workshop-step-link="${result.firstAlertStepKey}"]`
			);
			if ((await bubble.count()) > 0) {
				await bubble.first().click();
				const aside = page.locator('[data-workshop-step-content] aside[class*="aw-workshop-admonition-"]').first();
				await expect(aside).toBeVisible();
			}
		}
	});
});
