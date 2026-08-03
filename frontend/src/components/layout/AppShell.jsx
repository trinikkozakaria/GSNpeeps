import { Suspense, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";

import { useAuth } from "../../modules/auth/hooks/useAuth";
import { NotificationBell } from "../../modules/notifications/components/NotificationBell";
import { navigationForRole, roleLabel } from "../../routes/navigation/navigation";
import { Button } from "../ui/Button";

export const AppShell = () => {
  const auth = useAuth();
  const navigate = useNavigate();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const navigation = navigationForRole(auth.role);

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
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <a
        href="#main-content"
        className="sr-only z-50 rounded-md bg-white px-4 py-2 text-slate-950 focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
      >
        Lewati ke konten utama
      </a>
      <header className="border-b border-white/10 bg-slate-950">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-5 py-4 sm:px-8">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.25em] text-cyan-300">
              HR Information System
            </p>
            <p className="mt-1 text-xl font-bold tracking-tight">GSNpeeps</p>
          </div>
          <div className="flex items-center gap-4">
            <NotificationBell />
            <div className="text-right">
              <p className="text-sm font-semibold">{auth.user.nama}</p>
              <p className="text-xs text-slate-400">{roleLabel[auth.role]}</p>
            </div>
            <Button variant="secondary" onClick={handleLogout} disabled={isLoggingOut}>
              {isLoggingOut ? "Keluar…" : "Keluar"}
            </Button>
          </div>
        </div>
      </header>
      <div className="mx-auto grid max-w-7xl gap-8 px-5 py-6 sm:px-8 lg:grid-cols-[15rem_minmax(0,1fr)]">
        <nav aria-label="Navigasi utama">
          <ul className="flex gap-2 overflow-x-auto pb-2 lg:block lg:space-y-1 lg:overflow-visible">
            {navigation.map((item) => (
              <li key={item.path} className="shrink-0">
                <NavLink
                  to={item.path}
                  end={item.path === "/app"}
                  className={({ isActive }) =>
                    `block rounded-lg px-3 py-2 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-300 ${
                      isActive
                        ? "bg-cyan-300 text-slate-950"
                        : "text-slate-300 hover:bg-white/5 hover:text-white"
                    }`
                  }
                >
                  {item.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <main id="main-content" className="min-w-0">
          {/* Route dengan chunk terpisah memakai fallback yang sama dengan state memuat
              halaman lain sehingga perpindahan tidak terasa berbeda. */}
          <Suspense
            fallback={
              <p role="status" className="text-slate-300">
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
