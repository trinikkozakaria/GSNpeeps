import { describe, expect, it } from "vitest";

import { getMaximumEndDate } from "../data/sop-leave-policy";

describe("getMaximumEndDate", () => {
  it("counts only weekdays when the range does not touch a weekend", () => {
    // 2026-08-10 is a Monday; 3 business days lands on Wednesday.
    expect(getMaximumEndDate("2026-08-10", 3)).toBe("2026-08-12");
  });

  it("skips Saturday and Sunday when counting business days", () => {
    // 2026-08-14 is a Friday; 3 business days skips the weekend to Tuesday.
    expect(getMaximumEndDate("2026-08-14", 3)).toBe("2026-08-18");
  });

  it("does not count a weekend start date itself", () => {
    // 2026-08-15 is a Saturday; the first business day counted is Monday.
    expect(getMaximumEndDate("2026-08-15", 1)).toBe("2026-08-17");
  });

  it("returns an empty string for invalid input", () => {
    expect(getMaximumEndDate("", 3)).toBe("");
    expect(getMaximumEndDate("2026-08-10", null)).toBe("");
    expect(getMaximumEndDate("2026-08-10", 0)).toBe("");
  });
});
