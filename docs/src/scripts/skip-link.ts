/**
 * Skip Link Accessibility Enhancement
 *
 * Starlight renders the main content inside a `<main>` element but does not
 * expose a stable `id` on it.  The page title heading receives `id="_top"` by
 * default, which the custom SkipLink component previously targeted.  The name
 * `_top` implies "top of page" rather than "main content", creating ambiguity
 * for assistive technology users (WCAG 2.4.1).
 *
 * This script adds `id="main-content"` and `tabindex="-1"` to the `<main>`
 * element so that the skip link (`href="#main-content"`) lands on the correct
 * landmark and keyboard focus is reliably placed there.  The `tabindex="-1"`
 * attribute makes the element programmatically focusable without including it
 * in the natural tab order.
 *
 * The enhancement is applied on every page load and re-applied on Astro
 * client-side navigation so it works across all pages and navigations.
 */

function enhanceMainLandmark(): void {
	const main = document.querySelector<HTMLElement>('main');
	if (!main) {
		return;
	}
	if (!main.id) {
		main.id = 'main-content';
	}
	if (!main.hasAttribute('tabindex')) {
		main.setAttribute('tabindex', '-1');
	}
}

// Run on initial page load
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', enhanceMainLandmark);
} else {
	enhanceMainLandmark();
}

// Re-run on Astro client-side navigation
document.addEventListener('astro:page-load', enhanceMainLandmark);
