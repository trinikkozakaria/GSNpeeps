import { createBrowserRouter, Navigate } from "react-router-dom";

import { ForbiddenPage } from "../components/feedback/ForbiddenPage";
import { NotFoundPage } from "../components/feedback/NotFoundPage";
import { AppShell } from "../components/layout/AppShell";
import { AccountSecurityPage } from "../modules/auth/pages/AccountSecurityPage";
import { LoginPage } from "../modules/auth/pages/LoginPage";
import { ModuleAccessPage } from "../modules/auth/pages/ModuleAccessPage";
import { ResetPasswordPage } from "../modules/auth/pages/ResetPasswordPage";
import { RoleLandingPage } from "../modules/auth/pages/RoleLandingPage";
import { DashboardPage } from "../modules/dashboard/pages/DashboardPage";
import { MyProfilePage } from "../modules/profile/pages/MyProfilePage";
import { PersonalMetricsPage } from "../modules/profile/pages/PersonalMetricsPage";
import { EmployeeDetailPage } from "../modules/employees/pages/EmployeeDetailPage";
import { EmployeeCreatePage } from "../modules/employees/pages/EmployeeCreatePage";
import { EmployeeEditPage } from "../modules/employees/pages/EmployeeEditPage";
import { EmployeeListPage } from "../modules/employees/pages/EmployeeListPage";
import { AuthenticatedRoute } from "./guards/AuthenticatedRoute";
import { PublicOnlyRoute } from "./guards/PublicOnlyRoute";
import { RoleRoute } from "./guards/RoleRoute";
import { roles } from "./navigation/navigation";

const personalRoles = [roles.employee, roles.supervisor, roles.hr];
const monitoringRoles = [roles.hr, roles.topManagement];
const approvalRoles = [roles.supervisor, roles.hr, roles.topManagement];

export const router = createBrowserRouter([
  {
    element: <PublicOnlyRoute />,
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/reset-password", element: <ResetPasswordPage /> },
    ],
  },
  {
    element: <AuthenticatedRoute />,
    children: [
      { path: "/", element: <Navigate to="/app" replace /> },
      {
        path: "/app",
        element: <AppShell />,
        children: [
          { index: true, element: <RoleLandingPage /> },
          {
            element: <RoleRoute allowedRoles={personalRoles} />,
            children: [
              { path: "profil", element: <MyProfilePage /> },
              { path: "metrik-personal", element: <PersonalMetricsPage /> },
              { path: "absensi", element: <ModuleAccessPage title="Kehadiran Saya" description="Check-in dan riwayat kehadiran milik pengguna aktif." /> },
              { path: "pengajuan", element: <ModuleAccessPage title="Pengajuan Saya" description="Ketidakhadiran dan lembur milik pengguna aktif." /> },
            ],
          },
          {
            element: <RoleRoute allowedRoles={approvalRoles} />,
            children: [
              { path: "persetujuan", element: <ModuleAccessPage title="Persetujuan" description="Scope persetujuan ditentukan relasi bawahan dan tahap aktif oleh backend." /> },
            ],
          },
          {
            element: <RoleRoute allowedRoles={monitoringRoles} />,
            children: [
              { path: "karyawan", element: <EmployeeListPage /> },
              { path: "karyawan/:id", element: <EmployeeDetailPage /> },
              {
                element: <RoleRoute allowedRoles={[roles.hr]} />,
                children: [
                  { path: "karyawan/baru", element: <EmployeeCreatePage /> },
                  { path: "karyawan/:id/edit", element: <EmployeeEditPage /> },
                ],
              },
              { path: "dashboard", element: <DashboardPage /> },
              { path: "laporan-kehadiran", element: <ModuleAccessPage title="Laporan Kehadiran" description="Laporan organisasi untuk HR dan Top Management." /> },
              { path: "akses", element: <ModuleAccessPage title="AKSES" description="Role dan permission; Top Management hanya membaca." /> },
              { path: "audit", element: <ModuleAccessPage title="Audit Log" description="Riwayat aktivitas terkontrol dan teredaksi." readOnly /> },
            ],
          },
          { path: "notifikasi", element: <ModuleAccessPage title="Notifikasi" description="Notifikasi pribadi pengguna aktif." /> },
          { path: "keamanan", element: <AccountSecurityPage /> },
          { path: "*", element: <NotFoundPage /> },
        ],
      },
    ],
  },
  { path: "/forbidden", element: <ForbiddenPage /> },
  { path: "*", element: <NotFoundPage /> },
]);
