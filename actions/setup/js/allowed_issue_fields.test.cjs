// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { parseAllowedIssueFields, validateAllowedIssueFieldName, validateAllowedIssueFields } = req("./allowed_issue_fields.cjs");

describe("parseAllowedIssueFields", () => {
  it("returns empty array for undefined", () => {
    expect(parseAllowedIssueFields(undefined)).toEqual([]);
  });

  it("returns empty array for empty string", () => {
    expect(parseAllowedIssueFields("")).toEqual([]);
  });

  it("returns empty array for null", () => {
    expect(parseAllowedIssueFields(null)).toEqual([]);
  });

  it("parses a comma-separated string", () => {
    expect(parseAllowedIssueFields("title,body,labels")).toEqual(["title", "body", "labels"]);
  });

  it("trims whitespace from each field", () => {
    expect(parseAllowedIssueFields(" title , body , labels ")).toEqual(["title", "body", "labels"]);
  });

  it("deduplicates repeated fields", () => {
    expect(parseAllowedIssueFields("title,title,body")).toEqual(["title", "body"]);
  });

  it("accepts an array input directly", () => {
    expect(parseAllowedIssueFields(["title", "body"])).toEqual(["title", "body"]);
  });

  it("deduplicates array input", () => {
    expect(parseAllowedIssueFields(["title", "title", "body"])).toEqual(["title", "body"]);
  });

  it("filters out empty items from comma-separated string", () => {
    expect(parseAllowedIssueFields("title,,body")).toEqual(["title", "body"]);
  });

  it("returns a single field as an array", () => {
    expect(parseAllowedIssueFields("title")).toEqual(["title"]);
  });
});

describe("validateAllowedIssueFieldName", () => {
  it("does not throw when fieldName is empty string", () => {
    expect(() => validateAllowedIssueFieldName("", ["title"])).not.toThrow();
  });

  it("does not throw when allowedFields is empty array (no restriction)", () => {
    expect(() => validateAllowedIssueFieldName("title", [])).not.toThrow();
  });

  it("does not throw when allowedFields contains wildcard '*'", () => {
    expect(() => validateAllowedIssueFieldName("anything", ["*"])).not.toThrow();
  });

  it("does not throw when field is in the allowed list", () => {
    expect(() => validateAllowedIssueFieldName("title", ["title", "body"])).not.toThrow();
  });

  it("throws ERR_VALIDATION when field is not in allowed list", () => {
    expect(() => validateAllowedIssueFieldName("labels", ["title", "body"])).toThrow("ERR_VALIDATION");
  });

  it("is case-insensitive when checking allowed fields", () => {
    expect(() => validateAllowedIssueFieldName("Title", ["title"])).not.toThrow();
  });

  it("does not throw when allowedFields is not an array (no restriction)", () => {
    // non-array allowedFields is treated as no restriction
    expect(() => validateAllowedIssueFieldName("title", null)).not.toThrow();
  });

  it("error message includes the disallowed field name", () => {
    expect(() => validateAllowedIssueFieldName("milestone", ["title", "body"])).toThrow('"milestone"');
  });
});

describe("validateAllowedIssueFields", () => {
  it("does not throw when issueFields is empty array", () => {
    expect(() => validateAllowedIssueFields([], ["title"])).not.toThrow();
  });

  it("does not throw when issueFields is null", () => {
    expect(() => validateAllowedIssueFields(null, ["title"])).not.toThrow();
  });

  it("does not throw when all fields are allowed", () => {
    const fields = [
      { name: "title", value: "New Title" },
      { name: "body", value: "New Body" },
    ];
    expect(() => validateAllowedIssueFields(fields, ["title", "body"])).not.toThrow();
  });

  it("throws when any field is not allowed", () => {
    const fields = [
      { name: "title", value: "New Title" },
      { name: "milestone", value: "v1.0" },
    ];
    expect(() => validateAllowedIssueFields(fields, ["title", "body"])).toThrow("ERR_VALIDATION");
  });

  it("does not throw when allowedFields is empty (no restriction)", () => {
    const fields = [{ name: "milestone", value: "v1.0" }];
    expect(() => validateAllowedIssueFields(fields, [])).not.toThrow();
  });

  it("does not throw when allowedFields has wildcard '*'", () => {
    const fields = [
      { name: "title", value: "New" },
      { name: "milestone", value: "v2.0" },
    ];
    expect(() => validateAllowedIssueFields(fields, ["*"])).not.toThrow();
  });
});
