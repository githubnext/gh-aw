// @ts-check

import { describe, it, expect, vi, afterEach } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { normalizeReport, normalizeReportTitle, renderReport } = require("./report_ast.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");

/** @param {import("./report_ast.cjs").ReportBlock[]} children */
const report = children => ({ type: /** @type {const} */ "report", title: " [aw] Smoke:\r\nfailed ", children });
const paragraph = value => ({ type: /** @type {const} */ "paragraph", children: [{ type: /** @type {const} */ "text", value }] });

afterEach(() => vi.unstubAllEnvs());

describe("report AST", () => {
  it("normalizes deterministically without mutating the input", () => {
    const input = report([
      paragraph(""),
      { type: "heading", level: 1, children: [{ type: "text", value: "What happened" }] },
      { type: "details", summary: " Runtime\n evidence ", children: [paragraph("  e\u0301vidence\r\nhere  ")] },
      { type: "details", summary: "Empty", children: [] },
    ]);
    const copy = structuredClone(input);
    const normalized = normalizeReport(input);
    expect(input).toEqual(copy);
    expect(normalizeReport(normalized)).toEqual(normalized);
    expect(renderReport(input)).toEqual({
      title: "[aw] Smoke: failed",
      body: "## What happened\n\n<details>\n<summary>Runtime evidence</summary>\n\n\u00e9vidence here\n\n</details>\n",
    });
  });

  it("escapes literal metadata and HTML rather than allowing new sections or mentions", () => {
    const { body } = renderReport(report([paragraph("# injected\n</details><img src=x> **bold** [click](javascript:alert(1)) @octocat &amp; text")]));
    expect(body).not.toContain("\n# injected");
    expect(body).not.toContain("<img");
    expect(body).not.toContain("</details>");
    expect(body).toContain("\\# injected &lt;/details&gt;&lt;img src=x&gt;");
    expect(body).toContain("\\*\\*bold\\*\\*");
    expect(body).toContain("\\[click\\]");
    expect(body).toContain("`@octocat` &amp; text");
  });

  it("chooses fences after normalization and redaction", () => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "REPORT_TEST");
    vi.stubEnv("SECRET_REPORT_TEST", "report-secret-fixture");
    const { body } = renderReport(
      report([
        { type: "code_block", language: "text", value: "report-secret-fixture\r\n&#96;&#96;&#96;&#96;\n</details>\n# injected" },
        { type: "paragraph", children: [{ type: "code", value: "`edge`" }] },
      ])
    );
    expect(body).not.toContain("report-secret-fixture");
    expect(body).toContain("`````text\n");
    expect(body).toContain("\n````\n</details>\n# injected\n`````");
    expect(body).toContain("`` `edge` ``");
  });

  it("escapes HTML summaries without Markdown-specific backslashes", () => {
    const { body } = renderReport(report([{ type: "details", summary: 'Evidence [1] </summary><img src="x"> & data', children: [paragraph("details")] }]));
    expect(body).toContain('<summary>Evidence [1] &lt;/summary&gt;&lt;img src="x"&gt; &amp; data</summary>');
  });

  it("renders only explicit links and encodes Markdown destination delimiters", () => {
    const { body } = renderReport(report([{ type: "paragraph", children: [{ type: "link", url: "https://example.com/run(a)?q='value'", children: [{ type: "text", value: "Run [1]" }] }] }]));
    expect(body).toBe("[Run \\[1\\]](https://example.com/run%28a%29?q=%27value%27)\n");
  });

  it.each(["javascript:alert(1)", "data:text/html,test", "file:///tmp/test", "http://example.com", "https://user:password@example.com", "https://example.com/\n#injected", "//example.com"])("rejects unsafe link %s", url => {
    expect(() => renderReport(report([{ type: "paragraph", children: [{ type: "link", url, children: [{ type: "text", value: "Run" }] }] }]))).toThrow();
  });

  it("normalizes lists, alerts, tables, and nested heading levels", () => {
    const { body } = renderReport(
      report([
        { type: "list", ordered: true, items: [[paragraph("First")], [], [paragraph("Second")]] },
        { type: "alert", kind: "WARNING", children: [paragraph("Failure"), { type: "heading", level: 1, children: [{ type: "text", value: "Evidence" }] }] },
        { type: "table", headers: [[{ type: "text", value: "Tool" }]], rows: [[[{ type: "code", value: "a|b" }]], [[{ type: "text", value: "c|d" }]]] },
      ])
    );
    expect(body).toContain("1. First\n2. Second");
    expect(body).toContain("> [!WARNING]\n> Failure\n>\n> ### Evidence");
    expect(body).toContain("| `a\\|b` |");
    expect(body).toContain("| c\\|d |");
  });

  it("rejects raw nodes, malformed tables, invalid languages, and excessive nesting", () => {
    expect(() => renderReport(report([/** @type {never} */ { type: "raw", value: "<img>" }]))).toThrow("Unsupported report block");
    expect(() => renderReport(report([{ type: "table", headers: [[{ type: "text", value: "One" }]], rows: [[]] }]))).toThrow("header width");
    expect(() => renderReport(report([{ type: "code_block", language: "text\n</details>", value: "payload" }]))).toThrow("code language");
    expect(() => renderReport(report([{ type: "alert", kind: "INVALID", children: [paragraph("data")] }]))).toThrow("alert kind");
    const children = [];
    children.push({ type: "details", summary: "Cycle", children });
    expect(() => renderReport(report(children))).toThrow("maximum depth");
  });

  it("normalizes and bounds titles without Markdown escaping or broken Unicode", () => {
    expect(normalizeReportTitle("[aw]\u202e Smoke\r\n[model]:\x1b[31m failed\x1b[0m")).toBe("[aw] Smoke [model]: failed");
    expect(Array.from(normalizeReportTitle("\u{1f680}".repeat(300)))).toHaveLength(256);
  });

  it("stabilizes entity decoding, merged inline whitespace, and secrets exposed by controls", () => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "REPORT_TEST");
    vi.stubEnv("SECRET_REPORT_TEST", "report-secret-fixture");
    const input = report([
      {
        type: "paragraph",
        children: [
          { type: "text", value: "&amp;amp;amp;amp;amp;lt;tag&amp;amp;amp;amp;amp;gt; " },
          { type: "text", value: " report-secret-\u0000fixture" },
          { type: "code", value: "one " },
          { type: "code", value: " two" },
        ],
      },
    ]);
    const normalized = normalizeReport(input);
    expect(normalizeReport(normalized)).toEqual(normalized);
    expect(renderReport(input).body).not.toContain("report-secret-fixture");
    expect(renderReport(input).body).toContain("&lt;tag&gt;");
    expect(renderReport(input).body).toContain("`one two`");
  });

  it("escapes ordered-list syntax in literal paragraphs and refuses nested links", () => {
    expect(renderReport(report([paragraph("1. not a list")])).body).toBe("1\\. not a list\n");
    expect(() =>
      renderReport(
        report([
          {
            type: "paragraph",
            children: [{ type: "link", url: "https://example.com", children: [{ type: "link", url: "https://example.org", children: [{ type: "text", value: "nested" }] }] }],
          },
        ])
      )
    ).toThrow("cannot contain other links");
  });
});

