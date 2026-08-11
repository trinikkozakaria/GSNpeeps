import { Suspense, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";

import { useAuth } from "../../modules/auth/hooks/useAuth";
import { NotificationBell } from "../../modules/notifications/components/NotificationBell";
import { navigationForRole, roleLabel } from "../../routes/navigation/navigation";
import { Button } from "../ui/Button";

const navLinkClassName = ({ isActive }) =>
  `block rounded-lg px-3 py-2 text-sm font-bold focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700 ${
    isActive
      ? "bg-cyan-700 text-white"
      : "text-slate-600 hover:bg-slate-900/5 hover:text-slate-900"
  }`;

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
    <div className="min-h-screen bg-white text-slate-900">
      <a
        href="#main-content"
        className="sr-only z-50 rounded-md bg-slate-900 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
      >
        Lewati ke konten utama
      </a>
      <header className="fixed inset-x-0 top-0 z-40 h-16 border-b border-slate-900/10 bg-white">
        <div className="mx-auto flex h-full items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
          <div className="flex min-w-0 items-baseline gap-3">
            <p className="truncate text-lg font-bold tracking-tight">GSNpeeps</p>
            <p className="hidden text-xs font-semibold uppercase tracking-[0.25em] text-cyan-700 sm:block">
              HR Information System
            </p>
          </div>
          <div className="flex items-center gap-3 sm:gap-4">
            <NotificationBell />
            <div className="flex items-center gap-2.5">
              <span className="hidden text-right sm:block">
                <p className="text-sm font-semibold">{auth.user.nama}</p>
                <p className="text-xs text-slate-500">{roleLabel[auth.role]}</p>
              </span>
              <span className="block h-9 w-9 shrink-0 overflow-hidden rounded-full border border-slate-900/10 bg-slate-100">
                {auth.user.foto_profil_url ? (
                  <img
                    src={auth.user.foto_profil_url}
                    alt=""
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <span
                    aria-hidden="true"
                    className="flex h-full w-full items-center justify-center text-xs font-semibold uppercase text-slate-500"
                  >
                    {auth.user.nama?.[0] ?? "?"}
                  </span>
                )}
              </span>
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
            {navigation.map((item) =>
              item.children ? (
                // Label pengelompokan UI-saja (mis. "Pengajuan"): tidak pernah menjadi link
                // sendiri, hanya menyorot hierarki anaknya. Disembunyikan di strip mobile
                // karena indentasi tidak berarti pada layout horizontal; anaknya tetap
                // tampil sebagai item biasa di sana.
                <li key={item.label} className="contents lg:block">
                  <p className="hidden px-3 pt-3 text-xs font-semibold uppercase tracking-wider text-slate-500 lg:block">
                    {item.label}
                  </p>
                  <ul className="contents lg:block lg:space-y-1">
                    {item.children.map((child) => (
                      <li key={child.path} className="shrink-0">
                        <NavLink to={child.path} className={navLinkClassName}>
                          <span className="lg:pl-3">{child.label}</span>
                        </NavLink>
                      </li>
                    ))}
                  </ul>
                </li>
              ) : (
                <li key={item.path} className="shrink-0">
                  <NavLink to={item.path} end={item.path === "/app"} className={navLinkClassName}>
                    {item.label}
                  </NavLink>
                </li>
              ),
            )}
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
