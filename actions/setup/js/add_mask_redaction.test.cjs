// @ts-check
import { describe, it, expect, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { applyAddMaskRedaction, collectAddMaskedValues, isAddMaskCommandLine, redactArtifactMaskedValues, redactMaskedValues, unescapeWorkflowCommandValue } = require("./add_mask_redaction.cjs");

describe("add_mask_redaction", () => {
  describe("isAddMaskCommandLine", () => {
    it("detects add-mask command lines", () => {
      expect(isAddMaskCommandLine("::add-mask::secret")).toBe(true);
      expect(isAddMaskCommandLine("2026-01-01 ::add-mask::secret")).toBe(true);
      expect(isAddMaskCommandLine("no command here")).toBe(false);
    });
  });

  describe("unescapeWorkflowCommandValue", () => {
    it("decodes workflow command escapes", () => {
      expect(unescapeWorkflowCommandValue("a%0Ab%0Dc%25d")).toBe("a\nb\rc%d");
    });
  });

  describe("collectAddMaskedValues", () => {
    it("collects masked values from log content", () => {
      const log = ["starting", "::add-mask::abcd1234", "::add-mask::zz", "done"].join("\n");
      const result = collectAddMaskedValues(log);
      expect(result).toContain("abcd1234");
      expect(result).toContain("zz");
      expect(result).toHaveLength(2);
    });

    it("expands multi-line masked values into individual lines", () => {
      const log = "::add-mask::line-one%0Aline-two\n";
      expect(collectAddMaskedValues(log).sort()).toEqual(["line-one", "line-two"]);
    });

    it("splits carriage-return separated values", () => {
      const log = "::add-mask::part-one%0Dpart-two\n";
      expect(collectAddMaskedValues(log).sort()).toEqual(["part-one", "part-two"]);
    });

    it("keeps both the verbatim and trimmed forms of a padded value", () => {
      const values = collectAddMaskedValues("::add-mask:: padded-secret \n");
      expect(values).toContain(" padded-secret ");
      expect(values).toContain("padded-secret");
    });

    it("ignores empty values and de-duplicates", () => {
      const log = ["::add-mask::", "::add-mask::   ", "::add-mask::dup", "::add-mask::dup"].join("\n");
      expect(collectAddMaskedValues(log)).toEqual(["dup"]);
    });

    it("returns an empty array for empty content", () => {
      expect(collectAddMaskedValues("")).toEqual([]);
    });
  });

  describe("redactMaskedValues", () => {
    it("replaces every occurrence with ***", () => {
      expect(redactMaskedValues("token=abc and abc again", ["abc"])).toBe("token=*** and *** again");
    });

    it("handles regex metacharacters in masked values", () => {
      expect(redactMaskedValues("value a.c here", ["a.c"])).toBe("value *** here");
      expect(redactMaskedValues("value abc here", ["a.c"])).toBe("value abc here");
    });

    it("returns the text unchanged when there are no masked values", () => {
      expect(redactMaskedValues("plain", [])).toBe("plain");
    });

    it("redacts masks exceeding the regular expression size limit", () => {
      const mask = "x".repeat(200000);
      expect(redactMaskedValues(`before ${mask} after ${mask}`, [mask])).toBe("before *** after ***");
      expect(redactMaskedValues("unrelated output", [mask])).toBe("unrelated output");
    });

    it("preserves leftmost matching and supplied mask priority for overlaps", () => {
      expect(redactMaskedValues("abc ab", ["abc", "ab"])).toBe("*** ***");
      expect(redactMaskedValues("abc", ["ab", "abc"])).toBe("***c");
      expect(redactMaskedValues("zabc", ["abc", "za"])).toBe("***bc");
    });

    it("does not rescan replacements or create matches across removed values", () => {
      expect(redactMaskedValues("secret *", ["secret", "*"])).toBe("*** ***");
      expect(redactMaskedValues("axb", ["x", "ab"])).toBe("a***b");
    });

    it("matches ordered regex semantics while refreshing overlapping cached candidates", () => {
      const values = ["ab", "aba", "ba", "b", "a", "", "ab"];
      for (let offset = 0; offset < values.length; offset++) {
        const masks = [...values.slice(offset), ...values.slice(0, offset)];
        const pattern = new RegExp(masks.filter(Boolean).join("|"), "g");
        for (let length = 0; length <= 6; length++) {
          for (let bits = 0; bits < 2 ** length; bits++) {
            const text = Array.from({ length }, (_, index) => (bits & (1 << index) ? "a" : "b")).join("");
            expect(redactMaskedValues(text, masks)).toBe(text.replace(pattern, "***"));
          }
        }
      }
    });

    it("does not re-search later masks for every frequent match", () => {
      const masks = Array.from({ length: 500 }, (_, index) => `mask-${index.toString().padStart(4, "0")}`);
      const repeats = 2000;
      const suffix = masks.join(" ");
      const text = "hot".repeat(repeats) + " ".repeat(2 * 1024 * 1024) + suffix;
      const indexOf = String.prototype.indexOf;
      let searches = 0;
      const spy = vi.spyOn(String.prototype, "indexOf").mockImplementation(function (value, from) {
        if (this === text) searches++;
        return indexOf.call(this, value, from);
      });
      let result;
      try {
        result = redactMaskedValues(text, [...masks, "hot"]);
      } finally {
        spy.mockRestore();
      }
      expect(result).toBe("***".repeat(repeats) + " ".repeat(2 * 1024 * 1024) + masks.map(() => "***").join(" "));
      expect(searches).toBeLessThanOrEqual(masks.length * 2 + repeats + 1);
    });
  });

  describe("applyAddMaskRedaction", () => {
    it("strips add-mask command lines and redacts masked values", () => {
      const log = ["output using s3cr3t", "::add-mask::s3cr3t", "more output"].join("\n");
      const maskedValues = collectAddMaskedValues(log);
      expect(applyAddMaskRedaction(log, maskedValues)).toBe(["output using ***", "more output"].join("\n"));
    });

    it("returns falsy input unchanged", () => {
      expect(applyAddMaskRedaction("", ["x"])).toBe("");
    });

    describe("redactArtifactMaskedValues", () => {
      it.each(["json", "mixed"])("redacts multi-megabyte %s records without exhausting the regex stack", shape => {
        const mask = 'synthetic-"quoted"\\mask';
        const value = "x".repeat(10 * 1024 * 1024) + mask;
        const record = JSON.stringify({ text: value, number: shape === "mixed" ? 42 : 12345678 });
        const prefix = shape === "mixed" ? "tool output: " : "";
        const redacted = redactArtifactMaskedValues(prefix + record, [mask, "12345678"]);
        expect(redacted.includes(mask)).toBe(false);
        expect(redacted.includes(JSON.stringify(mask).slice(1, -1))).toBe(false);
        const parsed = JSON.parse(redacted.slice(prefix.length));
        expect(parsed.text.length).toBe(10 * 1024 * 1024 + 3);
        expect(parsed.text.endsWith("***")).toBe(true);
        expect(parsed.number).toBe(shape === "mixed" ? 42 : "***");
      });

      it("handles many escaped quotes and both odd and even backslash runs", () => {
        const mask = 'synthetic-"quoted"\\mask';
        const value = '\\"'.repeat(1024 * 1024) + mask + "\\\\";
        const record = JSON.stringify({ text: value, safe: "12345678" });
        const redacted = redactArtifactMaskedValues(record, [mask]);
        const parsed = JSON.parse(redacted);
        expect(parsed.text.length).toBe(2 * 1024 * 1024 + 5);
        expect(parsed.text.endsWith("***\\\\")).toBe(true);
        expect(parsed.safe).toBe("12345678");
      });

      it.each(["\n", "\r", "\u2028", "\u2029"])("recovers encoded masks after an invalid escaped line terminator", terminator => {
        const mask = 'synthetic-"quoted"\\mask';
        const prefix = '"invalid\\' + terminator;
        expect(redactArtifactMaskedValues(prefix + JSON.stringify(mask), [mask])).toBe(prefix + '"***"');
      });

      it.each(['"bad\\q ', '"bad\\u12gg ', '"bad\\u12 ', '"bad\u0001 ', '"bad\t ', '"bad\\v '])("recovers Unicode-encoded masks after invalid JSON string prefixes (%j)", prefix => {
        expect(redactArtifactMaskedValues(prefix + '"\\u006fpaque"', ["opaque"])).toBe(prefix + '"***"');
      });

      it("redacts decoded JSON strings and keys without mutating syntax or primitives", () => {
        const secret = 'quote"\\percent%0A';
        const value = { [secret]: [secret, 0, false, null], nested: { output: "first\nsecond" } };
        const masks = collectAddMaskedValues(`::add-mask::${secret.replace(/%/g, "%25")}\n::add-mask::first%0Asecond\n`);
        const redacted = redactArtifactMaskedValues(JSON.stringify(value, null, 2) + "\n", masks);
        expect(JSON.parse(redacted)).toEqual({ "***": ["***", 0, false, null], nested: { output: "***\n***" } });
        expect(redacted.endsWith("\n")).toBe(true);
        expect(value[secret][0]).toBe(secret);
      });

      it("keeps JSON valid when a short mask matches JSON punctuation", () => {
        expect(JSON.parse(redactArtifactMaskedValues('{"text":"\\""}\n', ['"']))).toEqual({ text: "***" });
      });

      it.each(["12345678", "-12345678", "12345678.5", "1.12345678e+9", "12345678901234567890"])("redacts numeric scalar %s using its original token and keeps JSON valid", token => {
        const redacted = redactArtifactMaskedValues(` \t${token}\r\n`, ["12345678"]);
        expect(redacted).toBe(` \t${JSON.stringify(token.replaceAll("12345678", "***"))}\r\n`);
        expect(JSON.parse(redacted)).toBe(token.replaceAll("12345678", "***"));
      });

      it("redacts numeric scalar records within mixed logs and JSONL", () => {
        const content = 'starting\n12345678\n{"data":"12345678"}\n-12345678\n123456\nfalse\nnull\n';
        const redacted = redactArtifactMaskedValues(content, ["12345678"]);
        expect(redacted).toBe('starting\n"***"\n{"data":"***"}\n"-***"\n123456\nfalse\nnull\n');
        for (const line of redacted.trim().split("\n").slice(1)) expect(() => JSON.parse(line)).not.toThrow();
      });

      it.each(["12345678", "-12345678", "12345678.5", "1.12345678e+9", "12345678901234567890"])("redacts nested numeric token %s without numeric normalization", token => {
        const content = ` \t{"token":${token},"ids":[${token},42],"nested":{"items":[{"token":${token}}]}}\r\n`;
        const redacted = redactArtifactMaskedValues(content, ["12345678"]);
        const masked = token.replaceAll("12345678", "***");
        expect(JSON.parse(redacted)).toEqual({ token: masked, ids: [masked, 42], nested: { items: [{ token: masked }] } });
        expect(redacted).not.toContain("12345678");
        expect(redacted.startsWith(" \t")).toBe(true);
        expect(redacted.endsWith("\r\n")).toBe(true);
      });

      it.each(["9007199254740993", "123456789012345678901234567890", "1e400"])("redacts an exact nested numeric credential %s before precision loss", token => {
        const content = `{"ids":[${token}],"text":"safe"}`;
        expect(JSON.parse(redactArtifactMaskedValues(content, [token]))).toEqual({ ids: ["***"], text: "safe" });
      });

      it("redacts nested numeric records in JSONL and mixed logs", () => {
        const records = '{"token":12345678,"safe":42}\n[42,{"ids":[12345678901234567890]}]\n{"nested":{"token":1.12345678e+9}}\n';
        const expected = [{ token: "***", safe: 42 }, [42, { ids: ["***90***90"] }], { nested: { token: "1.***e+9" } }];
        for (const prefix of ["", "::add-mask::12345678\nstarting\n"]) {
          const redacted = redactArtifactMaskedValues(prefix + records, ["12345678"]);
          expect(redacted).not.toContain("12345678");
          const lines = redacted.trim().split("\n");
          if (prefix) expect(lines.shift()).toBe("starting");
          expect(lines.map(line => JSON.parse(line))).toEqual(expected);
        }
      });

      it("does not match normalized numeric representations or alter unmatched nested records", () => {
        const content = '  {"values":[-0,1e+09,9007199254740993,1e400],"escaped":"quote\\"12345","safe":false}\r\n';
        expect(redactArtifactMaskedValues(content, ["12345678", "1000000000", "9007199254740992", "Infinity"])).toBe(content);
      });

      it("preserves nonmatching numeric tokens alongside matching nested numbers", () => {
        const content = '  {"values":[12345678,-0,1e+09,9007199254740993,1e400]}\r\n';
        expect(redactArtifactMaskedValues(content, ["12345678"])).toBe('  {"values":["***",-0,1e+09,9007199254740993,1e400]}\r\n');
      });

      it("keeps matching quoted numeric strings and numeric keys valid", () => {
        const content = '{"12345678":"quote\\"12345678","values":[12345678,123456]}';
        expect(JSON.parse(redactArtifactMaskedValues(content, ["12345678"]))).toEqual({ "***": 'quote"***', values: ["***", 123456] });
      });

      it.each(["123456", "-0", "1e+09", "9007199254740993", "1e400", "false", "null"])("preserves unmatched scalar %s byte-for-byte", token => {
        const content = ` \t${token}\r\n`;
        expect(redactArtifactMaskedValues(content, ["12345678"])).toBe(content);
      });

      it("redacts JSONL, Unicode escapes, and malformed adjacent records", () => {
        const content = '::add-mask::opaque\n{"data":"\\u006fpaque"}\n{"broken":"\\u006fpaque",\nplain opaque\n{"data":"safe"}\n';
        const redacted = redactArtifactMaskedValues(content, ["opaque"]);
        expect(redacted).toBe('{"data":"***"}\n{"broken":"***",\nplain ***\n{"data":"safe"}\n');
      });

      it("preserves untouched formatting and empty or unmasked inputs", () => {
        const content = '  {\n    "empty": "", "zero": 0\n  }\n';
        expect(redactArtifactMaskedValues(content, ["absent"])).toBe(content);
        expect(redactArtifactMaskedValues(content, [])).toBe(content);
        expect(redactArtifactMaskedValues("", ["secret"])).toBe("");
      });
    });
  });
});
