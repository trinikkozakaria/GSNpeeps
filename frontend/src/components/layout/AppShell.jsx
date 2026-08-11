import { Suspense, useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { useAuth } from "../../modules/auth/hooks/useAuth";
import { NotificationBell } from "../../modules/notifications/components/NotificationBell";
import { navigationForRole, roleLabel } from "../../routes/navigation/navigation";
import { Button } from "../ui/Button";

export const AppShell = () => {
  const auth = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const navigation = navigationForRole(auth.role);

  // "Kehadiran Saya" turut tersorot aktif ketika sub-alur Ajukan Ketidakhadiran/Lembur
  // sedang dibuka, karena keduanya adalah bagian dari alur kehadiran yang sama.
  const isItemActive = (item) => 
    (!item.activeMatchPaths?.some((path) => location.pathname === path));

  const handleLogout = async () => {
    if (isLoggingOut) {
      return;
    }
    setIsLoggingOut(true);
    try {
      await auth.logout();
    } finally {
      navigate("/login", { replace: true });
    }
  };

  return (
    <div className="min-h-screen bg-white text-slate-900">
      <a
        href="#main-content"
        className="sr-only z-50 rounded-md bg-slate-900 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
      >
        Lewati ke konten utama
      </a>
      <header className="fixed inset-x-0 top-0 z-40 h-16 border-b border-slate-900/10 bg-white">
        <div className="mx-auto flex h-full max-w-[100rem] items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
          <div className="flex min-w-0 items-baseline gap-3">
            <p className="truncate text-lg font-bold tracking-tight">GSNpeeps</p>
            <p className="hidden text-xs font-semibold uppercase tracking-[0.25em] text-cyan-700 sm:block">
              HR Information System
            </p>
          </div>
          <div className="flex items-center gap-3 sm:gap-4">
            <NotificationBell />
            <div className="hidden text-right sm:block">
              <p className="text-sm font-semibold">{auth.user.nama}</p>
              <p className="text-xs text-slate-500">{roleLabel[auth.role]}</p>
            </div>
            <Button variant="secondary" onClick={handleLogout} disabled={isLoggingOut}>
              {isLoggingOut ? "Keluar…" : "Keluar"}
            </Button>
          </div>
        </div>
      </header>
      <div className="pt-16">
        <nav
          aria-label="Navigasi utama"
          className="sticky top-16 z-30 border-b border-slate-900/10 bg-white px-4 py-2 sm:px-6 lg:fixed lg:inset-y-0 lg:top-16 lg:left-0 lg:z-30 lg:w-60 lg:overflow-y-auto lg:border-b-0 lg:border-r lg:px-4 lg:py-6"
        >
          <ul className="flex gap-2 overflow-x-auto pb-1 lg:block lg:space-y-1 lg:overflow-visible lg:pb-0">
            {navigation.map((item) => (
              <li key={item.path} className="shrink-0">
                <NavLink
                  to={item.path}
                  end={item.path === "/app"}
                  className={({ isActive }) =>
                    `block rounded-lg px-3 py-2 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700 ${
                      isActive && isItemActive(item)
                        ? "bg-cyan-700 text-white"
                        : "text-slate-600 hover:bg-slate-900/5 hover:text-slate-900"
                    }`
                  }
                >
                  {item.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <main id="main-content" className="min-w-0 px-4 py-6 sm:px-6 lg:ml-60 lg:px-8 lg:py-8">
          {/* Route dengan chunk terpisah memakai fallback yang sama dengan state memuat
              halaman lain sehingga perpindahan tidak terasa berbeda. */}
          <Suspense
            fallback={
              <p role="status" className="text-slate-600">
                Memuat halaman…
              </p>
            }
          >
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
};
