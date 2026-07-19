// @ts-check

import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import rehypeStringify from 'rehype-stringify';

/**
 * Remark plugin that applies markdown transformation to the content of
 * specific HTML tags that are treated as opaque HTML blocks by remark-parse.
 *
 * CommonMark block-level HTML elements such as `<details>` and `<summary>`
 * are treated as opaque HTML blocks by remark-parse, so any markdown syntax
 * they contain is left as raw text rather than being processed.  This plugin
 * runs after remark-parse to fill that gap: it visits every `html` MDAST node
 * and applies a full markdown transformation to the content of `<summary>`
 * (and similar inline-content elements), producing proper HTML.
 *
 * This is the correct, AST-level fix for GFM alerts that contain
 * `<details>/<summary>` with markdown links — replacing the previous
 * approach of manipulating rendered HTML strings after compilation.
 */

/**
 * Reusable processor for converting markdown inline content to HTML.
 *
 * `allowDangerousHtml` is required to preserve inline HTML (e.g. `<b>`,
 * `<code>`) that authors place inside target tags alongside their markdown.
 * This is safe here because:
 *  - the processor runs **at build time**, never in a browser;
 *  - inputs are documentation source files committed by repository maintainers
 *    and are not derived from end-user input;
 *  - no output is injected into an HTTP response or an untrusted context.
 */
const inlineProcessor = unified()
	.use(remarkParse)
	.use(remarkRehype, { allowDangerousHtml: true })
	.use(rehypeStringify, { allowDangerousHtml: true });

/**
 * Apply markdown transformation to text, producing HTML.
 * remark-parse wraps inline content in a `<p>` element; that wrapper is
 * stripped so the result can be placed back inside the original HTML tag.
 *
 * @param {string} text
 * @returns {string}
 */
function applyMarkdownTransformation(text) {
	const result = String(inlineProcessor.processSync(text));
	// Strip the outer <p>…</p> wrapper added by remark for a single paragraph.
	return result.replace(/^<p>([\s\S]*?)<\/p>\n?$/, '$1');
}

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
		node.value = processMarkdownInHtml(node.value);
	}

	const { children } = node;
	if (Array.isArray(children)) {
		for (const child of children) visit(child);
	}
}

/**
 * Tags whose text content should have markdown transformed to HTML.
 * These are inline-text contexts that appear as block-level HTML in markdown
 * and therefore bypass normal remark inline processing.
 *
 * @type {string[]}
 */
const INLINE_TEXT_TAGS = ['summary', 'figcaption', 'caption', 'dt', 'dd', 'th', 'td', 'li'];

/**
 * Apply markdown transformation inside the content of specific HTML tags.
 *
 * - Only targets known inline-text tags listed in INLINE_TEXT_TAGS.
 * - Text that already contains an `<a` tag is left untouched to avoid
 *   double-processing.
 *
 * @param {string} html
 * @returns {string}
 */
function processMarkdownInHtml(html) {
	const tagPattern = INLINE_TEXT_TAGS.join('|');
	const tagRe = new RegExp(
		`(<(?:${tagPattern})(?:\\s[^>]*)?>)([\\s\\S]*?)(<\\/(?:${tagPattern})>)`,
		'gi',
	);

	return html.replace(tagRe, (_match, openTag, content, closeTag) => {
		// Skip content that already contains an anchor tag to avoid double-processing.
		if (/<a[\s>]/i.test(content)) return _match;

		const processed = applyMarkdownTransformation(content);
		return openTag + processed + closeTag;
	});
}
