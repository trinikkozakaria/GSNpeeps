import { describe, expect, it } from "vitest";

import {
  changePasswordSchema,
  loginSchema,
  selfResetPasswordSchema,
} from "./auth-schema";

describe("auth schemas", () => {
  it("validates login fields from OpenAPI", () => {
    expect(loginSchema.safeParse({ email: "invalid", password: "short" }).success).toBe(false);
    expect(
      loginSchema.safeParse({
        email: "user@example.test",
        password: "valid-password",
      }).success,
    ).toBe(true);
  });

  it("requires matching new password confirmation", () => {
    const result = changePasswordSchema.safeParse({
      current_password: "current-password",
      new_password: "new-password-value",
      new_password_confirmation: "different-value",
    });
    expect(result.success).toBe(false);
  });

  it("requires email for public self reset", () => {
    const result = selfResetPasswordSchema.safeParse({
      current_password: "current-password",
      new_password: "new-password-value",
      new_password_confirmation: "new-password-value",
    });
    expect(result.success).toBe(false);
  });
});

