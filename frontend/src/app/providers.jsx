import { QueryClientProvider } from "@tanstack/react-query";

import { queryClient } from "../lib/query/query-client";
import { AuthProvider } from "../modules/auth/context/AuthContext";

export const AppProviders = ({ children }) => (
  <QueryClientProvider client={queryClient}>
    <AuthProvider>{children}</AuthProvider>
  </QueryClientProvider>
);
