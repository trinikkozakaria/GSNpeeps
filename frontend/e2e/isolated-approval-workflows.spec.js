import { expect, test } from "@playwright/test";

const liveMode = process.env.E2E_LIVE === "1";
const mutationMode = process.env.E2E_MUTATION === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;
const pdf = {
  name: "approval-e2e.pdf",
  mimeType: "application/pdf",
  buffer: Buffer.from("%PDF-1.4\n%%EOF"),
};

const bearer = (token) => ({ Authorization: `Bearer ${token}` });

const login = async (request, email) => {
  const response = await request.post("/api/v1/auth/login", {
    data: { email, password: seedPassword },
  });
  expect(response.status()).toBe(200);
  return (await response.json()).data;
};

const createLeave = (request, token, leaveTypeID, suffix, document) => {
  const multipart = {
    jenis_izin_id: leaveTypeID,
    tanggal_mulai: `2026-09-${suffix}`,
    tanggal_selesai: `2026-09-${suffix}`,
    alasan: `Pengajuan sintetis E2E nomor ${suffix}`,
  };
  if (document) multipart.dokumen_pendukung = document;
  return request.post("/api/v1/ketidakhadiran", { headers: bearer(token), multipart });
};

const decideLeave = (request, token, id, keputusan, catatan) =>
  request.put(`/api/v1/ketidakhadiran/${id}/decision`, {
    headers: bearer(token),
    data: catatan ? { keputusan, catatan } : { keputusan },
  });

const expectState = async (response, status, state) => {
  expect(response.status()).toBe(status);
  const body = await response.json();
  expect(body).toMatchObject({ success: true, data: { status: state } });
  return body.data.id;
};

