import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noEmptyCatchBlockRule } from "./no-empty-catch-block";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-empty-catch-block", () => {
  it("uses the correct docs URL", () => {
    expect(noEmptyCatchBlockRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-empty-catch-block");
  });

  it("accepts catch blocks that log, assign a fallback, or document intent", () => {
    ruleTester.run("no-empty-catch-block", noEmptyCatchBlockRule, {
      valid: [
        `try { risky(); } catch (err) { core.debug(getErrorMessage(err)); }`,
        `try { value = JSON.parse(raw); } catch { value = {}; }`,
        `try { risky(); } catch { /* best-effort cleanup */ }`,
        `try { risky(); } catch { /* best effort cleanup */ }`,
        `try { risky(); } catch (err) { throw err; }`,
        `try { risky(); } catch (err) {\n  // intentional no-op: file may not exist on first run\n}`,
        `try { risky(); } catch { /* intentional ignore: optional file is absent */ }`,
      ],
      invalid: [],
    });
  });

  it("reports catch blocks with no statements and no explanatory comment", () => {
    ruleTester.run("no-empty-catch-block", noEmptyCatchBlockRule, {
      valid: [],
      invalid: [
        {
          code: `try { risky(); } catch {}`,
          errors: [{ messageId: "noEmptyCatch" }],
        },
        {
          code: `try { risky(); } catch (err) {}`,
          errors: [{ messageId: "noEmptyCatch" }],
        },
        {
          code: `try { risky(); } catch (err) {\n\n}`,
          errors: [{ messageId: "noEmptyCatch" }],
        },
        {
          code: `try { risky(); } catch { /* TODO */ }`,
          errors: [{ messageId: "noEmptyCatch" }],
        },
        {
          code: `try { risky(); } catch { /* file processing failed */ }`,
          errors: [{ messageId: "noEmptyCatch" }],
        },
        {
          code: `try { risky(); } catch { /* eslint-ignore */ }`,
          errors: [{ messageId: "noEmptyCatch" }],
        },
      ],
    });
  });
});
