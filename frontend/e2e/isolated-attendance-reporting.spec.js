import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const mutationMode = process.env.E2E_MUTATION === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;
const bearer = (token) => ({ Authorization: `Bearer ${token}` });
const photo = {
  name: "attendance-reporting-e2e.png",
  mimeType: "image/png",
  buffer: Buffer.from(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
    "base64",
  ),
};

const login = async (request, email) => {
  const response = await request.post("/api/v1/auth/login", {
    data: { email, password: seedPassword },
  });
  expect(response.status()).toBe(200);
  return (await response.json()).data.token;
};

test("HR feed/report/export and Top Management read-only dashboard remain coherent", async ({ request }, testInfo) => {
  test.skip(!liveMode || !mutationMode, "Run only against a disposable real stack.");
  test.skip(testInfo.project.name !== "chromium", "The isolated workflow needs one project.");
  expect(new URL(testInfo.project.use.baseURL).port).not.toBe("8080");
  expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();

  const employeeToken = await login(request, "karyawan.tanpa.atasan@example.test");
  const hrToken = await login(request, "hr@example.test");
  const topToken = await login(request, "top.management@example.test");

  const checkIn = await request.post("/api/v1/absensi/checkin", {
    headers: bearer(employeeToken),
    multipart: {
      tipe: "check_in",
      mode_kerja: "WFA",
      gps_lat: "-7.25",
      gps_long: "112.75",
      foto: photo,
    },
  });
  expect(checkIn.status()).toBe(201);
  const attendance = (await checkIn.json()).data;

  expect(
    (
      await request.get("/api/v1/absensi/livefeed?tanggal=2026-08-03", {
        headers: bearer(employeeToken),
      })
    ).status(),
  ).toBe(403);

  for (const token of [hrToken, topToken]) {
    const feed = await request.get("/api/v1/absensi/livefeed?tanggal=2026-08-03", {
      headers: bearer(token),
    });
    expect(feed.status()).toBe(200);
    expect((await feed.json()).data.map((item) => item.id)).toContain(attendance.id);

    const report = await request.get("/api/v1/laporan/kehadiran?periode=2026-08&limit=100", {
      headers: bearer(token),
    });
    expect(report.status()).toBe(200);
    const reportBody = await report.json();
    expect(reportBody.meta.total_data).toBeGreaterThan(0);
    expect(reportBody.data.some((item) => item.employee_id === attendance.employee_id)).toBe(true);

    const dashboard = await request.get(
      "/api/v1/dashboard/metrik?periode=bulanan&tanggal_acuan=2026-08-03",
      { headers: bearer(token) },
    );
    expect(dashboard.status()).toBe(200);
    expect(await dashboard.json()).toMatchObject({
      success: true,
      data: {
        periode: {
          tipe: "bulanan",
          tanggal_mulai: "2026-08-01",
          tanggal_selesai: "2026-08-31",
          timezone: "Asia/Jakarta",
        },
      },
    });
  }

  const topExport = await request.get(
    "/api/v1/laporan/kehadiran/export?periode=2026-08&format=xlsx",
    { headers: bearer(topToken) },
  );
  expect(topExport.status()).toBe(403);

  for (const format of ["xlsx", "pdf"]) {
    const exported = await request.get(
      `/api/v1/laporan/kehadiran/export?periode=2026-08&format=${format}`,
      { headers: bearer(hrToken) },
    );
    expect(exported.status()).toBe(200);
    expect(exported.headers()["content-disposition"]).toContain(`.${format}`);
    const bytes = await exported.body();
    expect(bytes.length).toBeGreaterThan(100);
    expect(format === "xlsx" ? bytes.subarray(0, 2).toString() : bytes.subarray(0, 4).toString()).toBe(
      format === "xlsx" ? "PK" : "%PDF",
    );
  }
});
