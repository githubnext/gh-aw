/**
 * Skip Link Accessibility Enhancement
 *
 * Starlight renders the main content inside a `<main>` element but does not
 * expose a stable `id` on it.  The page title heading receives `id="_top"` by
 * default, which the custom SkipLink component previously targeted.  The name
 * `_top` implies "top of page" rather than "main content", creating ambiguity
 * for assistive technology users (WCAG 2.4.1).
 *
 * This script adds the Starlight-compatible `id="starlight__main"` and
 * `tabindex="-1"` to the `<main>` element so the skip link
 * (`href="#starlight__main"`) lands on the correct landmark and keyboard focus
 * is reliably placed there. For backward compatibility with existing links,
 * it also ensures a hidden `#main-content` anchor exists inside `<main>`.
 * The `tabindex="-1"` attribute makes the element programmatically focusable
 * without including it in the natural tab order.
 *
 * The enhancement is applied on every page load and re-applied on Astro
 * client-side navigation so it works across all pages and navigations.
 */

function enhanceMainLandmark(): void {
	const main = document.querySelector<HTMLElement>('main');
	if (!main) {
		return;
	}
	if (main.id !== 'starlight__main') {
		main.id = 'starlight__main';
	}
	if (!main.hasAttribute('tabindex')) {
		main.setAttribute('tabindex', '-1');
	}
	if (!main.querySelector('#main-content')) {
		const alias = document.createElement('span');
		alias.id = 'main-content';
		alias.setAttribute('aria-hidden', 'true');
		alias.className = 'sr-only';
		main.prepend(alias);
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
