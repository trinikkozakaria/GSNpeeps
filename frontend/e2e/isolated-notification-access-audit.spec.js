import { expect, test } from "@playwright/test";
import { assertDisposableMutationTarget } from "./helpers/mutation-target";

const liveMode = process.env.E2E_LIVE === "1";
const mutationMode = process.env.E2E_MUTATION === "1";
const seedPassword = process.env.E2E_SEED_PASSWORD;
const bearer = (token) => ({ Authorization: `Bearer ${token}` });

const login = async (request, email) => {
  const response = await request.post("/api/v1/auth/login", {
    data: { email, password: seedPassword },
  });
  expect(response.status()).toBe(200);
  return (await response.json()).data;
};

test.describe("isolated notification, access, and audit workflows", () => {
  test.beforeEach(async ({}, testInfo) => {
    test.skip(!liveMode, "Set E2E_LIVE=1 to exercise a real stack.");
    test.skip(!mutationMode, "Set E2E_MUTATION=1 only for a disposable stack.");
    test.skip(testInfo.project.name !== "chromium", "The isolated workflow needs one project.");
    expect(seedPassword, "E2E_SEED_PASSWORD must be provided").toBeTruthy();
    assertDisposableMutationTarget(testInfo.project.use.baseURL);
  });

  test("notifications stay recipient-scoped and HR permission changes are audited", async ({ request }) => {
    const employee = await login(request, "karyawan@example.test");
    const supervisor = await login(request, "atasan@example.test");
    const hr = await login(request, "hr@example.test");
    const top = await login(request, "top.management@example.test");

    const typeResponse = await request.get("/api/v1/master/jenis-izin?aktif=true", {
      headers: bearer(hr.token),
    });
    expect(typeResponse.status()).toBe(200);
    const approvalType = (await typeResponse.json()).data.find(
      (item) => item.kode === "IZIN-SYN-E2E",
    );
    const submitted = await request.post("/api/v1/ketidakhadiran", {
      headers: bearer(employee.token),
      multipart: {
        jenis_izin_id: approvalType.id,
        tanggal_mulai: "2026-09-08",
        tanggal_selesai: "2026-09-08",
        alasan: "Pengajuan khusus verifikasi notifikasi E2E",
      },
    });
    expect(submitted.status()).toBe(201);
    const submittedID = (await submitted.json()).data.id;

    const supervisorInbox = await request.get("/api/v1/notifikasi?limit=100", {
      headers: bearer(supervisor.token),
    });
    expect(supervisorInbox.status()).toBe(200);
    const supervisorItems = (await supervisorInbox.json()).data;
    const incoming = supervisorItems.filter(
      (item) =>
        item.tipe === "ketidakhadiran_baru" &&
        item.reference_type === "ketidakhadiran" &&
        item.reference_id === submittedID,
    );
    expect(incoming).toHaveLength(1);
    for (const notification of incoming) {
      expect(notification.reference_id).toBeTruthy();
    }

    const target = incoming[0];
    expect(supervisorItems.filter((item) => item.id === target.id)).toHaveLength(1);
    const foreignRead = await request.put(`/api/v1/notifikasi/${target.id}/read`, {
      headers: bearer(employee.token),
    });
    // D-032 menyamakan ID asing dan ID tidak ada agar keberadaan notifikasi tidak bocor.
    expect(foreignRead.status()).toBe(403);

    const unreadBefore = await request.get("/api/v1/notifikasi/unread-count", {
      headers: bearer(supervisor.token),
    });
    expect(unreadBefore.status()).toBe(200);
    const beforeCount = (await unreadBefore.json()).data.unread_count;
    expect(beforeCount).toBeGreaterThan(0);

    const markRead = await request.put(`/api/v1/notifikasi/${target.id}/read`, {
      headers: bearer(supervisor.token),
    });
    expect(markRead.status()).toBe(200);
    const unreadAfter = await request.get("/api/v1/notifikasi/unread-count", {
      headers: bearer(supervisor.token),
    });
    expect((await unreadAfter.json()).data.unread_count).toBe(beforeCount - 1);

    const dismiss = await request.delete(`/api/v1/notifikasi/${target.id}`, {
      headers: bearer(supervisor.token),
    });
    expect(dismiss.status()).toBe(200);
    const afterDismiss = await request.get("/api/v1/notifikasi?limit=100", {
      headers: bearer(supervisor.token),
    });
    expect((await afterDismiss.json()).data.map((item) => item.id)).not.toContain(target.id);

    const rolesResponse = await request.get("/api/v1/akses/role", { headers: bearer(hr.token) });
    const matrixResponse = await request.get("/api/v1/akses/permission", {
      headers: bearer(hr.token),
    });
    expect(rolesResponse.status()).toBe(200);
    expect(matrixResponse.status()).toBe(200);
    const roles = (await rolesResponse.json()).data;
    const permissions = (await matrixResponse.json()).data;
    const topRole = roles.find((role) => role.nama === "top_management");
    const topAccessRead = permissions.find(
      (permission) =>
        permission.role_id === topRole.id &&
        permission.modul === "akses" &&
        permission.aksi === "read",
    );
    expect(topAccessRead).toMatchObject({ is_allowed: true });

    const topMatrix = await request.get("/api/v1/akses/permission", {
      headers: bearer(top.token),
    });
    // Top Management hanya menerima dashboard ringkas; matriks akses tetap khusus HR.
    expect(topMatrix.status()).toBe(403);
    const forbiddenTopMutation = await request.put("/api/v1/akses/permission", {
      headers: bearer(top.token),
      data: {
        role_id: topRole.id,
        modul: "akses",
        aksi: "read",
        is_allowed: false,
      },
    });
    expect(forbiddenTopMutation.status()).toBe(403);

    const disable = await request.put("/api/v1/akses/permission", {
      headers: bearer(hr.token),
      data: {
        role_id: topRole.id,
        modul: "akses",
        aksi: "read",
        is_allowed: false,
      },
    });
    expect(disable.status()).toBe(200);
    const disabledMatrix = await request.get("/api/v1/akses/permission", {
      headers: bearer(hr.token),
    });
    expect(disabledMatrix.status()).toBe(200);
    expect(
      (await disabledMatrix.json()).data.find((permission) => permission.id === topAccessRead.id),
    ).toMatchObject({ is_allowed: false });

    const restore = await request.put("/api/v1/akses/permission", {
      headers: bearer(hr.token),
      data: {
        role_id: topRole.id,
        modul: "akses",
        aksi: "read",
        is_allowed: true,
      },
    });
    expect(restore.status()).toBe(200);
    const restoredMatrix = await request.get("/api/v1/akses/permission", {
      headers: bearer(hr.token),
    });
    expect(restoredMatrix.status()).toBe(200);
    expect(
      (await restoredMatrix.json()).data.find((permission) => permission.id === topAccessRead.id),
    ).toMatchObject({ is_allowed: true });

    const auditResponse = await request.get("/api/v1/akses/audit-log?modul=akses&limit=100", {
      headers: bearer(hr.token),
    });
    expect(auditResponse.status()).toBe(200);
    const accessAudits = (await auditResponse.json()).data.filter(
      (item) => item.aksi === "PERMISSION_UPDATE",
    );
    expect(accessAudits.length).toBeGreaterThanOrEqual(2);
    expect(accessAudits.some((item) => item.detail?.sesudah === false)).toBe(true);
    expect(accessAudits.some((item) => item.detail?.sesudah === true)).toBe(true);

    for (const account of [employee, supervisor, hr, top]) {
      await request.post("/api/v1/auth/logout", { headers: bearer(account.token) });
    }
  });
});
