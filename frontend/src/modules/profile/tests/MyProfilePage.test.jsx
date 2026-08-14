import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { employeeDetailFixture } from "../../employees/tests/employee-fixtures";
import { MyProfilePage } from "../pages/MyProfilePage";

const { profileState, refreshUserMock, updatePhotoMock } = vi.hoisted(() => ({
  profileState: { current: {} },
  refreshUserMock: vi.fn(),
  updatePhotoMock: vi.fn(),
}));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => ({
    role: "karyawan",
    user: { id: "user-1" },
    refreshUser: refreshUserMock,
  }),
}));

vi.mock("../../auth/api/auth-api", () => ({
  updateMyPhotoRequest: updatePhotoMock,
}));

vi.mock("../../../components/media/ProtectedImage", () => ({
  ProtectedImage: ({ alt = "" }) => <img alt={alt} src="blob:test-photo" />,
}));

vi.mock("../hooks/useProfile", () => ({
  useMyProfile: () => profileState.current,
}));

describe("MyProfilePage", () => {
  beforeEach(() => {
    refreshUserMock.mockReset();
    refreshUserMock.mockResolvedValue({});
    updatePhotoMock.mockReset();
    updatePhotoMock.mockResolvedValue({ foto_profil_url: "profile/user-1/photo.jpg" });
    profileState.current = {
      data: employeeDetailFixture,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue({}),
    };
  });

  it("keeps employee data read-only while allowing a profile photo upload", () => {
    render(<MyProfilePage />);

    expect(screen.getByRole("heading", { name: "Profil Saya" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /edit/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /edit/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /simpan/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Ganti foto profil")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Unggah foto" })).toBeDisabled();
  });

  it("refreshes the profile and navbar identity after a photo upload", async () => {
    const user = userEvent.setup();
    render(<MyProfilePage />);
    const photo = new File(["photo"], "profil.png", { type: "image/png" });

    await user.upload(screen.getByLabelText("Ganti foto profil"), photo);
    await user.click(screen.getByRole("button", { name: "Unggah foto" }));

    await waitFor(() => expect(updatePhotoMock).toHaveBeenCalledWith(photo));
    expect(refreshUserMock).toHaveBeenCalledTimes(1);
    expect(profileState.current.refetch).toHaveBeenCalledTimes(1);
    expect(await screen.findByText("Foto profil berhasil diperbarui.")).toBeInTheDocument();
  });

  it("directs data changes through HR", () => {
    render(<MyProfilePage />);

    expect(screen.getByText(/perubahan data lainnya tetap diajukan melalui HR/i)).toBeInTheDocument();
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
