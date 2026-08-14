import { expect, test } from "@playwright/test";
import { openNavigationGroups } from "./helpers/navigation";

const liveMode = process.env.E2E_LIVE === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;

const login = async (page, email) => {
  await page.goto("/login");
  await page.getByLabel("Email kerja").fill(email);
  await page.getByLabel(/^Password$/).fill(seedPassword);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/app$/);
};

test("protected employee cache does not cross an HR-to-Karyawan user switch", async ({ page }, testInfo) => {
  test.skip(!liveMode, "Set E2E_LIVE=1 to exercise a real stack.");
  test.skip(testInfo.project.name !== "chromium", "Seed accounts allow one active session each.");
  expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();

  await login(page, "hr@example.test");
  await (await openNavigationGroups(page))
    .getByRole("link", { name: "Employee Database", exact: true })
    .click();
  await expect(page.getByRole("heading", { level: 1, name: "Employee Database" })).toBeVisible();
  await expect(page.getByText("HR Sintetis", { exact: true }).first()).toBeVisible();
  await page.getByRole("button", { name: "Keluar" }).click();
  await expect(page).toHaveURL(/\/login$/);

  await login(page, "karyawan@example.test");
  await expect(page.getByText("HR Sintetis", { exact: true })).toHaveCount(0);
  const employeeRequests = [];
  page.on("request", (request) => {
    if (/\/api\/v1\/karyawan(?:[/?]|$)/.test(request.url())) employeeRequests.push(request.url());
  });
  await page.evaluate(() => {
    window.history.pushState({}, "", "/app/karyawan");
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
  await expect(page).toHaveURL(/\/forbidden$/);
  await expect(page.getByRole("heading", { name: "Anda tidak memiliki akses" })).toBeVisible();
  expect(employeeRequests).toEqual([]);
  await expect(page.getByText("HR Sintetis", { exact: true })).toHaveCount(0);
  await page.getByRole("link", { name: "Kembali ke beranda" }).click();
  await expect(page).toHaveURL(/\/app$/);
  await page.getByRole("button", { name: "Keluar" }).click();
});
