import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ProtectedDocumentPreview, ProtectedImage } from "./ProtectedImage";

const { protectedMediaRequestMock } = vi.hoisted(() => ({
  protectedMediaRequestMock: vi.fn(),
}));

vi.mock("../../lib/api/client", () => ({
  protectedMediaRequest: protectedMediaRequestMock,
}));

describe("ProtectedImage", () => {
  const createObjectURL = vi.fn(() => "blob:protected-image");
  const revokeObjectURL = vi.fn();

  beforeEach(() => {
    protectedMediaRequestMock.mockReset();
    createObjectURL.mockClear();
    revokeObjectURL.mockClear();
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL });
  });

  afterEach(() => vi.unstubAllGlobals());

  it("uses the object URL returned for a successfully fetched media blob", async () => {
    const blob = new Blob(["image bytes"], { type: "image/jpeg" });
    protectedMediaRequestMock.mockResolvedValue(blob);

    const view = render(
      <ProtectedImage path="attendance-photos/user-1/check-in.jpg" alt="Foto absensi" />,
    );

    const image = screen.getByAltText("Foto absensi");
    expect(image).toHaveAttribute("aria-busy", "true");
    expect(image.getAttribute("src")).toMatch(/^data:image\/svg\+xml/);

    await waitFor(() => {
      expect(protectedMediaRequestMock).toHaveBeenCalledWith(
        "attendance-photos/user-1/check-in.jpg",
        expect.any(AbortSignal),
      );
      expect(createObjectURL).toHaveBeenCalledWith(blob);
      expect(image).toHaveAttribute("src", "blob:protected-image");
    });

    expect(image).not.toHaveAttribute("aria-busy");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    view.unmount();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:protected-image");
  });

  it("shows a controlled fallback when the media endpoint returns HTML", async () => {
    protectedMediaRequestMock.mockResolvedValue(new Blob(["<html>setup</html>"], { type: "text/html" }));

    const view = render(<ProtectedImage path="employee-photos/user-1/photo.jpg" alt="Foto profil" />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Gambar gagal dimuat");
    expect(screen.getByAltText("Foto profil").getAttribute("src")).toMatch(/^data:image\/svg\+xml/);
    view.unmount();
  });

  it("renders protected image documents as a thumbnail and download", async () => {
    protectedMediaRequestMock.mockResolvedValue(new Blob(["image"], { type: "image/png" }));

    const view = render(<ProtectedDocumentPreview path="employees/doc.png" fileName="KTP.png" />);

    const preview = await screen.findByAltText("Pratinjau KTP.png");
    expect(preview).toHaveAttribute("src", "blob:protected-image");
    expect(screen.getByRole("link", { name: "Buka pratinjau KTP.png" })).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("link", { name: "Unduh KTP.png" })).toHaveAttribute("download", "KTP.png");
    view.unmount();
  });

  it("renders protected PDF documents inline", async () => {
    protectedMediaRequestMock.mockResolvedValue(new Blob(["pdf"], { type: "application/pdf" }));

    const view = render(<ProtectedDocumentPreview path="employees/contract.pdf" fileName="Kontrak.pdf" />);

    const preview = await screen.findByTitle("Pratinjau Kontrak.pdf");
    expect(preview).toHaveAttribute("src", "blob:protected-image");
    expect(screen.getByRole("link", { name: "Unduh Kontrak.pdf" })).toBeInTheDocument();
    view.unmount();
  });
});
