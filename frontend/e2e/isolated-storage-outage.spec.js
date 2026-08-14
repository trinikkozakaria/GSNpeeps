import { expect, test } from "@playwright/test";
import { assertDisposableMutationTarget } from "./helpers/mutation-target";

const liveMode = process.env.E2E_LIVE === "1";
const mutationMode = process.env.E2E_MUTATION === "1";
const outageMode = process.env.E2E_STORAGE_OUTAGE === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;

test("Nextcloud outage fails document upload without leaking internal details", async ({ request }, testInfo) => {
  test.skip(!liveMode || !mutationMode || !outageMode, "Run only while isolated Nextcloud is down.");
  test.skip(testInfo.project.name !== "chromium", "The isolated workflow needs one project.");
  assertDisposableMutationTarget(testInfo.project.use.baseURL);
  expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();

  const login = await request.post("/api/v1/auth/login", {
    data: { email: "karyawan@example.test", password: seedPassword },
  });
  expect(login.status()).toBe(200);
  const token = (await login.json()).data.token;
  const headers = { Authorization: `Bearer ${token}` };

  const hrLogin = await request.post("/api/v1/auth/login", {
    data: { email: "hr@example.test", password: seedPassword },
  });
  expect(hrLogin.status()).toBe(200);
  const hrHeaders = { Authorization: `Bearer ${(await hrLogin.json()).data.token}` };

  const typesResponse = await request.get("/api/v1/master/jenis-izin?aktif=true", {
    headers: hrHeaders,
  });
  expect(typesResponse.status()).toBe(200);
  const documentType = (await typesResponse.json()).data.find(
    (item) => item.kode === "CUTI-SYN-E2E",
  );

  const response = await request.post("/api/v1/ketidakhadiran", {
    headers,
    multipart: {
      jenis_izin_id: documentType.id,
      tanggal_mulai: "2026-09-09",
      tanggal_selesai: "2026-09-09",
      alasan: "Verifikasi kegagalan penyimpanan dokumen sintetis",
      dokumen_pendukung: {
        name: "storage-outage.pdf",
        mimeType: "application/pdf",
        buffer: Buffer.from("%PDF-1.4\n%%EOF"),
      },
    },
  });
  expect(response.status()).toBe(500);
  const body = await response.text();
  expect(body).not.toMatch(/nextcloud|webdav|password|credential|stack|dial tcp/i);
});
