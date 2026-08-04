import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;

const accounts = [
  {
    email: "karyawan@example.test",
    role: "Karyawan",
    allowed: ["Profil Saya", "Metrik Personal", "Kehadiran Saya", "Pengajuan Saya"],
    forbidden: ["Employee Database", "Dashboard HR", "Persetujuan", "AKSES", "Audit Log"],
    destinations: [
      ["Profil Saya", "Profil Saya"],
      ["Metrik Personal", "Metrik Personal"],
    ],
  },
  {
    email: "atasan@example.test",
    role: "Atasan",
    allowed: ["Profil Saya", "Metrik Personal", "Kehadiran Saya", "Persetujuan"],
    forbidden: ["Employee Database", "Dashboard HR", "AKSES", "Audit Log"],
    destinations: [["Persetujuan", "Persetujuan"]],
  },
  {
    email: "hr@example.test",
    role: "HR",
    allowed: ["Employee Database", "Dashboard HR", "Persetujuan", "AKSES", "Audit Log"],
    forbidden: [],
    destinations: [
      ["Employee Database", "Employee Database"],
      ["Dashboard HR", "Dashboard HR"],
      ["AKSES", "AKSES"],
      ["Audit Log", "Audit Log"],
    ],
  },
  {
    email: "top.management@example.test",
    role: "Top Management",
    allowed: ["Employee Database", "Dashboard HR", "Persetujuan", "AKSES", "Audit Log"],
    forbidden: ["Profil Saya", "Metrik Personal", "Kehadiran Saya", "Master Jenis Izin"],
    destinations: [
      ["Employee Database", "Employee Database"],
      ["Dashboard HR", "Dashboard HR"],
      ["AKSES", "AKSES"],
      ["Audit Log", "Audit Log"],
    ],
  },
];

test.describe("live Docker role authentication", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise the real Docker stack.");
    test.skip(testInfo.project.name !== "chromium", "The live seed accounts allow one active session each.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided in live mode").toBeTruthy();
  });

  for (const account of accounts) {
    test(`${account.role} logs in, receives role navigation, and logs out`, async ({ page }) => {
      const apiFailures = [];
      page.on("response", (response) => {
        if (response.url().includes("/api/v1/") && response.status() >= 400) {
          apiFailures.push(`${response.status()} ${response.url()}`);
        }
      });

      await page.goto("/login");
      await page.getByLabel("Email kerja").fill(account.email);
      await page.getByLabel(/^Password$/).fill(seedPassword);
      await page.getByRole("button", { name: "Masuk" }).click();

      await expect(page).toHaveURL(/\/app$/);
      await expect(page.getByRole("heading", { name: /Selamat datang/ })).toBeVisible();
      await expect(page.getByText(`Anda masuk sebagai ${account.role}.`)).toBeVisible();

      const navigation = page.getByRole("navigation", { name: "Navigasi utama" });
      for (const label of account.allowed) {
        await expect(navigation.getByRole("link", { name: label, exact: true })).toBeVisible();
      }
      for (const label of account.forbidden) {
        await expect(navigation.getByRole("link", { name: label, exact: true })).toHaveCount(0);
      }

      for (const [link, heading] of account.destinations) {
        await navigation.getByRole("link", { name: link, exact: true }).click();
        await expect(page.getByRole("heading", { level: 1, name: heading, exact: true })).toBeVisible();
      }

      if (account.role === "Karyawan") {
        const employeeRequests = [];
        page.on("request", (request) => {
          if (/\/api\/v1\/karyawan(?:[/?]|$)/.test(request.url())) {
            employeeRequests.push(request.url());
          }
        });
        await page.evaluate(() => {
          window.history.pushState({}, "", "/app/karyawan");
          window.dispatchEvent(new PopStateEvent("popstate"));
        });
        await expect(page).toHaveURL(/\/forbidden$/);
        await expect(page.getByRole("heading", { name: "Anda tidak memiliki akses" })).toBeVisible();
        expect(employeeRequests).toEqual([]);
        await page.getByRole("link", { name: "Kembali ke beranda" }).click();
        await expect(page).toHaveURL(/\/app$/);
      }
      expect(apiFailures).toEqual([]);

      await page.getByRole("button", { name: "Keluar" }).click();
      await expect(page).toHaveURL(/\/login$/);
      await expect(page.getByRole("heading", { name: "Masuk ke akun" })).toBeVisible();
    });
  }

});
