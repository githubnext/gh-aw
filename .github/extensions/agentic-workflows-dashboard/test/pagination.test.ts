import { describe, expect, it } from "vitest";

import { paginate } from "../src/pagination.js";

describe("paginate", () => {
  it("returns expected page metadata and items", () => {
    const result = paginate([1, 2, 3, 4, 5], 2, 2);
    expect(result.items).toEqual([3, 4]);
    expect(result.page).toBe(2);
    expect(result.totalPages).toBe(3);
    expect(result.hasNextPage).toBe(true);
    expect(result.hasPreviousPage).toBe(true);
  });

  it("clamps page to valid bounds", () => {
    const result = paginate([1, 2, 3], 999, 2);
    expect(result.page).toBe(2);
    expect(result.items).toEqual([3]);
    expect(result.hasNextPage).toBe(false);
  });
});
