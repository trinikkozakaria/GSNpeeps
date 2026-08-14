import { Suspense, useEffect, useMemo, useRef, useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { useAuth } from "../../modules/auth/hooks/useAuth";
import { NotificationBell } from "../../modules/notifications/components/NotificationBell";
import { navigationForRole, roleLabel } from "../../routes/navigation/navigation";
import { Button } from "../ui/Button";
import { ProtectedImage } from "../media/ProtectedImage";

const navLinkClassName = ({ isActive }) =>
  `block rounded-lg px-3 py-2 text-sm font-bold focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700 ${
    isActive
      ? "bg-cyan-700 text-white"
      : "text-slate-600 hover:bg-slate-900/5 hover:text-slate-900"
  }`;

const iconPaths = {
  Beranda: ["M3 11l9-8 9 8", "M5 10v10h14V10", "M9 20v-6h6v6"],
  Pribadi: ["M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z", "M4 21a8 8 0 0 1 16 0"],
  Pengajuan: ["M6 3h9l3 3v15H6Z", "M14 3v4h4", "M9 12h6", "M9 16h6"],
  Persetujuan: ["M9 5h6", "M9 3v4", "M5 5h14v16H5Z", "m8 15 2 2 4-5"],
  Organisasi: ["M12 12a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z", "M5 21a7 7 0 0 1 14 0", "M4 9a2 2 0 1 0 0-4", "M20 9a2 2 0 1 1 0-4"],
  Monitoring: ["M4 20V10", "M10 20V4", "M16 20v-7", "M22 20H2"],
  Informasi: ["M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z", "M12 10v7", "M12 7h.01"],
  "Master Data": ["M4 6c0-2 16-2 16 0s-16 2-16 0Z", "M4 6v6c0 2 16 2 16 0V6", "M4 12v6c0 2 16 2 16 0v-6"],
  Administrasi: ["M12 3 4 5 7 6v5c0 4-3 7-7 7s-7-3-7-7V6Z", "m9 14 2 2 4-5"],
  Akun: ["M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z", "M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2 3.46-.08-.02a1.7 1.7 0 0 0-1.8.18l-.1.06a1.7 1.7 0 0 0-.88 1.62V22h-4v-.08a1.7 1.7 0 0 0-.88-1.62l-.1-.06a1.7 1.7 0 0 0-1.8-.18l-.08.02-2-3.46.06-.06A1.7 1.7 0 0 0 4.6 15v-.12a1.7 1.7 0 0 0-1-1.44L3.5 13.4v-4l.08-.04a1.7 1.7 0 0 0 1-1.44V7.8a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2-3.46.08.02a1.7 1.7 0 0 0 1.8-.18l.1-.06A1.7 1.7 0 0 0 9.04.56V.5h4v.08a1.7 1.7 0 0 0 .88 1.62l.1.06a1.7 1.7 0 0 0 1.8.18l.08-.02 2 3.46-.06.06a1.7 1.7 0 0 0-.34 1.88v.12a1.7 1.7 0 0 0 1 1.44l.08.04v4l-.08.04a1.7 1.7 0 0 0-1 1.44Z"],
};

const NavigationIcon = ({ label }) => (
  <svg
    aria-hidden="true"
    viewBox="0 0 24 24"
    className="h-5 w-5"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    {(iconPaths[label] ?? iconPaths.Informasi).map((path) => <path key={path} d={path} />)}
  </svg>
);

const MenuIcon = () => (
  <svg
    aria-hidden="true"
    viewBox="0 0 24 24"
    className="h-5 w-5"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
  >
    <path d="M4 7h16M4 12h16M4 17h16" />
  </svg>
);

const compactItemClassName = (active) =>
  `flex h-12 w-12 items-center justify-center rounded-lg border text-xs font-bold focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700 ${
    active
      ? "border-cyan-700 bg-cyan-700 text-white"
      : "border-slate-900/15 bg-slate-900/5 text-slate-700 hover:bg-slate-900/10"
  }`;

export const AppShell = () => {
  const auth = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const sidebarRef = useRef(null);
  const collapsedFlyoutRef = useRef(null);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [collapsedGroup, setCollapsedGroup] = useState(null);
  const [collapsedFlyoutTop, setCollapsedFlyoutTop] = useState(80);
  const [openGroups, setOpenGroups] = useState({});
  const navigation = useMemo(() => navigationForRole(auth.role), [auth.role]);
  const collapsedGroupItem = navigation.find((item) => item.label === collapsedGroup);

  useEffect(() => {
    const activeGroup = navigation.find((item) =>
      item.children?.some((child) => location.pathname === child.path),
    );
    if (activeGroup) {
      setOpenGroups((current) => ({ ...current, [activeGroup.label]: true }));
    }
    setCollapsedGroup(null);
  }, [location.pathname, navigation]);

  useEffect(() => {
    if (!collapsedGroup) return undefined;
    const closeOnEscape = (event) => {
      if (event.key === "Escape") setCollapsedGroup(null);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [collapsedGroup]);

  useEffect(() => {
    if (sidebarCollapsed) return undefined;
    const closeExpandedSidebar = (event) => {
      const desktopViewport = window.matchMedia?.("(min-width: 1024px)");
      if (desktopViewport && !desktopViewport.matches) return;
      if (!sidebarRef.current?.contains(event.target)) {
        setSidebarCollapsed(true);
        setCollapsedGroup(null);
      }
    };
    document.addEventListener("pointerdown", closeExpandedSidebar);
    return () => document.removeEventListener("pointerdown", closeExpandedSidebar);
  }, [sidebarCollapsed]);

  useEffect(() => {
    if (!collapsedGroup) return undefined;
    const closeFlyout = (event) => {
      if (
        !sidebarRef.current?.contains(event.target) &&
        !collapsedFlyoutRef.current?.contains(event.target)
      ) {
        setCollapsedGroup(null);
      }
    };
    document.addEventListener("pointerdown", closeFlyout);
    return () => document.removeEventListener("pointerdown", closeFlyout);
  }, [collapsedGroup]);

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
                  <ProtectedImage
                    path={auth.user.foto_profil_url}
                    alt=""
                    className="h-full w-full object-cover"
                    showErrorMessage={false}
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
          ref={sidebarRef}
          aria-label="Navigasi utama"
          className={`sticky top-16 z-30 border-b border-slate-900/10 bg-white px-4 py-2 transition-[width] sm:px-6 lg:fixed lg:inset-y-0 lg:top-16 lg:left-0 lg:z-30 lg:overflow-y-auto lg:border-b-0 lg:border-r lg:px-4 lg:py-6 ${sidebarCollapsed ? "lg:w-20" : "lg:w-60"}`}
        >
          {sidebarCollapsed && (
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setSidebarCollapsed(false);
                setCollapsedGroup(null);
              }}
              aria-expanded="false"
              aria-label="Buka sidebar"
              className="mb-3 hidden h-12 min-h-0 w-12 px-0 py-0 lg:inline-flex"
            >
              <MenuIcon />
            </Button>
          )}
          <ul
            aria-label="Menu utama"
            className={`flex gap-2 overflow-x-auto pb-1 lg:space-y-1 lg:overflow-visible lg:pb-0 ${sidebarCollapsed ? "lg:hidden" : "lg:block"}`}
          >
            {navigation.map((item) => {
              const groupVisible = item.children ? Boolean(openGroups[item.label]) : false;
              return item.children ? (
                // Label pengelompokan UI-saja (mis. "Pengajuan"): tidak pernah menjadi link
                // sendiri, hanya menyorot hierarki anaknya. Disembunyikan di strip mobile
                // karena indentasi tidak berarti pada layout horizontal; anaknya tetap
                // tampil sebagai item biasa di sana.
                <li key={item.label} className="contents lg:block">
                  <Button type="button" variant="secondary" aria-expanded={groupVisible} onClick={() => setOpenGroups((current) => ({ ...current, [item.label]: !current[item.label] }))} className="hidden w-full justify-between lg:inline-flex">
                    <span>{item.label}</span><span aria-hidden="true">{openGroups[item.label] ? "▾" : "▸"}</span>
                  </Button>
                  <ul className={`contents lg:space-y-1 ${groupVisible ? "lg:block" : "lg:hidden"}`}>
                    {item.children.map((child) => (
                      <li key={child.path} className="shrink-0">
                        <NavLink to={child.path} className={navLinkClassName} title={child.label}>
                          <span className="lg:pl-3">{child.label}</span>
                        </NavLink>
                      </li>
                    ))}
                  </ul>
                </li>
              ) : (
                <li key={item.path} className="shrink-0">
                  <NavLink to={item.path} end={item.path === "/app"} className={navLinkClassName} title={item.label}>
                    <span>{item.label}</span>
                  </NavLink>
                </li>
              );
            })}
          </ul>
          {sidebarCollapsed && (
            <ul aria-label="Menu ringkas" className="hidden space-y-2 lg:block">
              {navigation.map((item) => {
                const active = item.children
                  ? item.children.some((child) => location.pathname === child.path)
                  : location.pathname === item.path;
                return (
                  <li key={item.label}>
                    {item.children ? (
                      <button
                        type="button"
                        aria-label={item.label}
                        aria-expanded={collapsedGroup === item.label}
                        title={item.label}
                        className={compactItemClassName(active)}
                        onClick={(event) => {
                          const buttonTop = event.currentTarget.getBoundingClientRect().top;
                          setCollapsedFlyoutTop(Math.max(72, Math.min(buttonTop, window.innerHeight - 260)));
                          setCollapsedGroup((current) => current === item.label ? null : item.label);
                        }}
                      >
                        <NavigationIcon label={item.label} />
                      </button>
                    ) : (
                      <NavLink
                        to={item.path}
                        end={item.path === "/app"}
                        aria-label={item.label}
                        title={item.label}
                        className={({ isActive }) => compactItemClassName(isActive)}
                      >
                        <NavigationIcon label={item.label} />
                      </NavLink>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </nav>
        {sidebarCollapsed && collapsedGroupItem && (
          <section
            ref={collapsedFlyoutRef}
            role="group"
            aria-label={`Submenu ${collapsedGroupItem.label}`}
            style={{ top: collapsedFlyoutTop }}
            className="fixed left-20 z-40 hidden w-64 rounded-xl border border-slate-900/15 bg-white p-3 shadow-xl lg:block"
          >
            <div className="mb-2 flex items-center justify-between gap-3 px-2">
              <p className="font-bold">{collapsedGroupItem.label}</p>
              <button
                type="button"
                aria-label={`Tutup submenu ${collapsedGroupItem.label}`}
                className="rounded-md px-2 py-1 text-slate-600 hover:bg-slate-900/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700"
                onClick={() => setCollapsedGroup(null)}
              >
                <span aria-hidden="true">×</span>
              </button>
            </div>
            <ul className="space-y-1">
              {collapsedGroupItem.children.map((child) => (
                <li key={child.path}>
                  <NavLink
                    to={child.path}
                    className={navLinkClassName}
                    onClick={() => setCollapsedGroup(null)}
                  >
                    {child.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </section>
        )}
        <main
          id="main-content"
          className={`min-w-0 px-4 py-6 transition-[margin] sm:px-6 lg:px-8 lg:py-8 ${sidebarCollapsed ? "lg:ml-20" : "lg:ml-60"}`}
        >
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
