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

  it("throws when body is present but no operation is resolvable", () => {
    expect(() => buildCommonEntityUpdateData({ body: "Body text" }, {})).toThrow("buildCommonEntityUpdateData: defaultOperation is required when body may be present");
  });

  it("returns hasCommonUpdates false when neither title nor body is present", () => {
    const result = buildCommonEntityUpdateData({}, {});

    expect(result.hasCommonUpdates).toBe(false);
    expect(result.updateData.title).toBeUndefined();
    expect(result.updateData._rawBody).toBeUndefined();
    expect(result.updateData._includeFooter).toBe(true);
  });

  it("populates _includeFooter false when config.footer is false", () => {
    const result = buildCommonEntityUpdateData({}, { footer: false });

    expect(result.updateData._includeFooter).toBe(false);
  });

  it('populates _includeFooter false when config.footer is the string "false"', () => {
    const result = buildCommonEntityUpdateData({}, { footer: "false" });

    expect(result.updateData._includeFooter).toBe(false);
  });

  it("does not include body in updateData.body by default when includeBodyInApiData is omitted", () => {
    const result = buildCommonEntityUpdateData({ body: "Body text" }, {}, { defaultOperation: "append" });

    expect(result.updateData._rawBody).toBe("Body text");
    expect(result.updateData.body).toBeUndefined();
  });

  it("handles title-only update without body operation", () => {
    const result = buildCommonEntityUpdateData({ title: "Only title" }, {});

    expect(result.updateData.title).toBe("Only title");
    expect(result.updateData._operation).toBeUndefined();
    expect(result.updateData._rawBody).toBeUndefined();
    expect(result.hasCommonUpdates).toBe(true);
  });

  it("does not call onBodyDisallowed when body is absent even if allow_body is false", () => {
    const onBodyDisallowed = vi.fn();

    buildCommonEntityUpdateData({}, { allow_body: false }, { onBodyDisallowed });

    expect(onBodyDisallowed).not.toHaveBeenCalled();
  });
});
