import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const mutationMode = process.env.E2E_MUTATION === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;
const photo = {
  name: "attendance-e2e.png",
  mimeType: "image/png",
  buffer: Buffer.from(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
    "base64",
  ),
};

const bearer = (token) => ({ Authorization: `Bearer ${token}` });

const login = async (request, email) => {
  const response = await request.post("/api/v1/auth/login", {
    data: { email, password: seedPassword },
  });
  expect(response.status()).toBe(200);
  return (await response.json()).data.token;
};

const record = (request, token, fields) =>
  request.post("/api/v1/absensi/checkin", {
    headers: bearer(token),
    multipart: { ...fields, foto: photo },
  });

const expectDomainError = async (response, code) => {
  expect(response.status()).toBe(422);
  expect(await response.json()).toMatchObject({ success: false, error: { code } });
};

test.describe("isolated attendance workflows", () => {
  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise a real stack.");
    test.skip(!mutationMode, "Set E2E_MUTATION=1 only for a disposable stack.");
    test.skip(testInfo.project.name !== "chromium", "The isolated workflow needs one browser project.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();
    expect(new URL(testInfo.project.use.baseURL).port).not.toBe("8080");
  });

  test("WFO radius, duplicate/sequence errors, WFH/WFA, and camera fallback are enforced", async ({
    context,
    page,
    request,
  }) => {
    const employeeToken = await login(request, "karyawan@example.test");
    const offices = await request.get("/api/v1/master/lokasi-kantor", {
      headers: bearer(employeeToken),
    });
    expect(offices.status()).toBe(200);
    const office = (await offices.json()).data[0];
    expect(office).toMatchObject({ kode: "OFFICE-SYN-001", is_active: true });

    await expectDomainError(
      await record(request, employeeToken, {
        tipe: "check_in",
        mode_kerja: "WFO",
        gps_lat: String(office.latitude + 0.01),
        gps_long: String(office.longitude),
        office_location_id: office.id,
      }),
      "OUT_OF_RADIUS",
    );

    const inside = await record(request, employeeToken, {
      tipe: "check_in",
      mode_kerja: "WFO",
      gps_lat: String(office.latitude),
      gps_long: String(office.longitude),
      office_location_id: office.id,
    });
    expect(inside.status()).toBe(201);
    expect(await inside.json()).toMatchObject({
      success: true,
      data: { tipe: "check_in", mode_kerja: "WFO", office_location_id: office.id },
    });

    await expectDomainError(
      await record(request, employeeToken, {
        tipe: "check_in",
        mode_kerja: "WFO",
        gps_lat: String(office.latitude),
        gps_long: String(office.longitude),
        office_location_id: office.id,
      }),
      "DUPLICATE_CHECKIN",
    );
    expect(
      (
        await record(request, employeeToken, {
          tipe: "check_out",
          mode_kerja: "WFO",
          gps_lat: String(office.latitude),
          gps_long: String(office.longitude),
          office_location_id: office.id,
        })
      ).status(),
    ).toBe(201);
    await request.post("/api/v1/auth/logout", { headers: bearer(employeeToken) });

    const supervisorToken = await login(request, "atasan@example.test");
    await expectDomainError(
      await record(request, supervisorToken, {
        tipe: "check_out",
        mode_kerja: "WFH",
        gps_lat: "-8.65",
        gps_long: "115.21",
      }),
      "CHECKOUT_WITHOUT_CHECKIN",
    );

    const topOrigin = new URL(test.info().project.use.baseURL).origin;
    await context.grantPermissions(["geolocation"], { origin: topOrigin });
    await context.setGeolocation({ latitude: -8.65, longitude: 115.21 });
    await page.addInitScript(() => {
      Object.defineProperty(navigator, "mediaDevices", {
        configurable: true,
        value: {
          getUserMedia: async () => {
            throw new DOMException("Camera denied for E2E", "NotAllowedError");
          },
        },
      });
    });
    await page.goto("/login");
    await page.getByLabel("Email kerja").fill("atasan@example.test");
    await page.getByLabel(/^Password$/).fill(seedPassword);
    await page.getByRole("button", { name: "Masuk" }).click();
    await page.getByRole("navigation", { name: "Navigasi utama" })
      .getByRole("link", { name: "Kehadiran Saya", exact: true })
      .click();
    await page.getByLabel("WFH", { exact: true }).check();
    await page.getByRole("button", { name: "Nyalakan kamera" }).click();
    await expect(page.getByText(/kamera ditolak/i)).toBeVisible();
    await page.getByLabel("Unggah foto absensi").setInputFiles(photo);
    await page.getByRole("button", { name: "Kirim absensi" }).click();
    const resultPanel = page.getByRole("region", { name: "Absensi tercatat" });
    await expect(resultPanel).toBeVisible();
    await expect(resultPanel.getByText("WFH", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Keluar" }).click();

    const hrToken = await login(request, "hr@example.test");
    const wfa = await record(request, hrToken, {
      tipe: "check_in",
      mode_kerja: "WFA",
      gps_lat: "-7.25",
      gps_long: "112.75",
    });
    expect(wfa.status()).toBe(201);
    expect(await wfa.json()).toMatchObject({
      success: true,
      data: { mode_kerja: "WFA", office_location_id: null },
    });
    await request.post("/api/v1/auth/logout", { headers: bearer(hrToken) });
  });
});