test.describe("isolated approval workflows", () => {
  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise a real stack.");
    test.skip(!mutationMode, "Set E2E_MUTATION=1 only for a disposable stack.");
    test.skip(testInfo.project.name !== "chromium", "The isolated workflow needs one project.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();
    expect(new URL(testInfo.project.use.baseURL).port).not.toBe("8080");
  });

  test("all leave routing rows, rejection, delegation, concurrency, and overtime pass", async ({ request }) => {
    const employee = await login(request, "karyawan@example.test");
    const employeeWithoutSupervisor = await login(request, "karyawan.tanpa.atasan@example.test");
    const supervisor = await login(request, "atasan@example.test");
    const hr = await login(request, "hr@example.test");
    const top = await login(request, "top.management@example.test");

    const typeResponse = await request.get("/api/v1/master/jenis-izin?aktif=true", {
      headers: bearer(hr.token),
    });
    expect(typeResponse.status()).toBe(200);
    const types = (await typeResponse.json()).data;
    const approvalType = types.find((item) => item.kode === "IZIN-SYN-E2E");
    const documentType = types.find((item) => item.kode === "CUTI-SYN-E2E");
    expect(approvalType).toBeTruthy();
    expect(documentType).toMatchObject({ memerlukan_dokumen: true });

    // Karyawan dengan atasan: Atasan -> HR.
    const supervisedID = await expectState(
      await createLeave(request, employee.token, approvalType.id, "01"),
      201,
      "menunggu_atasan",
    );
    expect(
      (
        await decideLeave(request, employee.token, supervisedID, "setujui")
      ).status(),
    ).toBe(403);
    const supervisorInbox = await request.get("/api/v1/ketidakhadiran", {
      headers: bearer(supervisor.token),
    });
    expect(supervisorInbox.status()).toBe(200);
    expect((await supervisorInbox.json()).data.map((item) => item.id)).toContain(supervisedID);
    await expectState(
      await decideLeave(request, supervisor.token, supervisedID, "setujui"),
      200,
      "menunggu_hr",
    );
    const staleSupervisorDecision = await decideLeave(
      request,
      supervisor.token,
      supervisedID,
      "setujui",
    );
    // Setelah tahap berpindah ke HR, atasan tidak lagi berwenang mengambil keputusan.
    expect(staleSupervisorDecision.status()).toBe(403);
    await expectState(
      await decideLeave(request, hr.token, supervisedID, "setujui"),
      200,
      "disetujui",
    );
    const supervisedDetail = await request.get(`/api/v1/ketidakhadiran/${supervisedID}`, {
      headers: bearer(employee.token),
    });
    expect(supervisedDetail.status()).toBe(200);
    expect((await supervisedDetail.json()).data.approval_history).toHaveLength(2);

    // Karyawan tanpa atasan langsung menuju HR; reject wajib catatan dan mengakhiri alur.
    const directHRID = await expectState(
      await createLeave(request, employeeWithoutSupervisor.token, approvalType.id, "02"),
      201,
      "menunggu_hr",
    );
    const rejectWithoutNote = await decideLeave(request, hr.token, directHRID, "tolak");
    expect(rejectWithoutNote.status()).toBe(422);
    await expectState(
      await decideLeave(request, hr.token, directHRID, "tolak", "Ditolak untuk pengujian E2E"),
      200,
      "ditolak",
    );
    expect((await decideLeave(request, hr.token, directHRID, "setujui")).status()).toBe(409);

    // Atasan mengajukan miliknya sendiri: langsung HR.
    const supervisorOwnID = await expectState(
      await createLeave(request, supervisor.token, approvalType.id, "03"),
      201,
      "menunggu_hr",
    );
    await expectState(
      await decideLeave(request, hr.token, supervisorOwnID, "setujui"),
      200,
      "disetujui",
    );

    // HR mengajukan miliknya sendiri: hanya Top Management yang boleh memutuskan.
    const hrOwnID = await expectState(
      await createLeave(request, hr.token, approvalType.id, "04"),
      201,
      "menunggu_top_management",
    );
    expect((await decideLeave(request, hr.token, hrOwnID, "setujui")).status()).toBe(403);
    await expectState(
      await decideLeave(request, top.token, hrOwnID, "setujui"),
      200,
      "disetujui",
    );

    // Delegasi hanya dari Atasan aktif dan menghasilkan histori immutable sebelum HR final.
    const delegatedID = await expectState(
      await createLeave(request, employee.token, approvalType.id, "05"),
      201,
      "menunggu_atasan",
    );
    const delegated = await request.put(`/api/v1/ketidakhadiran/${delegatedID}/delegate`, {
      headers: bearer(supervisor.token),
      data: {
        delegate_to: hr.user.id,
        catatan: "Delegasi sintetis ke HR",
      },
    });
    await expectState(delegated, 200, "menunggu_hr");
    await expectState(
      await decideLeave(request, hr.token, delegatedID, "setujui"),
      200,
      "disetujui",
    );
    const delegatedDetail = await request.get(`/api/v1/ketidakhadiran/${delegatedID}`, {
      headers: bearer(employee.token),
    });
    expect((await delegatedDetail.json()).data.approval_history[0]).toMatchObject({
      tahap: "atasan",
      keputusan: "didelegasikan",
    });

    // Dua keputusan nyata dikirim bersamaan; tepat satu menang.
    const concurrentID = await expectState(
      await createLeave(request, employeeWithoutSupervisor.token, approvalType.id, "06"),
      201,
      "menunggu_hr",
    );
    const concurrent = await Promise.all([
      decideLeave(request, hr.token, concurrentID, "setujui"),
      decideLeave(request, hr.token, concurrentID, "tolak", "Keputusan pesaing E2E"),
    ]);
    expect(concurrent.map((response) => response.status()).sort()).toEqual([200, 409]);
    const losingBody = await concurrent.find((response) => response.status() === 409).json();
    expect(losingBody.error.code).toBe("ALREADY_DECIDED");

    // Master yang mewajibkan dokumen menolak request kosong dan menerima PDF valid.
    const missingDocument = await createLeave(
      request,
      employee.token,
      documentType.id,
      "07",
    );
    expect(missingDocument.status()).toBe(422);
    expect((await missingDocument.json()).error.fields).toHaveProperty("dokumen_pendukung");
    await expectState(
      await createLeave(request, employee.token, documentType.id, "07", pdf),
      201,
      "menunggu_atasan",
    );

    // Lembur tidak membutuhkan dokumen dan mengikuti Atasan -> HR tanpa field kompensasi.
    const overtime = await request.post("/api/v1/lembur", {
      headers: bearer(employee.token),
      multipart: {
        tanggal: "2026-09-20",
        waktu_mulai: "18:00",
        waktu_selesai: "20:00",
        alasan: "Pengajuan lembur sintetis tanpa dokumen",
      },
    });
    const overtimeID = await expectState(overtime, 201, "menunggu_atasan");
    await expectState(
      await request.put(`/api/v1/lembur/${overtimeID}/decision`, {
        headers: bearer(supervisor.token),
        data: { keputusan: "setujui" },
      }),
      200,
      "menunggu_hr",
    );
    await expectState(
      await request.put(`/api/v1/lembur/${overtimeID}/decision`, {
        headers: bearer(hr.token),
        data: { keputusan: "setujui" },
      }),
      200,
      "disetujui",
    );
    const overtimeDetail = await request.get(`/api/v1/lembur/${overtimeID}`, {
      headers: bearer(employee.token),
    });
    expect(overtimeDetail.status()).toBe(200);
    const overtimeData = (await overtimeDetail.json()).data;
    expect(overtimeData).not.toHaveProperty("kompensasi");
    expect(overtimeData.dokumen_url).toBeNull();

    const recap = await request.get("/api/v1/lembur/rekap", { headers: bearer(hr.token) });
    expect(recap.status()).toBe(200);
    expect((await recap.json()).data.some((item) => item.employee_id === employee.user.employee_id)).toBe(true);

    for (const account of [employee, employeeWithoutSupervisor, supervisor, hr, top]) {
      await request.post("/api/v1/auth/logout", { headers: bearer(account.token) });
    }
  });
});
