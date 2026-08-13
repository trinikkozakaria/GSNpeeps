import { useContext } from "react";

import { AuthContext } from "../context/AuthContext";

export const useAuth = () => {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth harus digunakan di dalam AuthProvider");
  }
  return value;
};

