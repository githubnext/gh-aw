// @ts-check

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";
import { Parser } from "commonmark";
import { useExecutionProvenanceFixture, expectExecutionProvenance } from "./test_execution_provenance.js";

const require = createRequire(import.meta.url);
const { normalizeReportText, renderReport } = require("./report_ast.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { clearRedactedDomains } = require("./sanitize_content_core.cjs");
const { formatIssueBody } = require("./issue_body.cjs");
const parse = source => ({ type: "report", title: "Review boundaries", agentTainted: true, children: parseReportMarkdown(source, { agentTainted: true }) });
const publish = (source, options = {}) => formatIssueBody(sanitizeContent(normalizeReportText(source, { preserveImageAttributes: true }), options), { repo: context.repo });

useExecutionProvenanceFixture();

beforeEach(() => {
  vi.stubGlobal("core", { info: vi.fn(), warning: vi.fn(), debug: vi.fn() });
  vi.stubGlobal("context", { repo: { owner: "test-owner", repo: "test-repo" }, runId: 12345, serverUrl: "https://github.com", payload: {} });
  vi.stubEnv("GH_AW_ALLOWED_DOMAINS", "github.com");
  vi.stubEnv("GH_AW_ALLOWED_GITHUB_REFS", "repo");
  vi.stubEnv("GH_AW_SAFE_OUTPUTS_URLS", "allowed-only");
  vi.stubEnv("GH_AW_WORKFLOW_NAME", "Worker");
  vi.stubEnv("GH_AW_SAFE_OUTPUT_MESSAGES", "");
  vi.stubEnv("GH_AW_DETECTION_CONCLUSION", "");
  vi.stubEnv("GH_AW_WORKFLOW_SOURCE", "");
  vi.stubEnv("GH_AW_WORKFLOW_SOURCE_URL", "");
  vi.stubEnv("GH_AW_SECRET_NAMES", "");
});

afterEach(() => {
  clearRedactedDomains();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("image attribute boundaries before decoding", () => {
  it.each([
    ["Say &quot;hello&quot; &amp; goodbye", 'Say "hello" & goodbye'],
    ["Say &#34;hello&#34; &amp; goodbye", 'Say "hello" & goodbye'],
    ["Say &#x22;hello&#X22; &amp; goodbye", 'Say "hello" & goodbye'],
    ["It&apos;s a mascot", "It's a mascot"],
    ["It&#39;s a mascot", "It's a mascot"],
    ["It&#x27;s a mascot", "It's a mascot"],
    ["&#34; width=&#34;120", '" width="120'],
    ["&#x22; height=&#x22;80", '" height="80'],
    ["&amp;#34; align=&amp;#34;right", '" align="right'],
  ])("retains decoded attribute data without creating attributes: %s", (encoded, alt) => {
    const source = `<img src="https://github.com/image.png?a=1&amp;b=2" alt="${encoded}">`;
    for (const fragment of [source, sanitizeContent(source), sanitizeContent(source, { allowedAliases: ["copilot"] })]) {
      const input = parse(fragment);
      expect(input.children[0].children).toEqual([{ type: "image", url: "https://github.com/image.png?a=1&b=2", alt, agentTainted: true }]);
      const body = renderReport(input).body;
      expect(renderReport(parse(body)).body).toBe(body);
    }
    const body = publish(source);
    expectExecutionProvenance(body);
    expect(body).not.toContain("<img ");
    expect(body).toContain("https://github.com/image.png?a=1&b=2");
  });

  it.each(["&#39;", "&#x27;", "&apos;"])("preserves single-quoted attribute boundaries: %s", entity => {
    const source = `<img src='https://github.com/image.png' alt='It${entity}s fine' width='120'>`;
    expect(parse(sanitizeContent(source)).children[0].children[0]).toMatchObject({ type: "image", alt: "It's fine", width: 120 });
    const body = publish(source);
    expect(body).toContain('alt="It&#39;s fine" width="120"');
    expect(renderReport(parse(renderReport(parse(source)).body)).body).toBe(renderReport(parse(source)).body);
  });

  it("normalizes unquoted values as data before decoded quotes or spaces can create attributes", () => {
    const source = "<img src=https://github.com/image.png alt=&#34;&#32;width&#61;&#34;120>";
    for (const fragment of [source, sanitizeContent(source)]) {
      expect(parse(fragment).children[0].children).toEqual([{ type: "image", url: "https://github.com/image.png", alt: '" width="120', agentTainted: true }]);
    }
    expect(publish(source)).not.toContain("<img ");
  });

  it("redacts decoded quote-bearing secrets and complete formatting fragments", () => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "REVIEW_BOUNDARY");
    vi.stubEnv("SECRET_REVIEW_BOUNDARY", 'review-"secret"-fixture');
    const source = '<img src="https://github.com/image.png" alt="review-&#34;secret&#34;-fixture" width="120">\n\nreview-**"secret"**-fixture';
    for (const body of [renderReport(parse(source)).body, publish(source)]) {
      expect(body).not.toContain("secret");
      expect(body).toContain("***");
    }
  });

  it.each(["data&#58;image/png;base64,payload", "https://github.com&#64;outside.invalid/image.png", "https://outside.invalid/image.png"])("keeps decoded destinations subject to image policy: %s", url => {
    const source = `<img src="${url}" alt="Say &#34;hello&#34;" width="120">`;
    expect(renderReport(parse(source)).body).not.toContain("<img ");
    expect(publish(source)).not.toContain("<img ");
  });
});

describe("fence detection agrees during paragraph collection", () => {
  it.each(["```", "~~~"])("keeps %s examples literal after prose for all valid indents", fence => {
    for (const indent of ["", " ", "  ", "   "]) {
      const example = '<img src="https://github.com/image.png" width="120" alt="literal">';
      const source = `Prose\n${indent}${fence}html\n${example}\n${indent}${fence}\nAfter`;
      const input = parse(source);
      expect(input.children.map(node => node.type)).toEqual(["paragraph", "code_block", "paragraph"]);
      expect(input.children[1]).toMatchObject({ language: "html", value: example });
      const walker = new Parser().parse(publish(source)).walker();
      let event;
      const code = [];
      while ((event = walker.next())) {
        if (event.node.type === "code_block") code.push(event.node.literal);
        if (event.node.type.startsWith("html_")) expect(event.node.literal).not.toContain("<img ");
      }
      expect(code).toContain(example + "\n");
    }
  });

  it("does not treat invalid backtick info strings as paragraph-breaking fences", () => {
    const input = parse("Prose\n  ```bad`info\nAfter");
    expect(input.children).toHaveLength(1);
    expect(input.children[0].type).toBe("paragraph");
  });
});

describe("policy checks after supported and unsupported HTML", () => {
  it.each(["allowed-only", "allowed-or-code-region"])("rechecks active destinations and references under %s", policy => {
    vi.stubEnv("GH_AW_SAFE_OUTPUTS_URLS", policy);
    for (const tag of ['<b title="`">x</b>', '<img src="https://github.com/image.png" width="120" alt="`">']) {
      const source = `${tag} [click](https://outside.invalid/x) ![p](https://outside.invalid/pixel.png) other/repo#123 Fixes other/repo#456 \``;
      const body = publish(source);
      expect(body).not.toContain("https://outside.invalid");
      expect(body).not.toMatch(/(?:^|\s)other\/repo#123(?:\s|$)/);
      expect(body).not.toMatch(/(?:^|\s)Fixes other\/repo#456(?:\s|$)/);
      const walker = new Parser().parse(body).walker();
      let event;
      const code = [];
      while ((event = walker.next())) {
        if (event.node.type === "code") code.push(event.node.literal);
        if (["link", "image"].includes(event.node.type)) expect(new URL(event.node.destination).hostname).toBe("github.com");
      }
      expect(code.join(" ")).toContain("other/repo#123");
      expect(code.join(" ")).toContain("other/repo#456");
    }
  });

  it("preserves allowed cross-repository references when local references are forbidden", () => {
    vi.stubEnv("GH_AW_ALLOWED_GITHUB_REFS", "allowed/repo");
    const body = renderReport(parse("allowed/repo#123 Fixes allowed/repo#456 #789")).body;
    expect(body).toContain("allowed/repo#123 Fixes allowed/repo#456");
    expect(body).toContain("`#789`");
  });
});
