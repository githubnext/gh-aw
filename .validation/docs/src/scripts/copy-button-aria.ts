/**
 * Copy Button Accessibility Enhancement
 *
 * Expressive Code (used by Starlight for code blocks) renders copy-to-clipboard
 * buttons with a `title` attribute but without an `aria-label`. Screen readers do
 * not reliably announce `title`, so the button has no accessible name for
 * assistive technology users (WCAG 2.1 SC 4.1.2).
 *
 * This script finds all copy buttons that carry a `title` but no `aria-label`
 * and mirrors the `title` value into `aria-label`, giving the button an
 * accessible name without changing its visible appearance.
 *
 * The enhancement is applied on every page load and re-applied on Astro
 * client-side navigation so that dynamically inserted code blocks are also
 * covered.
 */

/**
 * Adds `aria-label` to every copy button that has `title` but no `aria-label`.
 */
function enhanceCopyButtons(): void {
	const buttons = document.querySelectorAll<HTMLButtonElement>(
		'.expressive-code button[title]:not([aria-label])',
	);

	buttons.forEach((button) => {
		const label = button.getAttribute('title');
		if (label) {
			button.setAttribute('aria-label', label);
		}
	});
}

// Run on initial page load
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', enhanceCopyButtons);
} else {
	enhanceCopyButtons();
}

// Re-run on Astro client-side navigation
document.addEventListener('astro:page-load', enhanceCopyButtons);
