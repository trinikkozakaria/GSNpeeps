import { createContext, useCallback, useEffect, useMemo, useRef, useState } from "react";

import { configureAuthBoundary } from "../../../lib/api/client";
import { queryClient } from "../../../lib/query/query-client";
import {
  currentUserRequest,
  loginRequest,
  logoutRequest,
} from "../api/auth-api";

export const AuthContext = createContext(null);

const anonymousState = {
  status: "unauthenticated",
  user: null,
  role: null,
  expiresAt: null,
};

export const AuthProvider = ({ children }) => {
  const tokenRef = useRef(null);
  const [state, setState] = useState({ ...anonymousState, status: "initializing" });

  const clearSession = useCallback(async () => {
    tokenRef.current = null;
    await queryClient.cancelQueries();
    queryClient.clear();
    setState(anonymousState);
  }, []);

  useEffect(() => {
    configureAuthBoundary({
      getAccessToken: () => tokenRef.current,
      onUnauthorized: clearSession,
    });
    setState(anonymousState);
    return () => {
      configureAuthBoundary({});
    };
  }, [clearSession]);

  useEffect(() => {
    if (state.status !== "authenticated" || !state.expiresAt) {
      return undefined;
    }
    const remaining = state.expiresAt - Date.now();
    if (remaining <= 0) {
      void clearSession();
      return undefined;
    }
    const timer = window.setTimeout(() => void clearSession(), remaining);
    return () => window.clearTimeout(timer);
  }, [clearSession, state.expiresAt, state.status]);

  const login = useCallback(
    async (credentials, signal) => {
      const loginData = await loginRequest(credentials, signal);
      tokenRef.current = loginData.token;
      try {
        const user = await currentUserRequest(signal);
        setState({
          status: "authenticated",
          user,
          role: user.role,
          expiresAt: Date.now() + loginData.expires_in * 1000,
        });
        return user;
      } catch (error) {
        await clearSession();
        throw error;
      }
    },
    [clearSession],
  );

  const logout = useCallback(
    async (signal) => {
      try {
        if (tokenRef.current) {
          await logoutRequest(signal);
        }
      } finally {
        await clearSession();
      }
    },
    [clearSession],
  );

  const value = useMemo(
    () => ({
      ...state,
      login,
      logout,
      clearSession,
    }),
    [clearSession, login, logout, state],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

