import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const eslintConfig = readFileSync(resolve(__dirname, "../eslint.config.cjs"), "utf8");

describe("eslint.config.cjs", () => {
  it("promotes require-http-response-error-listener to error severity", () => {
    expect(eslintConfig).toContain(`"gh-aw-custom/require-http-response-error-listener": "error"`);
  });

  it("keeps non-promoted custom rules at warning severity", () => {
    expect(eslintConfig).toContain(`"gh-aw-custom/no-throw-plain-object": "warn"`);
  });
});
