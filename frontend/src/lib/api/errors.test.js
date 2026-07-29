import { describe, expect, it } from "vitest";

import { normalizeApiError } from "./errors";

describe("normalizeApiError", () => {
  it("preserves contract field errors", () => {
    const result = normalizeApiError({
      response: {
        status: 422,
        data: {
          error: {
            code: "VALIDATION_ERROR",
            message: "Data belum valid",
            fields: { email: "Email tidak valid" },
          },
        },
      },
    });

    expect(result.status).toBe(422);
    expect(result.fields.email).toBe("Email tidak valid");
  });
});

