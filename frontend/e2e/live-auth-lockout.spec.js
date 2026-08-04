import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const lockoutMode = process.env.E2E_LOCKOUT === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;
const email = "karyawan@example.test";

const postLogin = (request, password) =>
  request.post("/api/v1/auth/login", { data: { email, password } });

const postReset = (request, currentPassword, newPassword) =>
  request.post("/api/v1/auth/reset-password", {
    data: {
      email,
      current_password: currentPassword,
      new_password: newPassword,
      new_password_confirmation: newPassword,
    },
  });

test.describe("live account lockout", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise the real Docker stack.");
    test.skip(!lockoutMode, "Set E2E_LOCKOUT=1 because this scenario temporarily locks a seed account.");
    test.skip(testInfo.project.name !== "chromium", "One project owns the live seed account.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided in live mode").toBeTruthy();
  });

  test("five failures lock the account, revoke its session, and self-reset restores it", async ({ request }) => {
    const temporaryPassword = `${seedPassword}#lockout-e2e`;

    const activeLogin = await postLogin(request, seedPassword);
    expect(activeLogin.status()).toBe(200);
    const activeToken = (await activeLogin.json()).data.token;

    for (let attempt = 1; attempt <= 4; attempt += 1) {
      const failure = await postLogin(request, `${seedPassword}#wrong-${attempt}`);
      expect(failure.status()).toBe(401);
      expect((await failure.json()).error.code).toBe("INVALID_CREDENTIALS");
    }

    const lock = await postLogin(request, `${seedPassword}#wrong-5`);
    expect(lock.status()).toBe(429);
    expect((await lock.json()).error.code).toBe("ACCOUNT_LOCKED");

    const revokedSession = await request.get("/api/v1/auth/me", {
      headers: { Authorization: `Bearer ${activeToken}` },
    });
    expect(revokedSession.status()).toBe(401);

    const recovery = await postReset(request, seedPassword, temporaryPassword);
    expect(recovery.status()).toBe(200);
    expect(await recovery.json()).toMatchObject({
      success: true,
      data: { password_changed: true, account_locked: false, sessions_revoked: true },
    });

    const temporaryLogin = await postLogin(request, temporaryPassword);
    expect(temporaryLogin.status()).toBe(200);

    const restore = await postReset(request, temporaryPassword, seedPassword);
    expect(restore.status()).toBe(200);
    const restoredLogin = await postLogin(request, seedPassword);
    expect(restoredLogin.status()).toBe(200);
    const restoredToken = (await restoredLogin.json()).data.token;
    expect(
      (
        await request.post("/api/v1/auth/logout", {
          headers: { Authorization: `Bearer ${restoredToken}` },
        })
      ).status(),
    ).toBe(200);
  });
});