describe("report Markdown import boundary", () => {
  it("parses template structure before substituting hostile scalar values", () => {
    const children = parseReportMarkdown("## What happened\n\n**{workflow}** failed.\n\n**Branch:** `{branch}`\n\n[Run]({url})", {
      minimumHeadingLevel: 2,
      context: { workflow: "Smoke\n\n## Injected\n[evil](javascript:x)", branch: "`\n</details>", url: "https://example.com/run(test)" },
    });
    const { body } = renderReport(report(children));
    expect(body.match(/^## /gm)).toEqual(["## "]);
    expect(body).toContain("Smoke \\#\\# Injected");
    expect(body).toContain("`` ` </details> ``");
    expect(body).toContain("[Run](https://example.com/run%28test%29)");
  });

  it("preserves nested details, code, alerts, lists, tables, and inline formatting", () => {
    const markdown = [
      "<details>",
      "<summary>Runtime</summary>",
      "",
      "> [!WARNING]",
      "> **Missing tool:** `github.read`",
      "",
      "### Evidence",
      "",
      "- First",
      "  - Nested",
      "- Second",
      "",
      "| Tool | Result |",
      "| --- | --- |",
      "| `gh` | *missing* |",
      "",
      "````text",
      "</details>",
      "```",
      "````",
      "",
      "</details>",
    ].join("\n");
    const { body } = renderReport(report(parseReportMarkdown(markdown)));
    expect(body).toContain("> [!WARNING]\n> **Missing tool:** `github.read`");
    expect(body).toContain("- First\n  \n  - Nested\n- Second");
    expect(body).toContain("| `gh` | *missing* |");
    expect(body).toContain("````text\n</details>\n```\n````");
    expect(body.endsWith("\n</details>\n")).toBe(true);
  });

  it("does not promote diagnostic headings to report sections or permit arbitrary HTML", () => {
    const { body } = renderReport(report(parseReportMarkdown("## Injected\n\n<img src=x>\n\n</details>\n\n[unsafe](javascript:alert)\n\n[Run](https://example.com) · [Source](https://example.org)")));
    expect(body).toContain("### Injected");
    expect(body).not.toContain("<img");
    expect(body).not.toContain("\n</details>");
    expect(body).toContain("link unavailable: invalid HTTPS URL");
    expect(body).toContain("[Run](https://example.com/) · [Source](https://example.org/)");
  });

  it("imports block fields explicitly and refuses missing fields or blocks in scalar positions", () => {
    expect(() => parseReportMarkdown("{missing}", { context: {} })).toThrow("Missing report template field");
    expect(() => parseReportMarkdown("{toString}", { context: {} })).toThrow("Missing report template field");
    expect(() => parseReportMarkdown("**{blocks}**", { context: { blocks: [paragraph("data")] } })).toThrow("cannot appear inline");
    const { body } = renderReport(report(parseReportMarkdown("{one}{two}", { context: { one: [paragraph("First")], two: [] } })));
    expect(body).toBe("First\n");
  });

  it("bounds recursive details and preserves unmatched inline delimiters as literal text", () => {
    expect(() => parseReportMarkdown("<details>\n<summary>Nested</summary>\n".repeat(20) + "payload")).toThrow("maximum depth");
    const { body } = renderReport(report(parseReportMarkdown("An unmatched `` delimiter and a_b_c identifier")));
    expect(body).toContain("An unmatched \\`\\` delimiter and a\\_b\\_c identifier");
  });
});
