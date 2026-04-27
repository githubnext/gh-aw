// @ts-check

/**
 * Wrap every `<table>` element inside `.sl-markdown-content` in a
 * `<div class="table-scroll-wrapper">` at build time.
 *
 * This provides a no-JS fallback for the responsive table overflow behaviour:
 * the wrapper carries `overflow-x: auto` so wide tables are horizontally
 * scrollable on medium-width screens (641 px – 768 px) without requiring any
 * client-side JavaScript.
 *
 * @returns {(tree: import('hast').Root) => void}
 */
export default function rehypeTableWrapper() {
	/**
	 * @param {import('hast').Root} tree
	 */
	return function transform(tree) {
		wrapTables(tree);
	};
}

/**
 * Recursively walk the hast tree and wrap every `table` element that is not
 * already inside a `.table-scroll-wrapper` in a new wrapper div.
 *
 * @param {import('hast').Parent} node
 * @param {import('hast').Parent | null} [parent]
 * @param {number} [index]
 */
function wrapTables(node, parent, index) {
	if (!node || typeof node !== 'object') return;

	if (
		node.type === 'element' &&
		/** @type {import('hast').Element} */ (node).tagName === 'table' &&
		!isAlreadyWrapped(parent)
	) {
		const wrapper = /** @type {import('hast').Element} */ ({
			type: 'element',
			tagName: 'div',
			properties: { className: ['table-scroll-wrapper'] },
			children: [node],
		});

		if (parent && Array.isArray(parent.children) && typeof index === 'number') {
			parent.children.splice(index, 1, wrapper);
		}
		return;
	}

	if ('children' in node && Array.isArray(node.children)) {
		for (let i = 0; i < node.children.length; i++) {
			wrapTables(
				/** @type {import('hast').Parent} */ (node.children[i]),
				node,
				i,
			);
		}
	}
}

/**
 * Returns true when the given parent element already is (or contains) a
 * table-scroll-wrapper so we do not double-wrap.
 *
 * @param {import('hast').Parent | null | undefined} parent
 * @returns {boolean}
 */
function isAlreadyWrapped(parent) {
	if (!parent || parent.type !== 'element') return false;
	const el = /** @type {import('hast').Element} */ (parent);
	const cls = el.properties?.className;
	if (Array.isArray(cls)) {
		return cls.includes('table-scroll-wrapper');
	}
	if (typeof cls === 'string') {
		return cls.split(' ').includes('table-scroll-wrapper');
	}
	return false;
}
