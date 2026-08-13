import { expect, test } from "@playwright/test";

// Route notifikasi, AKSES, dan Audit Log ditambahkan pada epic ini. Seluruhnya terlindungi:
// membukanya langsung tanpa sesi harus kembali ke login tanpa mengirim permintaan data.
const protectedRoutes = ["/app/notifikasi", "/app/akses", "/app/audit"];

for (const route of protectedRoutes) {
  test(`direct navigation to ${route} without a session returns to login`, async ({ page }) => {
    const apiRequests = [];
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/")) apiRequests.push(request.url());
    });

    await page.goto(route);

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("heading", { name: "Masuk ke akun" })).toBeVisible();

    expect(
      apiRequests.filter((url) => /\/(notifikasi|akses)/.test(url)),
    ).toHaveLength(0);
  });
}

// Query string tidak boleh membuat route terlindungi merender konten lebih dahulu.
test("access route with a role filter still requires a session", async ({ page }) => {
  await page.goto("/app/akses?role=11111111-1111-4111-8111-111111111111");

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "AKSES" })).toHaveCount(0);
});

test("audit route with filters still requires a session", async ({ page }) => {
  await page.goto("/app/audit?tanggal_mulai=2026-08-01&modul=akses&page=2");

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Audit Log" })).toHaveCount(0);
});

// Lonceng notifikasi berada di dalam shell terautentikasi; halaman publik tidak memilikinya
// dan karena itu tidak pernah meminta jumlah belum dibaca.
test("login page does not render the notification bell or request the unread count", async ({
  page,
}) => {
  const apiRequests = [];
  page.on("request", (request) => {
    if (request.url().includes("/api/v1/")) apiRequests.push(request.url());
  });

  await page.goto("/login");

  await expect(page.getByRole("heading", { name: "Masuk ke akun" })).toBeVisible();
  await expect(page.getByRole("link", { name: /Notifikasi/ })).toHaveCount(0);
  expect(apiRequests.filter((url) => url.includes("unread-count"))).toHaveLength(0);
});
