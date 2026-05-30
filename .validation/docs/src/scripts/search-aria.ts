/**
 * Search Accessibility Enhancement
 *
 * Starlight/pagefind renders the search results list dynamically via JavaScript.
 * Static analysis tools cannot detect the `aria-live` region on the results
 * container because it doesn't exist in the initial HTML. This script adds
 * `aria-live="polite"` and `aria-atomic="false"` to the results container after
 * it is inserted into the DOM, ensuring screen readers announce result counts
 * as the user types.
 *
 * The observer disconnects itself once the element is enhanced to avoid
 * unnecessary DOM observation overhead. On Astro client-side navigation the
 * observer is replaced so only one observer is active at any time.
 */

const RESULTS_SELECTORS = [
  // Starlight ≥ 0.20 pagefind-ui results wrapper
  '.pagefind-ui__results',
  // Fallback: the generic results list inside the search dialog
  'dialog[aria-label] ul[role="listbox"]',
  'dialog[aria-label] [role="status"]',
];

/** Active observer — only one exists at a time. */
let activeObserver: MutationObserver | null = null;

/**
 * Tries to find and enhance the search results container.
 * Returns true when the element was found and enhanced.
 */
function applyAriaLive(): boolean {
  for (const selector of RESULTS_SELECTORS) {
    const el = document.querySelector(selector);
    if (el && !el.getAttribute('aria-live')) {
      el.setAttribute('aria-live', 'polite');
      el.setAttribute('aria-atomic', 'false');
      return true;
    }
  }
  return false;
}

function observeSearchDialog(): void {
  // Disconnect any previous observer before starting a new one.
  if (activeObserver) {
    activeObserver.disconnect();
    activeObserver = null;
  }

  // If the element is already present (e.g. revisiting a page via back/forward
  // cache), enhance it immediately without spinning up an observer.
  if (applyAriaLive()) {
    return;
  }

  // Watch for the search dialog / results container being added to the DOM.
  // Disconnect as soon as the element is found and enhanced.
  activeObserver = new MutationObserver(() => {
    if (applyAriaLive()) {
      activeObserver?.disconnect();
      activeObserver = null;
    }
  });

  activeObserver.observe(document.body, { childList: true, subtree: true });
}

// Run on initial page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', observeSearchDialog);
} else {
  observeSearchDialog();
}

// Re-run on Astro client-side navigation (replaces the previous observer)
document.addEventListener('astro:page-load', observeSearchDialog);
