import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ProtectedImage } from "./ProtectedImage";

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
});
