// @ts-check

import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
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

  it("treats only the first equals sign as the key/value separator", () => {
    expect(parseRuntimeFeatures("key=a=b=c")).toEqual({
      key: "a=b=c",
    });
  });

  it("ignores blank lines and malformed empty keys", () => {
    const features = parseRuntimeFeatures("\n  \n=value\nvalid=\n");

    expect(features).toEqual({
      valid: "",
    });
    expect(hasRuntimeFeature(features, "valid")).toBe(true);
    expect(getRuntimeFeatureValue(features, "valid")).toBe("");
  });

  it("supports feature lookup helpers", () => {
    const features = parseRuntimeFeatures("flag\nmode=fast");

    expect(hasRuntimeFeature(features, "flag")).toBe(true);
    expect(hasRuntimeFeature(features, "missing")).toBe(false);
    expect(getRuntimeFeatureValue(features, "flag")).toBe(true);
    expect(getRuntimeFeatureValue(features, "mode")).toBe("fast");
    expect(getRuntimeFeatureValue(features, "missing")).toBeUndefined();
  });

  it("returns an empty map for nullish input", () => {
    expect(parseRuntimeFeatures(null)).toEqual({});
    expect(parseRuntimeFeatures(undefined)).toEqual({});
  });
});
