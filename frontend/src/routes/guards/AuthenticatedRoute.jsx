import { Navigate, Outlet, useLocation } from "react-router-dom";

import { SessionLoadingPage } from "../../components/feedback/SessionLoadingPage";
import { useAuth } from "../../modules/auth/hooks/useAuth";

export const AuthenticatedRoute = () => {
  const auth = useAuth();
  const location = useLocation();

  if (auth.status === "initializing") {
    return <SessionLoadingPage />;
  }
  if (auth.status !== "authenticated") {
    const returnTo = `${location.pathname}${location.search}`;
    return <Navigate to="/login" replace state={{ returnTo }} />;
  }
  return <Outlet />;
};

