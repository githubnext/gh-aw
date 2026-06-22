// @ts-check

import { describe, expect, it } from "vitest";

const { parseRuntimeFeatures, hasRuntimeFeature, getRuntimeFeatureValue } = require("./runtime_features.cjs");

describe("runtime_features", () => {
  it("parses newline-delimited flags and key value pairs", () => {
    const features = parseRuntimeFeatures("key\nkey2=value\nkey3 = spaced value");

    expect(features).toEqual({
      key: true,
      key2: "value",
      key3: "spaced value",
    });
  });

  it("ignores blank lines and malformed empty keys", () => {
    const features = parseRuntimeFeatures("\n  \n=value\nvalid=\n");

    expect(features).toEqual({
      valid: "",
    });
  });

  it("supports feature lookup helpers", () => {
    const features = parseRuntimeFeatures("flag\nmode=fast");

    expect(hasRuntimeFeature(features, "flag")).toBe(true);
    expect(hasRuntimeFeature(features, "missing")).toBe(false);
    expect(getRuntimeFeatureValue(features, "flag")).toBe(true);
    expect(getRuntimeFeatureValue(features, "mode")).toBe("fast");
    expect(getRuntimeFeatureValue(features, "missing")).toBeUndefined();
  });
});
