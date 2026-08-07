#!/usr/bin/env node

import assert from "node:assert/strict";
import { buildSlideDeckUrl, isAllowedPdfContentType, isPdf, validatePdfBytes, validateSlideDeckResponse } from "./ensure-docs-slide-pdf.js";

const safeSha = "0123456789abcdef0123456789abcdef01234567";

function responseWithHeaders(headers) {
  return { headers: new Headers(headers) };
}

function assertThrowsMatching(fn, pattern, testName) {
  assert.throws(fn, pattern);
  console.log(`PASS: ${testName}`);
}

assert.equal(isPdf(Buffer.from("%PDF-1.4\n")), true);
assert.equal(isPdf(Buffer.from("not a pdf")), false);
console.log("PASS: detects PDF header");

assert.equal(isAllowedPdfContentType("application/pdf"), true);
assert.equal(isAllowedPdfContentType("application/pdf; charset=binary"), true);
assert.equal(isAllowedPdfContentType("application/octet-stream"), true);
assert.equal(isAllowedPdfContentType("text/html"), false);
console.log("PASS: validates allowed PDF content types exactly");

validateSlideDeckResponse(
  responseWithHeaders({
    "content-type": "application/pdf",
    "content-length": "1024",
  })
);
console.log("PASS: accepts valid PDF response headers");

assertThrowsMatching(() => validateSlideDeckResponse(responseWithHeaders({ "content-type": "text/html" })), /Unexpected content-type/, "rejects non-PDF content types");

assertThrowsMatching(() => validateSlideDeckResponse(responseWithHeaders({ "content-type": "application/pdf", "content-length": "12x" })), /Unexpected content-length/, "rejects malformed content length");

assertThrowsMatching(() => validateSlideDeckResponse(responseWithHeaders({ "content-type": "application/pdf", "content-length": String(51 * 1024 * 1024) })), /exceeds limit/, "rejects oversized downloads before reading the body");

const pdfBytes = Buffer.from("%PDF-1.4\n%%EOF\n");
assert.strictEqual(validatePdfBytes(pdfBytes, "test PDF"), pdfBytes);
console.log("PASS: accepts validated PDF bytes");

assertThrowsMatching(() => validatePdfBytes(Buffer.from("<html></html>"), "test PDF"), /not a real PDF/, "rejects non-PDF bytes");

assert.equal(buildSlideDeckUrl("github/gh-aw", safeSha), `https://media.githubusercontent.com/media/github/gh-aw/${safeSha}/docs/slides/github-agentic-workflows.pdf`);
console.log("PASS: builds trusted slide deck URL");

assertThrowsMatching(() => buildSlideDeckUrl("github/../gh-aw", safeSha), /Unsafe repository path/, "rejects unsafe repository path");

assertThrowsMatching(() => buildSlideDeckUrl("github/gh-aw", "main"), /Unsafe git ref/, "rejects non-SHA git ref");

console.log("All ensure-docs-slide-pdf tests passed.");
