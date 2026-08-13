import { Navigate, Outlet } from "react-router-dom";

import { SessionLoadingPage } from "../../components/feedback/SessionLoadingPage";
import { useAuth } from "../../modules/auth/hooks/useAuth";

export const PublicOnlyRoute = () => {
  const auth = useAuth();
  if (auth.status === "initializing") {
    return <SessionLoadingPage />;
  }
  if (auth.status === "authenticated") {
    return <Navigate to="/app" replace />;
  }
  return <Outlet />;
};

