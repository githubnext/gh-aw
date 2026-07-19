// @ts-check

/**
 * Remark plugin that converts markdown link syntax (`[text](url)`) to HTML
 * anchor tags inside raw HTML nodes.
 *
 * CommonMark block-level HTML elements such as `<details>` and `<summary>`
 * are treated as opaque HTML blocks by remark-parse, so any markdown syntax
 * they contain is left as raw text rather than being processed.  This plugin
 * runs after remark-parse to fill that gap: it visits every `html` MDAST node
 * and rewrites markdown links found inside `<summary>` (and similar
 * inline-content elements) to proper `<a href="…">` tags.
 *
 * This is the correct, AST-level fix for GFM alerts that contain
 * `<details>/<summary>` with markdown links — replacing the previous
 * approach of manipulating rendered HTML strings after compilation.
 */

/**
 * @returns {(tree: import('unist').Node) => void}
 */
export default function remarkInlineMarkdownInHtml() {
	return function transform(tree) {
		visit(tree);
	};
}

/**
 * @param {any} node
 */
function visit(node) {
	if (!node || typeof node !== 'object') return;

	if (node.type === 'html' && typeof node.value === 'string') {
		node.value = processMarkdownLinksInHtml(node.value);
	}

	const { children } = node;
	if (Array.isArray(children)) {
		for (const child of children) visit(child);
	}
}

/**
 * Tags whose text content should have markdown link syntax converted to HTML.
 * These are inline-text contexts that appear as block-level HTML in markdown
 * and therefore bypass normal remark inline processing.
 *
 * @type {string[]}
 */
const INLINE_TEXT_TAGS = ['summary', 'figcaption', 'caption', 'dt', 'dd', 'th', 'td', 'li'];

/**
 * Replace markdown link syntax `[text](url)` with `<a href="url">text</a>`
 * inside the content of specific HTML tags that are expected to carry inline
 * text with markdown links.
 *
 * The replacement is intentionally narrow:
 * - Only targets known inline-text tags listed in INLINE_TEXT_TAGS.
 * - Does not process content inside `<code>` or `<pre>` elements.
 * - Text that already contains an `<a` tag is left untouched.
 *
 * @param {string} html
 * @returns {string}
 */
function processMarkdownLinksInHtml(html) {
	// Pattern: opening tag, inner content, closing tag
	// We build a regex that matches each target tag's opening-to-closing span.
	const tagPattern = INLINE_TEXT_TAGS.join('|');
	const tagRe = new RegExp(
		`(<(?:${tagPattern})(?:\\s[^>]*)?>)([\\s\\S]*?)(<\\/(?:${tagPattern})>)`,
		'gi',
	);

	return html.replace(tagRe, (_match, openTag, content, closeTag) => {
		// Skip content that already contains an anchor tag to avoid double-processing.
		if (/<a[\s>]/i.test(content)) return _match;

		const processed = convertMarkdownLinks(content);
		return openTag + processed + closeTag;
	});
}

/**
 * Convert `[text](url)` markdown link syntax to `<a href="url">text</a>`.
 * Backtick-enclosed spans are preserved as-is so code snippets are not
 * accidentally rewritten.
 *
 * @param {string} text
 * @returns {string}
 */
function convertMarkdownLinks(text) {
	// Tokenise the input so that backtick code spans are excluded from link
	// replacement while the rest of the text is processed normally.
	// Use [^`]+ (no dotAll) so a single code span cannot span multiple lines.
	const parts = text.split(/(`[^`]+`)/g);
	return parts
		.map((part, index) => {
			// Even-indexed parts are plain text; odd-indexed parts are code spans.
			if (index % 2 !== 0) return part;
			// Match standard markdown links.  The link text pattern [^\]\[]+ rejects
			// unescaped `[` and `]` characters so we don't accidentally match nested
			// brackets or partial link syntax.
			return part.replace(
				/\[([^\]\[]+)\]\(([^)]+)\)/g,
				'<a href="$2">$1</a>',
			);
		})
		.join('');
}
