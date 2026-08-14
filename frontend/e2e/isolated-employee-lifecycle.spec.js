import { expect, test } from "@playwright/test";
import { openNavigationGroups } from "./helpers/navigation";
import { assertDisposableMutationTarget } from "./helpers/mutation-target";

const liveMode = process.env.E2E_LIVE === "1";
const mutationMode = process.env.E2E_MUTATION === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;

test.describe("isolated employee lifecycle", () => {
  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise a real stack.");
    test.skip(!mutationMode, "Set E2E_MUTATION=1 only for a disposable stack.");
    test.skip(testInfo.project.name !== "chromium", "The isolated workflow needs one browser project.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();

    assertDisposableMutationTarget(testInfo.project.use.baseURL);
  });

  test("HR creates, filters, views, updates, and soft-deactivates a synthetic employee", async ({ page }) => {
    const apiFailures = [];
    page.on("response", (response) => {
      if (response.url().includes("/api/v1/") && response.status() >= 400) {
        apiFailures.push(`${response.status()} ${response.url()}`);
      }
    });

    await page.goto("/login");
    await page.getByLabel("Email kerja").fill("hr@example.test");
    await page.getByLabel(/^Password$/).fill(seedPassword);
    await page.getByRole("button", { name: "Masuk" }).click();
    await expect(page).toHaveURL(/\/app$/);

    await (await openNavigationGroups(page))
      .getByRole("link", { name: "Employee Database", exact: true })
      .click();
    await page.getByRole("link", { name: "Tambah karyawan" }).click();
    await expect(page.getByRole("heading", { name: "Tambah karyawan" })).toBeVisible();

    await page.getByLabel("NIP").fill("E2E-EMP-001");
    await page.getByLabel("Nama lengkap").fill("Karyawan Lifecycle E2E");
    await page.getByLabel("Email login").fill("employee.lifecycle@example.test");
    await page.getByLabel("Jenis kelamin").selectOption("L");
    await page.getByLabel("Tanggal lahir").fill("1995-05-15");
    await page.getByLabel("Tanggal bergabung").fill("2026-08-03");
    await page.getByLabel("Departemen").selectOption({ index: 1 });
    const departmentID = await page.getByLabel("Departemen").inputValue();
    await expect(page.getByLabel("Jabatan").locator("option")).not.toHaveCount(1);
    await page.getByLabel("Jabatan").selectOption({ index: 1 });
    const positionID = await page.getByLabel("Jabatan").inputValue();
    await page.getByLabel("Status pernikahan").selectOption("lajang");
    await page.getByLabel("Jalan").fill("Jalan Integrasi 1");
    await page.getByLabel("Kelurahan").fill("Karet");
    await page.getByLabel("Kecamatan").fill("Setiabudi");
    await page.getByLabel("Kota/Kabupaten").fill("Jakarta Selatan");
    await page.getByLabel("Provinsi").fill("DKI Jakarta");
    await page.getByLabel("Nomor KTP").fill("3174000000000001");
    await page.getByLabel("Nomor kontrak").fill("E2E-CONTRACT-001");
    await page.getByLabel("Tanggal mulai kontrak").fill("2026-08-03");
    await page.getByLabel("Tanggal berakhir kontrak").fill("2027-08-02");
    await page.getByLabel("Periode (YYYY-MM)").fill("2026-08");
    await page.getByLabel("Gaji pokok").fill("7500000");
    await page.getByRole("button", { name: "Simpan karyawan" }).click();

    await expect(page.getByRole("heading", { name: "Karyawan Lifecycle E2E" })).toBeVisible();
    await expect(page.getByText("E2E-EMP-001", { exact: true }).first()).toBeVisible();

    await expect(page.getByText("Belum ada dokumen yang diunggah.")).toBeVisible();
    await page.getByLabel("Jenis dokumen").selectOption({ label: "Dokumen E2E" });
    await page.getByLabel("Berkas dokumen").setInputFiles({
      name: "dokumen-e2e.pdf",
      mimeType: "application/pdf",
      buffer: Buffer.from("%PDF-1.4\n%%EOF"),
    });
    await page.getByRole("button", { name: "Unggah dokumen" }).click();
    await expect(page.getByRole("status").filter({ hasText: "Dokumen berhasil diunggah." })).toBeVisible();
    await expect(page.getByRole("link", { name: /dokumen-e2e\.pdf/ })).toBeVisible();
    await expect(page.getByTitle("Pratinjau dokumen-e2e.pdf")).toBeVisible();

    const exportResponsePromise = page.waitForResponse(
      (response) => response.url().includes("/api/v1/karyawan/export") && response.status() === 200,
    );
    await page.getByRole("group", { name: "Export karyawan ini" })
      .getByRole("button", { name: "PDF" })
      .click();
    const exportResponse = await exportResponsePromise;
    expect(exportResponse.headers()["content-type"]).toContain("application/pdf");
    expect(exportResponse.headers()["content-disposition"]).toContain("attachment");
    await expect(page.getByText(/Berkas .*\.pdf berhasil diunduh\./)).toBeVisible();

    await page.getByRole("link", { name: "Edit", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Edit karyawan" })).toBeVisible();
    await expect(page.getByLabel("Nama lengkap")).toHaveValue("Karyawan Lifecycle E2E");
    await page.getByLabel("Nama lengkap").fill("Karyawan Lifecycle E2E Diperbarui");
    await page.getByRole("button", { name: "Simpan perubahan" }).click();
    await expect(page.getByRole("heading", { name: "Karyawan Lifecycle E2E Diperbarui" })).toBeVisible();

    await page.getByRole("link", { name: /Kembali ke daftar/ }).click();
    await page.getByLabel("Cari nama atau NIP").fill("E2E-EMP-001");
    const activeRow = page.getByRole("row", { name: /E2E-EMP-001/ });
    await expect(activeRow).toContainText("Karyawan Lifecycle E2E Diperbarui");
    await activeRow.getByRole("link", { name: /Lihat detail/ }).click();

    await page.getByRole("button", { name: "Nonaktifkan", exact: true }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("bukan dihapus permanen");
    await dialog.getByRole("button", { name: "Nonaktifkan karyawan" }).click();

    await expect(page).toHaveURL(/\/app\/karyawan\?status=nonaktif$/);
    await page.getByLabel("Cari nama atau NIP").fill("E2E-EMP-001");
    const inactiveRow = page.getByRole("row", { name: /E2E-EMP-001/ });
    await expect(inactiveRow).toContainText("Nonaktif");

    // Bulk CSV membuat akun dan email domain perusahaan secara otomatis.
    await page.getByRole("button", { name: "Bulk upload" }).click();
    const bulkDialog = page.getByRole("dialog", { name: "Bulk upload karyawan" });
    const bulkCSV = [
      "nip,nama,email,jenis_kelamin,tanggal_lahir,tanggal_join,department_id,position_id,atasan_id,status_pernikahan,role,jalan,kota,provinsi,nomor_ktp,nomor_kontrak,jenis_kontrak,tanggal_mulai_kontrak,tanggal_berakhir_kontrak",
      `E2E-BULK-001,Pegawai Bulk E2E,,P,1994-04-04,2026-08-03,${departmentID},${positionID},,lajang,karyawan,Jalan Bulk,Jakarta,DKI Jakarta,3174000000000002,E2E-CONTRACT-BULK,PKWT,2026-08-03,2027-08-02`,
    ].join("\n");
    await bulkDialog.locator('input[type="file"]').setInputFiles({
      name: "bulk-karyawan-e2e.csv",
      mimeType: "text/csv",
      buffer: Buffer.from(bulkCSV),
    });
    await bulkDialog.getByRole("button", { name: "Unggah", exact: true }).click();
    await expect(bulkDialog.getByRole("status")).toContainText("1 karyawan dibuat, 0 gagal");
    await expect(bulkDialog.getByText(/^Baris 2: .*@janjikupadamu\.id$/)).toBeVisible();
    await bulkDialog.getByRole("button", { name: "Tutup" }).click();

    // HR dapat mengubah batas master dan hasilnya langsung dipakai form pengajuan.
    await page.goto("/app/master/jenis-izin");
    await page.getByRole("button", { name: "Edit Ibadah Haji" }).first().click();
    const leaveDialog = page.getByRole("dialog", { name: /Edit Ibadah Haji/i });
    await leaveDialog.getByLabel("Maksimal hari izin").fill("29");
    await leaveDialog.getByRole("button", { name: "Simpan perubahan" }).click();
    await expect(leaveDialog).toHaveCount(0);
    await page.goto("/app/absensi/ketidakhadiran");
    await page.getByLabel("Jenis izin").selectOption({ label: "Ibadah Haji" });
    await expect(page.getByText("Maksimal 29 hari per pengajuan", { exact: true })).toBeVisible();

    expect(apiFailures).toEqual([]);

    await page.getByRole("button", { name: "Keluar" }).click();
    await expect(page).toHaveURL(/\/login$/);
  });
});
