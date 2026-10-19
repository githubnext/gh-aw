// @ts-check

import { afterEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { renderSizedReport, fitMarkdownBody, SIZE_NOTICE } = require("./report_size.cjs");
const { renderReport } = require("./report_ast.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");
/** @param {string} value @param {boolean} [agentTainted] @returns {import("./report_ast.cjs").ReportBlock} */
const paragraph = (value, agentTainted = true) => ({ type: "paragraph", agentTainted, children: [{ type: "text", value, agentTainted }] });
/** @param {import("./report_ast.cjs").ReportBlock[]} children @returns {import("./report_ast.cjs").Report} */
const report = children => ({ type: "report", title: "Report", children });

afterEach(() => vi.unstubAllEnvs());

describe("automatic report size redaction", () => {
  it("preserves exact-boundary reports and replaces over-limit text without slicing it", () => {
    const input = report([paragraph("x".repeat(99))]);
    expect(renderSizedReport(input, 100).body).toHaveLength(100);
    const fitted = renderSizedReport(report([paragraph("x".repeat(100))]), 100);
    expect(fitted.body).toBe(`${SIZE_NOTICE}\n`);
    expect(fitted.body.length).toBeLessThanOrEqual(100);
  });

  it("measures escaped output rather than source text length", () => {
    const input = report([paragraph("<x>".repeat(20))]);
    expect(renderReport(input).body.length).toBeGreaterThan(100);
    expect(renderSizedReport(input, 100).body).toBe(`${SIZE_NOTICE}\n`);
  });

  it("preserves framework labels while replacing large agent inline slots", () => {
    const input = report([
      {
        type: "paragraph",
        agentTainted: false,
        children: [
          { type: "text", value: "Framework label: ", agentTainted: false },
          { type: "code", value: "x".repeat(1000), agentTainted: true },
        ],
      },
    ]);
    expect(renderSizedReport(input, 100).body).toBe(`Framework label: ${SIZE_NOTICE}\n`);
  });

  it("preserves markers inside agent containers and reserves the entire footer", () => {
    const marker = "<!-- custom: keep-all-marker-content -->";
    const footer = {
      type: /** @type {const} */ "footer",
      agentTainted: false,
      children: [paragraph("Footer diagnostic " + "f".repeat(100), true)],
      markers: [{ type: /** @type {const} */ "xml_marker", name: "gh-aw-agentic-workflow", value: "Workflow", fields: [] }],
    };
    const input = { ...report([{ type: "details", summary: "Evidence", agentTainted: true, children: [paragraph("x".repeat(1000)), { type: "xml_marker", name: "custom", value: "keep-all-marker-content", agentTainted: true }] }]), footer };
    const copy = structuredClone(input);
    const protectedFooter = renderReport({ ...report([]), footer }).body.trim();
    const body = renderSizedReport(input, 400).body;
    expect(body.length).toBeLessThanOrEqual(400);
    expect(body).toContain(marker);
    expect(body).toContain(protectedFooter);
    expect(body).toContain("<summary>Evidence</summary>");
    expect(input).toEqual(copy);
  });

  it("never shortens protected footer diagnostics embedded in the main tree", () => {
    const diagnostic = { ...paragraph("Footer diagnostic " + "d".repeat(200)), sizeProtected: true };
    const body = renderSizedReport(report([paragraph("x".repeat(1000)), diagnostic]), 300).body;
    expect(body).toContain("Footer diagnostic " + "d".repeat(200));
    expect(body).toContain(SIZE_NOTICE);
  });

  it("coalesces thousands of small replacements without exceeding the budget", () => {
    const body = renderSizedReport(report(Array.from({ length: 1000 }, () => paragraph("x"))), 100).body;
    expect(body).toBe(`${SIZE_NOTICE}\n`);
  });

  it.each(["😀", "&amp;lt;", "` @someone "])("fits large %s input without broken encoding or fences", value => {
    const body = renderSizedReport(report([paragraph(value.repeat(1000))]), 100).body;
    expect(body).toBe(`${SIZE_NOTICE}\n`);
    expect(body).not.toMatch(/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/u);
  });

  it("fails explicitly when protected markers or footer content alone cannot fit", () => {
    expect(() => renderSizedReport(report([{ type: "xml_marker", name: "custom", value: "m".repeat(200), agentTainted: true }]), 100)).toThrow("protected markers, footers");
    expect(() => renderSizedReport({ ...report([paragraph("x".repeat(1000))]), footer: { type: "footer", children: [paragraph("f".repeat(200))], markers: [] } }, 100)).toThrow("cannot be reduced");
  });

  it("keeps structured data typed and intact when it fits, or replaces the entire node", () => {
    const data = { nested: [{ value: "x".repeat(1000), numeric: 42, boolean: false, empty: null }], text: "&quot; literal" };
    const input = report([{ type: "structured_data", data, agentTainted: true }]);
    expect(JSON.parse(renderSizedReport(input).body.split("\n").slice(1, -2).join("\n"))).toEqual(data);
    expect(renderSizedReport(input, 100).body).toBe(`${SIZE_NOTICE}\n`);
    expect(input.children[0]).toEqual({ type: "structured_data", data, agentTainted: true });
  });

  it("imports valid JSON fences as structured nodes, not truncatable text", () => {
    const nodes = parseReportMarkdown('```json\n{"items":[1,true,null,{"label":"&quot;"}]}\n```', { agentTainted: true });
    expect(nodes).toEqual([{ type: "structured_data", data: { items: [1, true, null, { label: "&quot;" }] }, agentTainted: true }]);
    expect(parseReportMarkdown("```json\n{invalid}\n```")[0].type).toBe("code_block");
    expect(parseReportMarkdown('```text\n{"value":1}\n```')[0].type).toBe("code_block");
  });

  it.each([null, true, 42, "literal &quot;", [1, true, null]])("keeps JSON value %j typed", data => {
    expect(parseReportMarkdown("```json\n" + JSON.stringify(data) + "\n```")[0]).toEqual({ type: "structured_data", data });
  });

  it("does not promote data fields into trusted provenance or size protection", () => {
    const children = parseReportMarkdown('```json\n{"agentTainted":false,"sizeProtected":true,"payload":"' + "x".repeat(1000) + '"}\n```', { agentTainted: true });
    expect(children[0].agentTainted).toBe(true);
    expect(children[0]).not.toHaveProperty("sizeProtected");
    expect(renderSizedReport(report(children), 100).body).toBe(`${SIZE_NOTICE}\n`);
  });

  it("does not change JSON metadata into size-redactable structured payloads", () => {
    const nodes = parseReportMarkdown('<!-- aw-essential-meta: {"payload":"' + "x".repeat(200) + '"} -->', { allowMetadata: true, agentTainted: true });
    expect(nodes[0].type).toBe("json_marker");
    expect(() => renderSizedReport(report(nodes), 100)).toThrow("cannot be reduced");
  });

  it("retains secret redaction before fitting and rechecks the final notice output", () => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "SIZING_SECRET");
    vi.stubEnv("SECRET_SIZING_SECRET", "size-fitting-secret-fixture");
    const body = renderSizedReport(report([paragraph("size-fitting-secret-fixture " + "x".repeat(1000))]), 100).body;
    expect(body).toBe(`${SIZE_NOTICE}\n`);
    expect(body).not.toContain("size-fitting-secret-fixture");
  });

  it("does not let size protection bypass secret redaction", () => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "SIZING_SECRET");
    vi.stubEnv("SECRET_SIZING_SECRET", "size-fitting-secret-fixture");
    const protectedNode = { ...paragraph("size-fitting-secret-fixture", false), sizeProtected: true };
    const body = renderSizedReport(report([protectedNode])).body;
    expect(body).not.toContain("size-fitting-secret-fixture");
    expect(body).toContain("\\*\\*\\*");
  });

  it("reserves literal prefix and footer strings without ellipsing them", () => {
    const prefix = "<details>\n<summary>Framework</summary>\n\n";
    const suffix = "\n\n</details>\nFooter in full\n<!-- marker: keep -->";
    const fitted = fitMarkdownBody("x".repeat(1000), { prefix, suffix, maxLength: 200 });
    expect(fitted).toBe(prefix + SIZE_NOTICE + suffix);
    expect(fitted.length).toBeLessThanOrEqual(200);
    expect(fitMarkdownBody("**literal**", { prefix, suffix, maxLength: 200 })).toBe(prefix + "**literal**" + suffix);
    expect(() => fitMarkdownBody("large", { suffix: "f".repeat(100), maxLength: 100 })).toThrow("Protected markers and footers");
    expect(fitMarkdownBody("x".repeat(100), { maxLength: SIZE_NOTICE.length })).toBe(SIZE_NOTICE);
  });

  it.each([0, -1, 1.5, NaN, Infinity])("rejects invalid size budgets %s", maxLength => {
    expect(() => renderSizedReport(report([paragraph("Body")]), maxLength)).toThrow("positive integer");
  });
});
