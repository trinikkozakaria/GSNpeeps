import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AuthProvider } from "../context/AuthContext";
import { useAuth } from "../hooks/useAuth";
import { readStoredSession, writeStoredSession } from "../utils/session-cookie";

const { currentUserMock, loginMock, logoutMock } = vi.hoisted(() => ({
  currentUserMock: vi.fn(),
  loginMock: vi.fn(),
  logoutMock: vi.fn(),
}));

vi.mock("../api/auth-api", () => ({
  currentUserRequest: currentUserMock,
  loginRequest: loginMock,
  logoutRequest: logoutMock,
}));

const user = {
  id: "11111111-1111-4111-8111-111111111111",
  employee_id: "22222222-2222-4222-8222-222222222222",
  nama: "Rina",
  email: "rina@example.test",
  role: "hr",
};

const Probe = () => {
  const auth = useAuth();
  return <span data-testid="status">{`${auth.status}:${auth.role ?? "-"}`}</span>;
};

const renderProvider = () =>
  render(
    <AuthProvider>
      <Probe />
    </AuthProvider>,
  );

const wipeCookies = () => {
  document.cookie
    .split(";")
    .map((part) => part.trim().split("=")[0])
    .filter(Boolean)
    .forEach((name) => {
      document.cookie = `${name}=; path=/; Max-Age=0`;
    });
};

describe("AuthProvider session restore", () => {
  beforeEach(() => {
    wipeCookies();
    currentUserMock.mockReset();
    loginMock.mockReset();
    logoutMock.mockReset();
  });

  afterEach(() => {
    wipeCookies();
  });

  it("stays unauthenticated when no session cookie exists", async () => {
    renderProvider();

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent("unauthenticated:-"),
    );
    expect(currentUserMock).not.toHaveBeenCalled();
  });

  it("restores an authenticated session from the cookie after reload", async () => {
    writeStoredSession({ token: "jwt-token", expiresAt: Date.now() + 28800 * 1000 });
    currentUserMock.mockResolvedValue(user);

    renderProvider();

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent("authenticated:hr"),
    );
    expect(currentUserMock).toHaveBeenCalled();
  });

  it("drops the cookie when the stored token is rejected", async () => {
    writeStoredSession({ token: "revoked-token", expiresAt: Date.now() + 28800 * 1000 });
    currentUserMock.mockRejectedValue({ status: 401, code: "UNAUTHORIZED" });

    renderProvider();

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent("unauthenticated:-"),
    );
    expect(readStoredSession()).toBeNull();
  });

  it("stores the token on login", async () => {
    loginMock.mockResolvedValue({
      token: "fresh-token",
      token_type: "Bearer",
      expires_in: 28800,
      user,
    });
    currentUserMock.mockResolvedValue(user);

    const LoginProbe = () => {
      const auth = useAuth();
      return (
        <button
          type="button"
          onClick={() => void auth.login({ email: user.email, password: "secret-password" })}
        >
          masuk
        </button>
      );
    };

    render(
      <AuthProvider>
        <LoginProbe />
        <Probe />
      </AuthProvider>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent("unauthenticated:-"),
    );
    screen.getByRole("button", { name: "masuk" }).click();

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent("authenticated:hr"),
    );
    expect(readStoredSession()?.token).toBe("fresh-token");
  });
});
