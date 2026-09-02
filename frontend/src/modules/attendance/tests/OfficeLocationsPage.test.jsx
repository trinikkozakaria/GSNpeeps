import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OfficeLocationsPage } from "../pages/OfficeLocationsPage";

const { officesState, createMock, updateMock, deactivateMock } = vi.hoisted(() => ({
  officesState: { current: { data: [], isPending: false } },
  createMock: vi.fn(),
  updateMock: vi.fn(),
  deactivateMock: vi.fn(),
}));

vi.mock("../hooks/useAttendance", () => ({
  useOfficeLocations: () => officesState.current,
  useCreateOfficeLocation: () => ({ mutateAsync: createMock, isPending: false }),
  useUpdateOfficeLocation: () => ({ mutateAsync: updateMock, isPending: false }),
  useDeactivateOfficeLocation: () => ({ mutate: deactivateMock, mutateAsync: deactivateMock, isPending: false }),
}));

const rows = [
  { id: "loc-1", kode: "HQ", nama: "Kantor Pusat", alamat: "Jl. Sudirman 1", latitude: -6.2, longitude: 106.8 },
  { id: "loc-2", kode: "BDG", nama: "Kantor Bandung", alamat: null, latitude: -6.9, longitude: 107.6 },
];

const table = () => within(screen.getByRole("region", { name: "Lokasi kantor aktif" }));

describe("OfficeLocationsPage", () => {
  beforeEach(() => {
    createMock.mockReset().mockResolvedValue({});
    updateMock.mockReset().mockResolvedValue({});
    deactivateMock.mockReset().mockResolvedValue({});
    officesState.current = { data: rows, isPending: false };
  });

  it("shows a loading state while master data is pending", () => {
    officesState.current = { data: undefined, isPending: true };
    render(<OfficeLocationsPage />);
    expect(screen.getByRole("status")).toHaveTextContent("Memuat lokasi kantor");
  });

  it("lists active office locations with coordinates and an address fallback", () => {
    render(<OfficeLocationsPage />);
    expect(table().getByText("Kantor Pusat")).toBeInTheDocument();
    expect(table().getByText("-6.2, 106.8")).toBeInTheDocument();
    expect(table().getByText("—")).toBeInTheDocument(); // alamat null
  });

  it("creates a location with a trimmed, numeric payload and null address when blank", async () => {
    const user = userEvent.setup();
    render(<OfficeLocationsPage />);

    await user.type(screen.getByLabelText("Kode"), "  SBY  ");
    await user.type(screen.getByLabelText("Nama"), "  Kantor Surabaya  ");
    await user.type(screen.getByLabelText("Latitude"), "-7.25");
    await user.type(screen.getByLabelText("Longitude"), "112.75");
    await user.click(screen.getByRole("button", { name: "Tambah lokasi" }));

    await waitFor(() => expect(createMock).toHaveBeenCalledTimes(1));
    expect(createMock).toHaveBeenCalledWith({
      kode: "SBY",
      nama: "Kantor Surabaya",
      alamat: null,
      latitude: -7.25,
      longitude: 112.75,
      is_active: true,
    });
    expect(updateMock).not.toHaveBeenCalled();
  });

  it("loads a row into the form and updates it by id", async () => {
    const user = userEvent.setup();
    render(<OfficeLocationsPage />);

    await user.click(table().getAllByRole("button", { name: "Edit" })[0]);
    expect(screen.getByLabelText("Kode")).toHaveValue("HQ");

    const nama = screen.getByLabelText("Nama");
    await user.clear(nama);
    await user.type(nama, "Kantor Pusat Baru");
    await user.click(screen.getByRole("button", { name: "Simpan lokasi" }));

    await waitFor(() => expect(updateMock).toHaveBeenCalledTimes(1));
    expect(updateMock).toHaveBeenCalledWith({
      id: "loc-1",
      payload: expect.objectContaining({ kode: "HQ", nama: "Kantor Pusat Baru", latitude: -6.2, longitude: 106.8 }),
    });
    expect(createMock).not.toHaveBeenCalled();
  });

  it("deactivates a row through its id", async () => {
    const user = userEvent.setup();
    render(<OfficeLocationsPage />);

    await user.click(table().getAllByRole("button", { name: "Nonaktifkan" })[0]);
    expect(deactivateMock).toHaveBeenCalledWith("loc-1");
  });

  it("surfaces a save error without losing the form", async () => {
    createMock.mockRejectedValueOnce(new Error("Kode atau nama sudah digunakan"));
    const user = userEvent.setup();
    render(<OfficeLocationsPage />);

    await user.type(screen.getByLabelText("Kode"), "HQ");
    await user.type(screen.getByLabelText("Nama"), "Duplikat");
    await user.type(screen.getByLabelText("Latitude"), "-6.2");
    await user.type(screen.getByLabelText("Longitude"), "106.8");
    await user.click(screen.getByRole("button", { name: "Tambah lokasi" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Kode atau nama sudah digunakan");
    expect(screen.getByLabelText("Kode")).toHaveValue("HQ");
  });
});
