import { expect, test } from "@playwright/test";

// Route baru Employee Database, Profil Saya, Metrik Personal, dan Dashboard tidak boleh
// dapat dibuka langsung tanpa sesi. Backend tetap menjadi otoritas; test ini memastikan
// guard frontend tidak pernah merender konten terlindungi sebelum identitas terselesaikan.
const protectedRoutes = [
  "/app/karyawan",
  "/app/karyawan/baru",
  "/app/dashboard",
  "/app/profil",
  "/app/metrik-personal",
];

for (const route of protectedRoutes) {
  test(`direct navigation to ${route} without a session returns to login`, async ({ page }) => {
    const apiRequests = [];
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/")) apiRequests.push(request.url());
    });

    await page.goto(route);

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("heading", { name: "Masuk ke akun" })).toBeVisible();

    // Tidak ada permintaan data sensitif yang dikirim untuk route terlarang.
    expect(
      apiRequests.filter((url) => /\/(karyawan|dashboard|profil)/.test(url)),
    ).toHaveLength(0);
  });
}
