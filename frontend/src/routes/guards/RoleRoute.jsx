import { Navigate, Outlet } from "react-router-dom";

import { canAccessRoles } from "../navigation/navigation";
import { useAuth } from "../../modules/auth/hooks/useAuth";

export const RoleRoute = ({ allowedRoles }) => {
  const { role } = useAuth();
  if (!canAccessRoles(role, allowedRoles)) {
    return <Navigate to="/forbidden" replace />;
  }
  return <Outlet />;
};

