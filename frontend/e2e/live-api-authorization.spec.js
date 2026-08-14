import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;

const login = async (request, email) => {
  const response = await request.post("/api/v1/auth/login", {
    data: { email, password: seedPassword },
  });
  expect(response.status()).toBe(200);
  const body = await response.json();
  expect(body.success).toBe(true);
  expect(body.data.token).toBeTruthy();
  return body.data.token;
};

const bearer = (token) => ({ Authorization: `Bearer ${token}` });

test.describe("live API authorization", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise the real Docker stack.");
    test.skip(testInfo.project.name !== "chromium", "One project owns the live seed sessions.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided in live mode").toBeTruthy();
  });

  test("anonymous request cannot read employees", async ({ request }) => {
    const response = await request.get("/api/v1/karyawan");
    expect(response.status()).toBe(401);
  });

  for (const account of ["karyawan@example.test", "atasan@example.test"]) {
    test(`${account} cannot read the employee database`, async ({ request }) => {
      const token = await login(request, account);
      const response = await request.get("/api/v1/karyawan", { headers: bearer(token) });
      expect(response.status()).toBe(403);
    });
  }

  test("HR can read employees and the permission matrix", async ({ request }) => {
    const token = await login(request, "hr@example.test");
    const headers = bearer(token);

    await expect((await request.get("/api/v1/karyawan", { headers })).status()).toBe(200);
    await expect((await request.get("/api/v1/akses/permission", { headers })).status()).toBe(200);
  });

  test("Top Management cannot read employees or mutate permissions", async ({ request }) => {
    const token = await login(request, "top.management@example.test");
    const headers = bearer(token);

    await expect((await request.get("/api/v1/karyawan", { headers })).status()).toBe(403);
    const mutation = await request.put("/api/v1/akses/permission", {
      headers,
      data: {
        // Payload sengaja mustahil lolos validasi. Jika guard authorization mengalami
        // regresi, request tetap tidak dapat mengubah permission pada database live-test.
        role_id: "not-a-uuid",
        modul: "not-a-module",
        aksi: "not-an-action",
        diizinkan: true,
      },
    });
    expect(mutation.status()).toBe(403);
  });

  test("logout invalidates the previous token immediately", async ({ request }) => {
    const token = await login(request, "karyawan@example.test");
    const headers = bearer(token);

    expect((await request.post("/api/v1/auth/logout", { headers })).status()).toBe(200);
    expect((await request.get("/api/v1/auth/me", { headers })).status()).toBe(401);
  });
});
