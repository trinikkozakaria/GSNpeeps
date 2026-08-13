import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LoginPage } from "../pages/LoginPage";

const { loginMock } = vi.hoisted(() => ({ loginMock: vi.fn() }));

vi.mock("../hooks/useAuth", () => ({
  useAuth: () => ({ login: loginMock }),
}));

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/login"]}>
      <LoginPage />
    </MemoryRouter>,
  );

describe("LoginPage", () => {
  beforeEach(() => {
    loginMock.mockReset();
  });

  it("validates before submitting", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(screen.getByRole("button", { name: "Masuk" }));

    expect(await screen.findByText("Email wajib diisi")).toBeInTheDocument();
    expect(loginMock).not.toHaveBeenCalled();
  });

  it("submits one valid login command", async () => {
    loginMock.mockResolvedValue({ role: "karyawan" });
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Email kerja"), "user@example.test");
    await user.type(screen.getByLabelText("Password"), "valid-password");
    await user.click(screen.getByRole("button", { name: "Masuk" }));

    expect(loginMock).toHaveBeenCalledTimes(1);
    expect(loginMock).toHaveBeenCalledWith({
      email: "user@example.test",
      password: "valid-password",
    });
  });

  it("shows official locked-account recovery guidance", async () => {
    loginMock.mockRejectedValue({ code: "ACCOUNT_LOCKED", fields: {} });
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Email kerja"), "user@example.test");
    await user.type(screen.getByLabelText("Password"), "valid-password");
    await user.click(screen.getByRole("button", { name: "Masuk" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Akun terkunci. Pulihkan password untuk membuka akun.",
    );
    expect(screen.getByRole("link", { name: "Pulihkan password" })).toHaveAttribute(
      "href",
      "/reset-password",
    );
  });
});
