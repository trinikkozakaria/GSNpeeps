import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CheckInPage } from "../pages/CheckInPage";

const { recordMock, officesState } = vi.hoisted(() => ({
  recordMock: vi.fn(),
  officesState: { current: { data: [] } },
}));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => ({ role: "karyawan", user: { id: "user-1", nama: "Karyawan Uji" } }),
}));

vi.mock("../hooks/useAttendance", () => ({
  useOfficeLocations: () => officesState.current,
  useRecordAttendance: () => ({ mutateAsync: recordMock, isPending: false }),
  useLiveFeed: () => ({ data: [], isPending: false, isError: false }),
}));

const stopTrack = vi.fn();

const grantCamera = () => {
  const track = { stop: stopTrack };
  const stream = { getTracks: () => [track] };
  vi.stubGlobal("navigator", {
    ...navigator,
    mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(stream) },
    geolocation: {
      getCurrentPosition: (success) =>
        success({ coords: { latitude: -6.2, longitude: 106.8, accuracy: 10 } }),
    },
  });
  return stream;
};

const denyCamera = (name = "NotAllowedError") => {
  const error = new Error("denied");
  error.name = name;
  vi.stubGlobal("navigator", {
    ...navigator,
    mediaDevices: { getUserMedia: vi.fn().mockRejectedValue(error) },
    geolocation: {
      getCurrentPosition: (success) =>
        success({ coords: { latitude: -6.2, longitude: 106.8, accuracy: 10 } }),
    },
  });
};

const photoFile = () => {
  const file = new File(["x"], "absensi.jpg", { type: "image/jpeg" });
  Object.defineProperty(file, "size", { value: 2048 });
  return file;
};

