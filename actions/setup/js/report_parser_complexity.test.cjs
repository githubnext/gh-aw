// @ts-check

import { describe, it, expect } from "vitest";
import { createRequire } from "node:module";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { executionEnv } from "./test_execution_provenance.js";

const require = createRequire(import.meta.url);
const { parseReportMarkdown } = require("./report_markdown.cjs");
const { neutralizeMarkdownLinkTitles } = require("./sanitize_content_core.cjs");
const directory = fileURLToPath(new URL(".", import.meta.url));

describe("report parser delimiter fidelity", () => {
  it.each([
    ["[outer [inner]](https://github.com/run(a))", "https://github.com/run(a)"],
    ["[escaped \\] label](https://github.com/a\\(b\\))", "https://github.com/a(b)"],
    ["prefix \\[literal] [run](https://github.com/a)", "https://github.com/a"],
    ["[broken]( \n[run](https://github.com/a)", "https://github.com/a"],
    ["[broken\n[run](https://github.com/a)", "https://github.com/a"],
    ["[broken](https://github.com/x \n[run](https://github.com/a)", "https://github.com/a"],
    ["[label \\\ncontinued](https://github.com/a)", "https://github.com/a"],
  ])("preserves valid links around escaped and incomplete delimiters: %s", (source, url) => {
    const blocks = parseReportMarkdown(source, { agentTainted: true });
    const nodes = blocks.flatMap(block => ("children" in block ? block.children : []));
    expect(nodes).toContainEqual(expect.objectContaining({ type: "link", url, agentTainted: true }));
  });
});

describe("bounded link title normalization", () => {
  // The former patterns are a compatibility oracle only for short fuzz inputs.
  function reference(source) {
    return source
      .replace(/(\[)([^\]]*)(\]\()(?:(<[^>]*>)|([^\s)]+))\s+(?:"([^"]*)"|'([^']*)'|\(([^)]*)\))(\s*\))/g, (_match, bracket, label, separator, angle, bare, double, single, parenthesized, close) => {
        const title = double !== undefined ? double : single !== undefined ? single : parenthesized;
        return `${bracket}${label} (${title})${separator}${angle !== undefined ? angle : bare}${close}`;
      })
      .replace(/^(\[[^\]]+\]:\s+\S+)\s+(?:"[^"]*"|'[^']*'|\([^)]*\))\s*$/gm, "$1");
  }

  it.each([
    '[outer [label](<url[brackets]> "visible")',
    "[label](url 'visible')",
    "[label](url (visible))",
    '[label](<url with spaces> "visible"\n)',
    '[label](<url "visible")',
    '[label](url "unclosed\n[next](url "visible")',
    '[outer\n[reference]: url "hidden"\n\nNext',
    "[reference]:\nurl 'hidden'\r\n   Next",
    "[reference]: url (hidden)\u2028\u2029  Next",
    '[reference]: url "hidden"\n  \n  ',
    "[]: url 'not a reference'",
    '[reference]: url "hidden" suffix',
  ])("preserves title visibility and reference stripping: %s", source => {
    expect(neutralizeMarkdownLinkTitles(source)).toBe(reference(source));
  });

  it.each([0x29009188, 0x18829290, 0x12345678, 0xabcdef01])("matches existing title security behavior for 1000 seeded mutations: %i", seed => {
    const atoms = ["[", "]", "(", ")", ":", "<", ">", "!", "\\", "\n", "\r", "\u2028", "\u2029", " ", "\t", "a", '"', "'", "https://github.com/a", '[a](url "title")', '[a]: url "title"', "![x](<a> (title))"];
    function next(maximum) {
      seed ^= seed << 13;
      seed ^= seed >>> 17;
      seed ^= seed << 5;
      return (seed >>> 0) % maximum;
    }
    for (let index = 0; index < 1000; index++) {
      let source = "";
      for (let part = 0, count = 1 + next(30); part < count; part++) source += atoms[next(atoms.length)];
      expect(neutralizeMarkdownLinkTitles(source), `seed case ${index}: ${JSON.stringify(source)}`).toBe(reference(source));
    }
  });
});

describe("publication complexity at the sanitizer input limit", () => {
  const worker = `
    const assert = require("node:assert/strict");
    const { performance } = require("node:perf_hooks");
    const { Parser } = require("commonmark");
    global.core = { info() {}, warning() {}, debug() {} };
    global.context = { repo: { owner: "test-owner", repo: "test-repo" }, runId: 12345, serverUrl: "https://github.com", payload: {} };
    const { sanitizeContent } = require("./sanitize_content.cjs");
    const { formatIssueBody } = require("./issue_body.cjs");
    const limit = 524288;
    const url = "https://github.com/a";
    const patterns = {
      brackets: "[".repeat(limit),
      escaped: "\\\\[".repeat(limit / 2),
      destinations: "[a](".repeat(limit / 4),
      angles: "[a](<".repeat(Math.ceil(limit / 5)).slice(0, limit),
      titles: '[a](u "'.repeat(Math.ceil(limit / 7)).slice(0, limit),
      references: ("[" + "a".repeat(510) + "\\n").repeat(limit / 512),
      punctuation: url + ")".repeat(limit - url.length),
      mixed: url + ")]}!?;:".repeat(Math.ceil(limit / 7)).slice(0, limit - url.length),
    };
    const source = patterns[process.argv[1]];
    const start = performance.now();
    const body = formatIssueBody(sanitizeContent(source), { repo: context.repo, footer: false });
    assert(body.length <= 65536);
    const walker = new Parser().parse(body).walker();
    let event;
    while ((event = walker.next())) {
      if (event.entering && ["link", "image"].includes(event.node.type)) {
        assert.equal(new URL(event.node.destination).hostname, "github.com");
        assert.equal(new URL(event.node.destination).protocol, "https:");
      }
    }
    console.log(JSON.stringify({ sourceLength: source.length, bodyLength: body.length, elapsedMs: performance.now() - start }));
  `;

  it.each(["brackets", "escaped", "destinations", "angles", "titles", "references", "punctuation", "mixed"])(
    "bounds %s without weakening sanitization or size fitting",
    pattern => {
      const result = spawnSync(process.execPath, ["-e", worker, pattern], {
        cwd: directory,
        encoding: "utf8",
        timeout: 8000,
        env: {
          ...process.env,
          ...executionEnv,
          GH_AW_ALLOWED_DOMAINS: "github.com",
          GH_AW_SAFE_OUTPUTS_URLS: "allowed-or-code-region",
          GH_AW_SAFE_OUTPUT_MESSAGES: "",
          GH_AW_DETECTION_CONCLUSION: "",
          GH_AW_WORKFLOW_NAME: "Worker",
          GH_AW_WORKFLOW_SOURCE: "",
          GH_AW_WORKFLOW_SOURCE_URL: "",
          GH_AW_TRACKER_ID: "",
          GH_AW_SECRET_NAMES: "",
        },
      });
      expect(result.error, result.stderr).toBeUndefined();
      expect(result.status, result.stderr).toBe(0);
      const measured = JSON.parse(result.stdout);
      expect(measured.sourceLength).toBe(524288);
      expect(measured.bodyLength).toBeLessThanOrEqual(65536);
      expect(measured.elapsedMs).toBeLessThan(8000);
    },
    10000
  );
});
