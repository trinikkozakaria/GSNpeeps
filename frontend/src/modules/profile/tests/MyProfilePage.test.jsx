import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { employeeDetailFixture } from "../../employees/tests/employee-fixtures";
import { MyProfilePage } from "../pages/MyProfilePage";

const { profileState } = vi.hoisted(() => ({ profileState: { current: {} } }));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => ({ role: "karyawan", user: { id: "user-1" } }),
}));

vi.mock("../hooks/useProfile", () => ({
  useMyProfile: () => profileState.current,
}));

describe("MyProfilePage", () => {
  beforeEach(() => {
    profileState.current = {
      data: employeeDetailFixture,
      isPending: false,
      isError: false,
    };
  });

  it("renders the profile read-only without any edit affordance", () => {
    render(<MyProfilePage />);

    expect(screen.getByRole("heading", { name: "Profil Saya" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /edit/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /edit/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /simpan/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it("directs data changes through HR", () => {
    render(<MyProfilePage />);

    expect(screen.getByText(/ajukan\s+perubahan melalui HR/i)).toBeInTheDocument();
  });

  it("shows only the current month salary and no full history", () => {
    render(<MyProfilePage />);

    expect(screen.getByRole("heading", { name: "Gaji bulan berjalan" })).toBeInTheDocument();
    expect(screen.getByText("Agustus 2026")).toBeInTheDocument();
    expect(screen.getByText(/riwayat gaji penuh tidak tersedia di sini/i)).toBeInTheDocument();
    expect(screen.queryByText(/riwayat gaji/i, { selector: "h2" })).not.toBeInTheDocument();
  });

  it("does not render a document section that the profile response never returns", () => {
    render(<MyProfilePage />);

    expect(screen.queryByRole("heading", { name: "Dokumen karyawan" })).not.toBeInTheDocument();
  });

  it("shows a retryable error state", () => {
    profileState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    render(<MyProfilePage />);

    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia.");
    expect(screen.getByRole("button", { name: "Coba lagi" })).toBeInTheDocument();
  });
});
