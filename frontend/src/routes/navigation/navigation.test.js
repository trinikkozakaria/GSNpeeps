import { describe, expect, it } from "vitest";

import { navigationForRole, roles } from "./navigation";

const labels = (role) =>
  navigationForRole(role).flatMap((item) => [
    item.label,
    ...(item.children?.map((child) => child.label) ?? []),
  ]);

const childLabels = (role, groupLabel) =>
  navigationForRole(role).find((item) => item.label === groupLabel)?.children.map((child) => child.label) ?? [];

describe("role navigation", () => {
  it("keeps employee and supervisor away from HR access", () => {
    expect(labels(roles.employee)).not.toContain("AKSES");
    expect(labels(roles.supervisor)).not.toContain("Employee Database");
    expect(labels(roles.supervisor)).toContain("Persetujuan");
  });

  it("gives HR administration navigation", () => {
    expect(labels(roles.hr)).toContain("Employee Database");
    expect(labels(roles.hr)).toContain("AKSES");
    expect(labels(roles.hr)).toContain("Metrik Personal");
  });

  it("keeps Top Management read navigation without personal metrics", () => {
    expect(labels(roles.topManagement)).not.toContain("Dashboard HR");
    expect(labels(roles.topManagement)).not.toContain("AKSES");
    expect(labels(roles.topManagement)).toContain("Persetujuan");
    expect(labels(roles.topManagement)).not.toContain("Metrik Personal");
  });

  // Top Management tidak mengajukan koreksi (backend menolak) sehingga tidak punya
  // riwayat pribadi di menu "Pribadi", tetapi mengawasi dan memutus koreksi milik HR
  // lewat antrean di menu "Persetujuan".
  it("gives Top Management the correction queue via Persetujuan, not the personal page", () => {
    expect(childLabels(roles.topManagement, "Pribadi")).not.toContain("Koreksi Absensi");
    expect(childLabels(roles.topManagement, "Persetujuan")).toContain("Koreksi Absensi");
  });

  it("fails closed for an unknown role", () => {
    expect(navigationForRole("super_admin")).toEqual([]);
  });
});

