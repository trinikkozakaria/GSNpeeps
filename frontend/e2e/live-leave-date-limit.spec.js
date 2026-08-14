import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;

test.describe("live leave date limit", () => {
  test.beforeEach(({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise the real Docker stack.");
    test.skip(testInfo.project.name !== "chromium", "One browser is enough for the live account.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided in live mode").toBeTruthy();
  });

  test("shows the SOP maximum and locks the inclusive end date", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email kerja").fill("karyawan.tanpa.atasan@example.test");
    await page.getByLabel(/^Password$/).fill(seedPassword);
    await page.getByRole("button", { name: "Masuk" }).click();
    await expect(page).toHaveURL(/\/app$/);

    await page.goto("/app/absensi/ketidakhadiran");
    await page.getByLabel("Jenis izin").selectOption({ label: "Ibadah Haji" });
    await expect(page.getByText("Maksimal 30 hari per pengajuan", { exact: true })).toBeVisible();

    await page.getByLabel("Tanggal mulai").fill("2026-08-10");
    const endDate = page.getByLabel("Tanggal selesai");
    await expect(endDate).toHaveValue("2026-09-08");
    await expect(endDate).toHaveAttribute("min", "2026-08-10");
    await expect(endDate).toHaveAttribute("max", "2026-09-08");
    await expect(page.getByText(/30 hari kalender.*tanggal lebih awal/i)).toBeVisible();

    await page.getByRole("button", { name: "Keluar" }).click();
    await expect(page).toHaveURL(/\/login$/);
  });
});