describe("CheckInPage", () => {
  beforeEach(() => {
    recordMock.mockReset();
    recordMock.mockResolvedValue({
      id: "11111111-1111-4111-8111-111111111111",
      employee_id: "22222222-2222-4222-8222-222222222222",
      tanggal: "2026-08-03",
      tipe: "check_in",
      mode_kerja: "WFH",
      waktu: "2026-08-03T02:00:00Z",
      status: "tepat_waktu",
    });
    stopTrack.mockReset();
    officesState.current = { data: [{ id: "33333333-3333-4333-8333-333333333333", nama: "Kantor Uji" }] };
    // jsdom tidak mengimplementasikan canvas.toBlob; stub agar capture dapat diuji.
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({ drawImage: vi.fn() }));
    HTMLCanvasElement.prototype.toBlob = vi.fn((callback) =>
      callback(new Blob(["x"], { type: "image/jpeg" })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not request the camera before an explicit user action", () => {
    const stream = grantCamera();
    render(<CheckInPage />);

    expect(navigator.mediaDevices.getUserMedia).not.toHaveBeenCalled();
    expect(stream).toBeDefined();
  });

  it("opens the camera only after the user asks for it", async () => {
    grantCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));

    await waitFor(() => expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledTimes(1));
    expect(await screen.findByRole("button", { name: "Ambil foto" })).toBeInTheDocument();
  });

  it("stops every MediaStream track after capture", async () => {
    grantCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    await user.click(await screen.findByRole("button", { name: "Ambil foto" }));

    await waitFor(() => expect(stopTrack).toHaveBeenCalled());
    expect(await screen.findByText(/foto siap dikirim/i)).toBeInTheDocument();
  });

  it("releases the camera when the page unmounts", async () => {
    grantCamera();
    const user = userEvent.setup();
    const { unmount } = render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    await screen.findByRole("button", { name: "Ambil foto" });
    unmount();

    expect(stopTrack).toHaveBeenCalled();
  });

  it("offers the fallback upload when camera permission is denied", async () => {
    denyCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/akses kamera ditolak/i);
    expect(screen.getByLabelText("Unggah foto absensi")).toBeInTheDocument();
  });

  it("offers the fallback upload when the browser has no camera API", async () => {
    vi.stubGlobal("navigator", { ...navigator, mediaDevices: undefined });
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/tidak mendukung kamera/i);
    expect(screen.getByLabelText("Unggah foto absensi")).toBeInTheDocument();
  });

  it("requires an office location for WFO", async () => {
    denyCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    await user.upload(await screen.findByLabelText("Unggah foto absensi"), photoFile());
    await user.click(screen.getByRole("button", { name: "Kirim absensi" }));

    expect(await screen.findByText(/pilih lokasi kantor untuk mode kerja wfo/i)).toBeInTheDocument();
    expect(recordMock).not.toHaveBeenCalled();
  });

  // WFH dan WFA tetap mengirim koordinat tetapi tidak membawa lokasi kantor.
  it("submits WFH without an office location", async () => {
    denyCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    await user.click(screen.getByRole("radio", { name: "WFH" }));
    await user.upload(await screen.findByLabelText("Unggah foto absensi"), photoFile());
    await user.click(screen.getByRole("button", { name: "Kirim absensi" }));

    await waitFor(() => expect(recordMock).toHaveBeenCalledTimes(1));
    const payload = recordMock.mock.calls[0][0];
    expect(payload.mode_kerja).toBe("WFH");
    expect(payload.office_location_id).toBeUndefined();
    expect(payload.gps_lat).toBe(-6.2);
    expect(payload.gps_long).toBe(106.8);
  });

  it("rejects an unsupported fallback photo before calling the API", async () => {
    denyCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    const oversize = new File(["x"], "besar.jpg", { type: "image/jpeg" });
    Object.defineProperty(oversize, "size", { value: 6 * 1024 * 1024 });
    await user.upload(await screen.findByLabelText("Unggah foto absensi"), oversize);
    await user.click(screen.getByRole("button", { name: "Kirim absensi" }));

    expect(await screen.findByText("Ukuran foto melebihi batas 5 MB.")).toBeInTheDocument();
    expect(recordMock).not.toHaveBeenCalled();
  });

  it("explains attendance domain errors returned by the server", async () => {
    const cases = [
      ["OUT_OF_RADIUS", /di luar radius kantor/i],
      ["DUPLICATE_CHECKIN", /check-in hari ini sudah tercatat/i],
      ["CHECKOUT_WITHOUT_CHECKIN", /memerlukan check-in aktif/i],
      ["NON_WORKING_DAY", /senin sampai jumat/i],
    ];

    for (const [code, expected] of cases) {
      recordMock.mockReset();
      recordMock.mockRejectedValue({ code, message: "raw" });
      denyCamera();
      const user = userEvent.setup();
      const view = render(<CheckInPage />);

      await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
      await user.click(screen.getByRole("radio", { name: "WFA" }));
      await user.upload(await screen.findByLabelText("Unggah foto absensi"), photoFile());
      await user.click(screen.getByRole("button", { name: "Kirim absensi" }));

      expect(await screen.findByText(expected)).toBeInTheDocument();
      view.unmount();
    }
  });

  it("shows the server time and status after a successful submission", async () => {
    denyCamera();
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    await user.click(screen.getByRole("radio", { name: "WFH" }));
    await user.upload(await screen.findByLabelText("Unggah foto absensi"), photoFile());
    await user.click(screen.getByRole("button", { name: "Kirim absensi" }));

    expect(await screen.findByRole("heading", { name: "Absensi tercatat" })).toBeInTheDocument();
    expect(screen.getByText("Tepat waktu")).toBeInTheDocument();
    expect(screen.getByText(/menggunakan waktu server/i)).toBeInTheDocument();
  });

  it("reports a denied geolocation without calling the API", async () => {
    const error = new Error("denied");
    error.name = "NotAllowedError";
    vi.stubGlobal("navigator", {
      ...navigator,
      mediaDevices: { getUserMedia: vi.fn().mockRejectedValue(error) },
      geolocation: {
        getCurrentPosition: (_success, failure) => failure({ code: 1 }),
      },
    });
    const user = userEvent.setup();
    render(<CheckInPage />);

    await user.click(screen.getByRole("button", { name: "Nyalakan kamera" }));
    await user.click(screen.getByRole("radio", { name: "WFH" }));
    await user.upload(await screen.findByLabelText("Unggah foto absensi"), photoFile());
    await user.click(screen.getByRole("button", { name: "Kirim absensi" }));

    expect(await screen.findByText(/izin lokasi ditolak/i)).toBeInTheDocument();
    expect(recordMock).not.toHaveBeenCalled();
  });
});
