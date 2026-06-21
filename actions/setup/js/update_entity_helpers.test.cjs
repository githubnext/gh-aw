import { describe, it, expect, vi } from "vitest";
const { buildCommonEntityUpdateData } = require("./update_entity_helpers.cjs");

describe("update_entity_helpers.cjs - buildCommonEntityUpdateData", () => {
  it("returns hasCommonUpdates true and populates title, body fields, and footer when title and body are provided", () => {
    const result = buildCommonEntityUpdateData(
      { title: "New title", body: "Body text" },
      {},
      {
        defaultOperation: "append",
      }
    );

    expect(result.updateData.title).toBe("New title");
    expect(result.updateData._operation).toBe("append");
    expect(result.updateData._rawBody).toBe("Body text");
    expect(result.updateData._includeFooter).toBe(true);
    expect(result.hasCommonUpdates).toBe(true);
  });

  it("prefers configDefaultOperation over defaultOperation for body operation", () => {
    const result = buildCommonEntityUpdateData(
      { body: "Body text" },
      {},
      {
        defaultOperation: "append",
        configDefaultOperation: "replace",
      }
    );

    expect(result.updateData._operation).toBe("replace");
  });

  it("includes body in api data when includeBodyInApiData is true", () => {
    const result = buildCommonEntityUpdateData(
      { body: "Body text" },
      {},
      {
        defaultOperation: "append",
        includeBodyInApiData: true,
      }
    );

    expect(result.updateData.body).toBe("Body text");
  });

  it("item.operation takes precedence over configDefaultOperation and defaultOperation", () => {
    const result = buildCommonEntityUpdateData(
      { body: "Body text", operation: "prepend" },
      {},
      {
        defaultOperation: "append",
        configDefaultOperation: "replace",
      }
    );

    expect(result.updateData._operation).toBe("prepend");
  });

  it("skips title when allowTitle is false and does not set hasCommonUpdates", () => {
    const result = buildCommonEntityUpdateData({ title: "Should be ignored" }, {}, { allowTitle: false, defaultOperation: "append" });

    expect(result.updateData.title).toBeUndefined();
    expect(result.hasCommonUpdates).toBe(false);
  });

  it("invokes onBodyDisallowed when body updates are blocked", () => {
    const onBodyDisallowed = vi.fn();

    const result = buildCommonEntityUpdateData(
      { body: "Body text" },
      { allow_body: false },
      {
        defaultOperation: "append",
        onBodyDisallowed,
      }
    );

    expect(onBodyDisallowed).toHaveBeenCalledTimes(1);
    expect(result.updateData._rawBody).toBeUndefined();
    expect(result.hasCommonUpdates).toBe(false);
  });
});
