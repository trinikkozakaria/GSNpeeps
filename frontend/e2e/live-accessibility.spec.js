import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;
const accounts = [
  ["karyawan@example.test", "Karyawan"],
  ["atasan@example.test", "Atasan"],
  ["hr@example.test", "HR"],
  ["top.management@example.test", "Top Management"],
];

const assertNoBlockingViolations = async (page, testInfo, label) => {
  const result = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "best-practice"])
    .analyze();
  await testInfo.attach(`${label}-axe.json`, {
    body: JSON.stringify(result.violations, null, 2),
    contentType: "application/json",
  });
  const blockers = result.violations.filter(({ impact }) =>
    ["critical", "serious"].includes(impact),
  );
  expect(blockers, `${label} has critical/serious accessibility violations`).toEqual([]);
};

test.describe("live accessibility smoke", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise the real Docker stack.");
    test.skip(testInfo.project.name !== "chromium", "Run one browser against shared seed accounts.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided in live mode").toBeTruthy();
  });

  test("login supports keyboard focus, reduced motion, narrow viewport, and axe", async ({ page }, testInfo) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.setViewportSize({ width: 640, height: 720 });
    await page.goto("/login");
    const email = page.getByLabel("Email kerja");
    const password = page.getByLabel(/^Password$/);
    await email.focus();
    await expect(email).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(password).toBeFocused();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await assertNoBlockingViolations(page, testInfo, "login-narrow");
  });

  for (const [email, role] of accounts) {
    test(`${role} landing page has no blocking axe findings`, async ({ page }, testInfo) => {
      await page.goto("/login");
      await page.getByLabel("Email kerja").fill(email);
      await page.getByLabel(/^Password$/).fill(seedPassword);
      await page.getByRole("button", { name: "Masuk" }).click();
      await expect(page).toHaveURL(/\/app$/);
      await expect(page.getByRole("heading", { name: /Selamat datang/ })).toBeVisible();
      await assertNoBlockingViolations(page, testInfo, `${role}-landing`);
      await page.getByRole("button", { name: "Keluar" }).click();
      await expect(page).toHaveURL(/\/login$/);
    });
  }

  test("HR access workflow reflows and keeps dialog keyboard focus safe", async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 640, height: 720 });
    await page.goto("/login");
    await page.getByLabel("Email kerja").fill("hr@example.test");
    await page.getByLabel(/^Password$/).fill(seedPassword);
    await page.getByRole("button", { name: "Masuk" }).click();
    await expect(page).toHaveURL(/\/app$/);

    await page.getByRole("link", { name: "AKSES" }).click();
    await expect(page.getByRole("heading", { name: "AKSES", level: 1 })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await assertNoBlockingViolations(page, testInfo, "hr-access-narrow");

    const trigger = page.getByRole("button", { name: /^(Cabut|Izinkan)$/ }).first();
    await trigger.click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Batal" })).toBeFocused();
    await assertNoBlockingViolations(page, testInfo, "hr-access-dialog");

    await page.keyboard.press("Shift+Tab");
    await expect(dialog.getByRole("button", { name: /^(Cabut|Izinkan)$/ })).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(dialog.getByRole("button", { name: "Batal" })).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(trigger).toBeFocused();

    await page.getByRole("button", { name: "Keluar" }).click();
    await expect(page).toHaveURL(/\/login$/);
  });
});
