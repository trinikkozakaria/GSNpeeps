import { describe, expect, it } from "vitest";

import { safeReturnPath } from "./safe-return-path";

describe("safeReturnPath", () => {
  it("accepts an internal path", () => {
    expect(safeReturnPath("/app/notifikasi?page=2")).toBe("/app/notifikasi?page=2");
  });

  it("rejects open redirects", () => {
    expect(safeReturnPath("//evil.example/path")).toBe("/app");
    expect(safeReturnPath("https://evil.example/path")).toBe("/app");
  });
});

