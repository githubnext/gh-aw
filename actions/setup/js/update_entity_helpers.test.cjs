import { describe, it, expect, vi } from "vitest";
const { buildCommonEntityUpdateData } = require("./update_entity_helpers.cjs");

describe("update_entity_helpers.cjs - buildCommonEntityUpdateData", () => {
  it("builds shared title/body/footer fields with defaults", () => {
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

  it("uses configured operation fallback and can include body api field", () => {
    const result = buildCommonEntityUpdateData(
      { body: "Body text" },
      { default_operation: "replace" },
      {
        defaultOperation: "append",
        configDefaultOperation: "replace",
        includeBodyInApiData: true,
      }
    );

    expect(result.updateData._operation).toBe("replace");
    expect(result.updateData.body).toBe("Body text");
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
