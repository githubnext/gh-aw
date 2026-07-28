import { describe, expect, it } from "vitest";
import { resolveDataSchema } from "./data_schema_normalizer.cjs";

describe("data_schema_normalizer", () => {
  it("accepts shorthand object syntax", () => {
    const schema = resolveDataSchema(
      {
        verdict: "string",
        criteria_passed: "number",
      },
      "safe-outputs.data"
    );

    expect(schema).toEqual({
      type: "object",
      properties: {
        verdict: { type: "string" },
        criteria_passed: { type: "number" },
      },
      required: ["criteria_passed", "verdict"],
      additionalProperties: false,
    });
  });

  it("accepts JSON string schema syntax", () => {
    const schema = resolveDataSchema(
      JSON.stringify({
        type: "object",
        properties: {
          verdict: { type: "string", enum: ["APPROVE", "REJECT"] },
        },
        required: ["verdict"],
        additionalProperties: false,
      }),
      "safe-outputs.data"
    );

    expect(schema.type).toBe("object");
    expect(schema.properties.verdict.enum).toEqual(["APPROVE", "REJECT"]);
    expect(schema.additionalProperties).toBe(false);
  });

  it("rejects schemas with unsupported keywords", () => {
    expect(() =>
      resolveDataSchema(
        {
          type: "object",
          properties: {
            verdict: {
              type: "string",
              $ref: "#/$defs/verdict",
            },
          },
        },
        "safe-outputs.data"
      )
    ).toThrow("unsupported keyword");
  });

  it("rejects additionalProperties: true for codex compatibility", () => {
    expect(() =>
      resolveDataSchema(
        {
          type: "object",
          properties: {
            verdict: "string",
          },
          additionalProperties: true,
        },
        "safe-outputs.data"
      )
    ).toThrow("must be false for OpenAI Codex structured outputs compatibility");
  });
});
