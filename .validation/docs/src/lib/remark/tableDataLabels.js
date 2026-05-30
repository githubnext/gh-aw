// @ts-check

/**
 * Add `data-label` attributes to markdown table cells at build time.
 * This ensures mobile card-style tables have labels even when JavaScript is disabled.
 *
 * @returns {(tree: import('unist').Node) => void}
 */
export default function remarkTableDataLabels() {
	/**
	 * @param {import('unist').Node} tree
	 */
	return function transform(tree) {
		visit(tree);
	};
}

/**
 * @param {any} node
 */
function visit(node) {
	if (!node || typeof node !== 'object') return;

	if (node.type === 'table' && Array.isArray(node.children) && node.children.length > 0) {
		const [headerRow, ...bodyRows] = node.children;
		const hasHeaderRow =
			headerRow && headerRow.type === 'tableRow' && Array.isArray(headerRow.children);
		const headers = hasHeaderRow
			? headerRow.children.map((cell) => getText(cell).trim())
			: [];

		for (const row of bodyRows) {
			if (!Array.isArray(row?.children)) continue;

			for (const [index, cell] of row.children.entries()) {
				const label = headers[index];
				if (!label || !cell || typeof cell !== 'object') continue;

				if (!cell.data?.hProperties || typeof cell.data.hProperties !== 'object') {
					cell.data = { ...(cell.data || {}), hProperties: {} };
				}
				cell.data.hProperties['data-label'] = label;
			}
		}
	}

	const { children } = node;
	if (Array.isArray(children)) {
		for (const child of children) visit(child);
	}
}

/**
 * @param {any} node
 * @returns {string}
 */
function getText(node) {
	if (!node || typeof node !== 'object') return '';

	if (node.type === 'text' && typeof node.value === 'string') {
		return node.value;
	}

	if (node.type === 'inlineCode' && typeof node.value === 'string') {
		return node.value;
	}

	if (!Array.isArray(node.children)) return '';
	return node.children.map((child) => getText(child)).join('');
}
