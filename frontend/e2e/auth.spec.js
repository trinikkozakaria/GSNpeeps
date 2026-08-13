import { expect, test } from "@playwright/test";

test("login page is reachable on desktop and mobile", async ({ page }) => {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Masuk ke akun" })).toBeVisible();
  await expect(page.getByLabel("Email kerja")).toBeVisible();
  await expect(page.getByLabel(/^Password$/)).toBeVisible();
  await expect(page).toHaveTitle(/Login — GSNpeeps/);
});

test("anonymous protected navigation returns to login", async ({ page }) => {
  await page.goto("/app/karyawan");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Masuk ke akun" })).toBeVisible();
});
