#!/usr/bin/env node
// @ts-check

/**
 * Unit tests for remarkInlineMarkdownInHtml.
 *
 * Run with: node docs/src/lib/remark/inlineMarkdownInHtml.test.js
 */

import remarkInlineMarkdownInHtml from './inlineMarkdownInHtml.js';

let passed = 0;
let failed = 0;

function assertEqual(actual, expected, label) {
	if (actual === expected) {
		console.log(`  ✓ ${label}`);
		passed++;
	} else {
		console.error(`  ✗ ${label}`);
		console.error(`    expected: ${JSON.stringify(expected)}`);
		console.error(`    actual:   ${JSON.stringify(actual)}`);
		failed++;
	}
}

function makeHtmlNode(value) {
	return { type: 'html', value };
}

function makeTree(...htmlNodes) {
	return { type: 'root', children: htmlNodes };
}

function runTransform(tree) {
	const transform = remarkInlineMarkdownInHtml();
	transform(tree);
	return tree;
}

// -------------------------------------------------------------------
// Test: markdown link inside <summary> is converted to <a href="...">
// -------------------------------------------------------------------
console.log('\nmarkdown link inside <summary>:');
{
	const tree = makeTree(
		makeHtmlNode('<summary>Open a Codespace terminal in [Step 6](06-install-gh-aw.md) to install</summary>'),
	);
	runTransform(tree);
	assertEqual(
		tree.children[0].value,
		'<summary>Open a Codespace terminal in <a href="06-install-gh-aw.md">Step 6</a> to install</summary>',
		'single link converted',
	);
}

// -------------------------------------------------------------------
// Test: multiple links inside <summary>
// -------------------------------------------------------------------
console.log('\nmultiple links inside <summary>:');
{
	const tree = makeTree(
		makeHtmlNode('<summary>See [Step 6](step6.md) and [Step 7](step7.md) for details</summary>'),
	);
	runTransform(tree);
	assertEqual(
		tree.children[0].value,
		'<summary>See <a href="step6.md">Step 6</a> and <a href="step7.md">Step 7</a> for details</summary>',
		'both links converted',
	);
}

// -------------------------------------------------------------------
// Test: backtick code spans inside <summary> are converted to <code>
// -------------------------------------------------------------------
console.log('\nbacktick code in <summary>:');
{
	const tree = makeTree(
		makeHtmlNode('<summary>Install `gh-aw` via [Step 6](step6.md)</summary>'),
	);
	runTransform(tree);
	assertEqual(
		tree.children[0].value,
		'<summary>Install <code>gh-aw</code> via <a href="step6.md">Step 6</a></summary>',
		'backtick converted to <code>, link converted',
	);
}

// -------------------------------------------------------------------
// Test: link-like syntax inside backtick code span is NOT converted
// -------------------------------------------------------------------
console.log('\nlink syntax inside code span is not converted:');
{
	const tree = makeTree(
		makeHtmlNode('<summary>Use `[var](param)` to configure</summary>'),
	);
	runTransform(tree);
	assertEqual(
		tree.children[0].value,
		'<summary>Use <code>[var](param)</code> to configure</summary>',
		'code span link syntax not converted, backtick rendered as <code>',
	);
}

// -------------------------------------------------------------------
// Test: <summary> that already contains <a> tags is left untouched
// -------------------------------------------------------------------
console.log('\n<summary> with existing <a> tag not double-processed:');
{
	const input = '<summary>See <a href="step6.md">Step 6</a> for details</summary>';
	const tree = makeTree(makeHtmlNode(input));
	runTransform(tree);
	assertEqual(tree.children[0].value, input, 'existing anchor tag not rewritten');
}

// -------------------------------------------------------------------
// Test: html nodes without recognised tags are left unchanged
// -------------------------------------------------------------------
console.log('\nhtml nodes without target tags:');
{
	const input = '<blockquote>[link text](url) text</blockquote>';
	const tree = makeTree(makeHtmlNode(input));
	runTransform(tree);
	assertEqual(tree.children[0].value, input, 'non-target tag left unchanged');
}

// -------------------------------------------------------------------
// Test: link inside GFM alert + <summary> (the full real-world case)
// -------------------------------------------------------------------
console.log('\nfull GFM alert with <details>/<summary>:');
{
	const tree = makeTree(
		makeHtmlNode(
			'<details>\n<summary>Using the <b>CCA</b>? Open a terminal in [Step 6](06-install-gh-aw.md) to install `gh-aw`.</summary>\n\n**More info:** See [Step 1](01-prerequisites.md) for the checklist.\n\n</details>',
		),
	);
	runTransform(tree);
	const expected =
		'<details>\n<summary>Using the <b>CCA</b>? Open a terminal in <a href="06-install-gh-aw.md">Step 6</a> to install <code>gh-aw</code>.</summary>\n\n**More info:** See [Step 1](01-prerequisites.md) for the checklist.\n\n</details>';
	assertEqual(tree.children[0].value, expected, 'link in <summary> converted, body outside tag unchanged');
}

// -------------------------------------------------------------------
// Test: nested transform (html nodes inside paragraph children)
// -------------------------------------------------------------------
console.log('\nhtml node nested inside paragraph:');
{
	const htmlNode = makeHtmlNode('<summary>See [Step 6](step6.md)</summary>');
	const tree = { type: 'root', children: [{ type: 'paragraph', children: [htmlNode] }] };
	runTransform(tree);
	assertEqual(
		htmlNode.value,
		'<summary>See <a href="step6.md">Step 6</a></summary>',
		'nested html node processed',
	);
}

// -------------------------------------------------------------------
// Summary
// -------------------------------------------------------------------
console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
