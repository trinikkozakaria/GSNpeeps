import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  __testing,
  clearStoredSession,
  readStoredSession,
  writeStoredSession,
} from "./session-cookie";

const wipeCookies = () => {
  document.cookie
    .split(";")
    .map((part) => part.trim().split("=")[0])
    .filter(Boolean)
    .forEach((name) => {
      document.cookie = `${name}=; path=/; Max-Age=0`;
    });
};

describe("session-cookie", () => {
  beforeEach(() => {
    wipeCookies();
  });

  afterEach(() => {
    vi.useRealTimers();
    wipeCookies();
  });

  it("returns null when no session cookie exists", () => {
    expect(readStoredSession()).toBeNull();
  });

  it("persists a token so it survives a page reload", () => {
    const expiresAt = Date.now() + 28800 * 1000;
    writeStoredSession({ token: "jwt-token", expiresAt });

    expect(document.cookie).toContain(`${__testing.COOKIE_NAME}=`);
    expect(readStoredSession()).toEqual({ token: "jwt-token", expiresAt });
  });

  it("drops an expired session instead of returning it", () => {
    vi.useFakeTimers();
    const expiresAt = Date.now() + 60 * 1000;
    writeStoredSession({ token: "jwt-token", expiresAt });

    vi.advanceTimersByTime(61 * 1000);

    expect(readStoredSession()).toBeNull();
    expect(document.cookie).not.toContain("jwt-token");
  });

  it("refuses to store a token that is already expired", () => {
    writeStoredSession({ token: "jwt-token", expiresAt: Date.now() - 1000 });
    expect(readStoredSession()).toBeNull();
  });

  it("discards a corrupted cookie value", () => {
    document.cookie = `${__testing.COOKIE_NAME}=not-json; path=/`;
    expect(readStoredSession()).toBeNull();
    expect(document.cookie).not.toContain("not-json");
  });

  it("clears the stored session", () => {
    writeStoredSession({ token: "jwt-token", expiresAt: Date.now() + 60_000 });
    clearStoredSession();
    expect(readStoredSession()).toBeNull();
  });
});
