// @ts-check

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";
import { spawnSync } from "node:child_process";
import { Parser } from "commonmark";

const require = createRequire(import.meta.url);
const { canonicalizePublicationText, clearRedactedDomains } = require("./sanitize_content_core.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");
const { renderReport } = require("./report_ast.cjs");
const url = "https://github.com/image.png";
// Exercise the private scanner through its production entry point.
const preserve = text => canonicalizePublicationText(text, { preserveImageAttributes: true });
const parse = source => ({ type: "report", title: "Attribute boundaries", agentTainted: true, children: parseReportMarkdown(source, { agentTainted: true }) });
const images = source =>
  parse(source)
    .children.flatMap(block => block.children ?? [])
    .filter(node => node.type === "image");
const escapeQuotes = text => text.replace(/"/g, "&quot;").replace(/'/g, "&apos;");

beforeEach(() => {
  vi.stubGlobal("core", { info: vi.fn(), warning: vi.fn(), debug: vi.fn() });
  vi.stubGlobal("context", { repo: { owner: "test-owner", repo: "test-repo" }, serverUrl: "https://github.com", payload: {} });
  vi.stubEnv("GH_AW_ALLOWED_DOMAINS", "github.com");
  vi.stubEnv("GH_AW_ALLOWED_GITHUB_REFS", "repo");
  vi.stubEnv("GH_AW_SAFE_OUTPUTS_URLS", "allowed-only");
  vi.stubEnv("GH_AW_SECRET_NAMES", "");
});

afterEach(() => {
  clearRedactedDomains();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("preserveImageAttributeQuotes canonicalization", () => {
  it.each([
    ["", ""],
    ["before < after", "before < after"],
    ['<img alt="plain" width="12">', '<img alt="plain" width="12">'],
    ["<img alt='say \"hi\"'>", "<img alt='say &quot;hi&quot;'>"],
    ['<img alt="it\'s fine">', '<img alt="it&apos;s fine">'],
    ["<IMG\tALT = '&#x22;' SRC = https://github.com/image.png />", "<IMG\tALT = '&quot;' SRC = https://github.com/image.png />"],
    ["<img alt=mascot>", "<img alt=mascot>"],
    ["<img alt=&quot;>", "<img alt=&quot;>"],
    ["<img alt=&#34;&#32;width&#61;&#34;120>", '<img alt="&quot; width=&quot;120">'],
    ["<img alt=\uff02>", '<img alt="&quot;">'],
    ["<img alt=a\u200bb>", '<img alt="ab">'],
    ["<img alt=&amp;amp;>", '<img alt="&">'],
    ["<img alt = \t &#34;>", '<img alt = \t "&quot;">'],
    ["<img alt=>", "<img alt=>"],
    ["<img alt=", "<img alt="],
    ["prefix <img alt=\"&#34;\"> middle <img alt='&#39;'> suffix", "prefix <img alt=\"&quot;\"> middle <img alt='&apos;'> suffix"],
    ['<img alt="a &gt; b &lt; c">', '<img alt="a > b < c">'],
    ['<img alt="&#34;"\twidth = 12 />', '<img alt="&quot;"\twidth = 12 />'],
  ])("has an explicit canonical result for %j", (source, expected) => {
    expect(preserve(source)).toBe(expected);
    expect(preserve(expected)).toBe(expected);
  });

  it.each(["image", "imgx", "img:extra", "img=", "/img", "div"])("does not apply image-specific quoting to <%s>", tag => {
    const source = `<${tag} alt="&#34;">`;
    expect(preserve(source)).toBe(`<${tag} alt=""">`);
    expect(preserve(source)).toBe(canonicalizePublicationText(source));
  });

  it("keeps ordinary canonicalization unchanged unless preservation is requested", () => {
    const source = '<img alt="&#34;"> &#34; &#39; &amp; \uff21\u200b';
    expect(canonicalizePublicationText(source)).toBe('<img alt="""> " \' & A');
    expect(preserve(source)).toBe('<img alt="&quot;"> " \' & A');
  });

  const encodings = [
    ["&#34;", '"'],
    ["&#x22;", '"'],
    ["&#X22;", '"'],
    ["&#00034;", '"'],
    ["&quot;", '"'],
    ["&amp;#34;", '"'],
    ["&amp;amp;#34;", '"'],
    ["&#38;#34;", '"'],
    ["&\u200b#34;", '"'],
    ["\uff02", '"'],
    ["&#39;", "'"],
    ["&#x27;", "'"],
    ["&#X27;", "'"],
    ["&#00039;", "'"],
    ["&apos;", "'"],
    ["&amp;#39;", "'"],
    ["&amp;amp;#39;", "'"],
    ["&#38;#39;", "'"],
    ["&\u200b#39;", "'"],
    ["\uff07", "'"],
  ];

  describe.each(['"', "'", ""])("attribute delimiter %j", delimiter => {
    it.each(encodings)("retains %s as data instead of creating attributes", (encoded, decoded) => {
      const raw = `${encoded}&#32;width&#61;${encoded}9999&#32;height&#61;${encoded}9998&#32;align&#61;${encoded}right`;
      const alt = `${decoded} width=${decoded}9999 height=${decoded}9998 align=${decoded}right`;
      const source = `<img src="${url}" alt=${delimiter}${raw}${delimiter}>`;
      const expected = `<img src="${url}" alt=${delimiter || '"'}${escapeQuotes(alt)}${delimiter || '"'}>`;
      expect(preserve(source)).toBe(expected);
      expect(preserve(expected)).toBe(expected);
      expect(images(source)).toEqual([{ type: "image", url, alt, agentTainted: true }]);
      for (const options of [{}, { allowedAliases: ["copilot"] }]) {
        expect(images(sanitizeContent(source, options))).toEqual([{ type: "image", url, alt, agentTainted: true }]);
      }
      expect(renderReport(parse(source)).body).not.toContain("<img ");
    });
  });

  it.each([
    `<img src="${url}" alt="unterminated`,
    `<img src="${url}" alt="first\nsecond">`,
    `<img src="${url}" alt="first\rsecond">`,
    `<img src="${url}" alt=bad"value>`,
    `<img src="${url}" alt=a=b>`,
    `<img src="${url}" alt=\`broken\`>`,
    `<img src="${url}" alt="first" alt="second">`,
    `<imgx src="${url}" alt="&#34;">`,
  ])("does not repair unsupported or malformed HTML into an image: %j", source => {
    for (const fragment of [preserve(source), sanitizeContent(source), sanitizeContent(source, { allowedAliases: ["copilot"] })]) {
      expect(images(fragment)).toEqual([]);
      expect(renderReport(parse(fragment)).body).not.toContain("<img ");
    }
  });

  it.each(['onerror="&#34;"', 'style="display:none"', 'title="&#39;"', 'data-hidden="&#34;"'])("does not preserve dangerous %s attributes during sanitization", attribute => {
    const source = `<img src="${url}" ${attribute}>`;
    expect(images(preserve(source))).toEqual([]);
    for (const options of [{}, { allowedAliases: ["copilot"] }]) {
      const sanitized = sanitizeContent(source, options);
      expect(images(sanitized)).toEqual([{ type: "image", url, alt: "", agentTainted: true }]);
      expect(renderReport(parse(sanitized)).body).toBe(`![](${url})\n`);
    }
  });

  it.each(["<img alt=broken", '<img alt="unclosed\n', '<img alt="unclosed\r', "<not-image>", "prose "])("resumes scanning after %j", prefix => {
    const source = `${prefix}<img src="${url}" alt="&#34;"> tail`;
    expect(preserve(source)).toBe(`${prefix}<img src="${url}" alt="&quot;"> tail`);
    expect(images(source)).toEqual([{ type: "image", url, alt: '"', agentTainted: true }]);
  });

  it.each(['"', "'", ""])("does not bypass image URL policy with %j attributes", delimiter => {
    for (const destination of ["data&#58;image/png;base64,fixture", "javascript&#58;void(0)", "https://outside.invalid/image.png", "https://github.com&#64;outside.invalid/image.png"]) {
      const source = `<img src=${delimiter}${destination}${delimiter} alt="&#34;" width="12">`;
      for (const fragment of [source, preserve(source), sanitizeContent(source), sanitizeContent(source, { allowedAliases: ["copilot"] })]) {
        expect(images(fragment)).toEqual([]);
        expect(renderReport(parse(fragment)).body).not.toContain("<img ");
      }
    }
  });

  it.each(['"', "'", ""])("keeps decoded secrets redacted with %j attributes", delimiter => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "ATTRIBUTE_TEST");
    vi.stubEnv("SECRET_ATTRIBUTE_TEST", 'attribute-"synthetic"-fixture');
    const source = `<img src="${url}" alt=${delimiter}attribute-&#34;synthetic&#34;-fixture${delimiter} width="12">`;
    for (const fragment of [source, sanitizeContent(source), sanitizeContent(source, { allowedAliases: ["copilot"] })]) {
      const body = renderReport(parse(fragment)).body;
      expect(body).toContain("***");
      expect(body).not.toContain("synthetic");
    }
  });

  it.each(['"', "'", ""])("normalizes legitimate layout values without weakening validation for %j attributes", delimiter => {
    const source = `<img src="${url}" width=${delimiter}&#49;&#50;${delimiter} height=${delimiter}\uff18\uff10${delimiter} align=${delimiter}R\u200bIGHT${delimiter}>`;
    expect(images(source)).toEqual([{ type: "image", url, alt: "", width: 12, height: 80, align: "right", agentTainted: true }]);
    for (const field of ["width", "height", "align"]) {
      const invalid = `<img src="${url}" ${field}=${delimiter}12&#34;&#32;width&#61;&#34;120${delimiter}>`;
      expect(images(invalid)).toEqual([]);
      expect(images(sanitizeContent(invalid))).toEqual([]);
    }
  });

  it.each(['"', "'", ""])("fails closed for excessive encoding depth in %j attributes", delimiter => {
    const nested = `&${"amp;".repeat(80)}#34;`;
    expect(() => preserve(`<img alt=${delimiter}${nested}${delimiter}>`)).toThrow("Publication text contains excessively nested encodings");
  });
});

describe("seeded image-attribute boundary mutations", () => {
  it.each([0x18829290, 0x29009188, 0x12345678])("preserves attribute data and typed layout across 400 mutations with seed %i", seed => {
    const atoms = [
      ["safe", "safe"],
      ["&#34;", '"'],
      ["&quot;", '"'],
      ["&#x27;", "'"],
      ["&apos;", "'"],
      ["&amp;amp;#34;", '"'],
      ["&#32;", " "],
      ["&#61;", "="],
      ["&lt;", "<"],
      ["&gt;", ">"],
      ["&amp;", "&"],
      ["&#96;", "`"],
      ["\uff02", '"'],
      ["\uff07", "'"],
      ["\uff21", "A"],
      ["\u200b", ""],
    ];
    function next(maximum) {
      seed ^= seed << 13;
      seed ^= seed >>> 17;
      seed ^= seed << 5;
      return (seed >>> 0) % maximum;
    }
    for (let index = 0; index < 400; index++) {
      const delimiter = ['"', "'", ""][next(3)];
      let raw = "start";
      let alt = "start";
      for (let part = 0, count = 1 + next(12); part < count; part++) {
        const [encoded, decoded] = atoms[next(atoms.length)];
        raw += encoded;
        alt += decoded;
      }
      const source = `<img src="${url}" alt=${delimiter}${raw}${delimiter} width="7" height="9" align="right">`;
      const normalized = preserve(source);
      const expected = { type: "image", url, alt: alt.replace(/\s+/g, " "), width: 7, height: 9, align: "right", agentTainted: true };
      expect(images(normalized), `mutation ${index}: ${JSON.stringify(source)}`).toEqual([expected]);
      expect(preserve(normalized)).toBe(normalized);
      const body = renderReport(parse(normalized)).body;
      expect(renderReport(parse(body)).body).toBe(body);
      const walker = new Parser().parse(body).walker();
      const html = [];
      let event;
      while ((event = walker.next())) {
        if (event.entering && event.node.type.startsWith("html_")) html.push(event.node.literal);
        if (event.node.type === "link" || event.node.type === "image") throw new Error("Attribute data escaped into Markdown media");
      }
      expect(html).toEqual([body.trim()]);
    }
  });
});

describe("image-attribute scanner input-limit bounds", () => {
  it.each(["angles", "unterminated", "quoted", "unquoted"])(
    "bounds %s at 524288 input characters in an isolated process",
    pattern => {
      const worker = `
        const assert = require("node:assert/strict");
        const { canonicalizePublicationText } = require("./sanitize_content_core.cjs");
        const limit = 524288;
        const pattern = process.argv[1];
        let source, expected;
        if (pattern === "angles") source = expected = "<".repeat(limit);
        if (pattern === "unterminated") source = expected = '<img alt="' + "x".repeat(limit - 10);
        if (pattern === "quoted") {
          const count = limit - '<img alt="">'.length;
          source = '<img alt="' + "\\uff02".repeat(count) + '">';
          expected = '<img alt="' + "&quot;".repeat(count) + '">';
        }
        if (pattern === "unquoted") {
          const token = "<img alt=&#34;>";
          const count = Math.floor(limit / token.length);
          const tail = "x".repeat(limit % token.length);
          source = token.repeat(count) + tail;
          expected = '<img alt="&quot;">'.repeat(count) + tail;
        }
        assert.equal(source.length, limit);
        const result = canonicalizePublicationText(source, { preserveImageAttributes: true });
        assert.equal(result, expected);
        assert(result.length <= 6 * limit);
        assert.equal(canonicalizePublicationText(result, { preserveImageAttributes: true }), result);
      `;
      const result = spawnSync(process.execPath, ["-e", worker, pattern], { cwd: new URL(".", import.meta.url), encoding: "utf8", timeout: 8000 });
      expect(result.error, result.stderr).toBeUndefined();
      expect(result.status, result.stderr).toBe(0);
    },
    10000
  );
});
