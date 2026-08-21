import { Suspense, useEffect, useMemo, useRef, useState } from "react";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { useAuth } from "../../modules/auth/hooks/useAuth";
import { NotificationBell } from "../../modules/notifications/components/NotificationBell";
import {
  isNavItemActive,
  navigationForRole,
  navLinkTarget,
  roleLabel,
} from "../../routes/navigation/navigation";
import { ProtectedImage } from "../media/ProtectedImage";

const navLinkClassName = ({ isActive }) =>
  `block rounded-lg px-3 py-1.5 text-left text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700 ${
    isActive
      ? "font-bold text-cyan-700"
      : "font-normal text-slate-600 hover:bg-slate-900/5 hover:text-slate-900"
  }`;

const topLevelNavLinkClassName = ({ isActive }) =>
  `flex min-h-10 w-full items-center gap-2 rounded-lg border px-3 text-left text-sm font-semibold focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700 ${
    isActive
      ? "border-cyan-700 bg-cyan-700 text-white"
      : "border-slate-900/15 bg-slate-900/[0.03] text-slate-800 hover:bg-slate-900/[0.07]"
  }`;

const iconPaths = {
  Beranda: ["m3 11 9-8 9 8", "M5 10v11h14V10", "M9 21v-7h6v7"],
  Pribadi: ["M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z", "M4.5 21a7.5 7.5 0 0 1 15 0"],
  Pengajuan: ["M6 2h8l4 4v16H6Z", "M14 2v6h4", "M9 13h6", "M9 17h6"],
  Persetujuan: ["M9 5H6a2 2 0 0 0-2 2v14h16V7a2 2 0 0 0-2-2h-3", "M9 3h6v4H9Z", "m8 15 2 2 4-5"],
  Organisasi: ["M16 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2", "M10 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z", "M17 11a3 3 0 1 0 0-6", "M22 21v-2a4 4 0 0 0-3-3.87"],
  Monitoring: ["M3 3v18h18", "M8 17v-4", "M13 17V8", "M18 17V5"],
  Informasi: ["M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z", "M12 11v6", "M12 7h.01"],
  "Master Data": ["M4 5c0-2 16-2 16 0s-16 2-16 0Z", "M4 5v7c0 2 16 2 16 0V5", "M4 12v7c0 2 16 2 16 0v-7"],
  Administrasi: ["M5 3h9l4 4v10H5Z", "M14 3v5h4", "M8 11h6", "M8 14h4", "m13 18 6-6 2 2-6 6-3 1Z"],
  Akun: ["M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.09a2 2 0 0 1 1 1.73v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.09a2 2 0 0 1-1-1.74v-.51a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2Z", "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"],
};

const NavigationIcon = ({ label }) => (
  <svg
    aria-hidden="true"
    viewBox="0 0 24 24"
    className="h-6 w-6 shrink-0"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    {(iconPaths[label] ?? iconPaths.Informasi).map((path) => <path key={path} d={path} />)}
  </svg>
);

const MenuIcon = ({ className = "h-5 w-5" }) => (
  <svg
    aria-hidden="true"
    viewBox="0 0 24 24"
    className={className}
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
  >
    <path d="M4 7h16M4 12h16M4 17h16" />
  </svg>
);

const compactItemClassName = (active) =>
  `flex h-12 w-12 items-center justify-center rounded-lg border text-xs font-bold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-cyan-700 ${
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
  const mobileMenuButtonRef = useRef(null);
  const profileMenuRef = useRef(null);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [collapsedGroup, setCollapsedGroup] = useState(null);
  const [collapsedFlyoutTop, setCollapsedFlyoutTop] = useState(80);
  const [openGroups, setOpenGroups] = useState({});
  // Menu navbar mobile memakai render yang sama dengan compact icon list sidebar yang
  // diciutkan di desktop; hanya visibilitas dan trigger-nya berbeda per breakpoint.
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const navigation = useMemo(() => navigationForRole(auth.role), [auth.role]);
  const collapsedGroupItem = navigation.find((item) => item.label === collapsedGroup);
  const compactMenuVisible = sidebarCollapsed || mobileMenuOpen;

  useEffect(() => {
    const activeGroup = navigation.find((item) =>
      item.children?.some((child) => location.pathname === child.path),
    );
    if (activeGroup) {
      setOpenGroups((current) => ({ ...current, [activeGroup.label]: true }));
    }
    setCollapsedGroup(null);
    setMobileMenuOpen(false);
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

  useEffect(() => {
    if (!mobileMenuOpen) return undefined;
    const closeOnEscape = (event) => {
      if (event.key === "Escape") {
        setMobileMenuOpen(false);
        setCollapsedGroup(null);
      }
    };
    const closeOnOutsideClick = (event) => {
      if (
        !sidebarRef.current?.contains(event.target) &&
        !collapsedFlyoutRef.current?.contains(event.target) &&
        !mobileMenuButtonRef.current?.contains(event.target)
      ) {
        setMobileMenuOpen(false);
        setCollapsedGroup(null);
      }
    };
    window.addEventListener("keydown", closeOnEscape);
    document.addEventListener("pointerdown", closeOnOutsideClick);
    return () => {
      window.removeEventListener("keydown", closeOnEscape);
      document.removeEventListener("pointerdown", closeOnOutsideClick);
    };
  }, [mobileMenuOpen]);

  useEffect(() => {
    if (!profileMenuOpen) return undefined;
    const closeMenu = (event) => {
      if (event.key === "Escape" || !profileMenuRef.current?.contains(event.target)) {
        setProfileMenuOpen(false);
      }
    };
    window.addEventListener("keydown", closeMenu);
    document.addEventListener("pointerdown", closeMenu);
    return () => {
      window.removeEventListener("keydown", closeMenu);
      document.removeEventListener("pointerdown", closeMenu);
    };
  }, [profileMenuOpen]);

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
          <div className="flex min-w-0 items-center gap-3">
            <button
              ref={mobileMenuButtonRef}
              type="button"
              onClick={() => {
                setMobileMenuOpen((current) => !current);
                setCollapsedGroup(null);
              }}
              aria-expanded={mobileMenuOpen}
              aria-label={mobileMenuOpen ? "Tutup menu navigasi" : "Buka menu navigasi"}
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-slate-600 hover:bg-slate-900/5 hover:text-slate-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700 lg:hidden"
            >
              <MenuIcon />
            </button>
            <img src="/assets/logo.svg" alt="Logo GSN" className="h-10 w-10 shrink-0 object-contain" />
            <p className="truncate text-lg font-bold tracking-tight">GSNpeeps</p>
            <p className="hidden text-xs font-semibold uppercase tracking-[0.25em] text-cyan-700 sm:block">
              HR Information System
            </p>
          </div>
          <div className="flex items-center gap-2 sm:gap-3">
            <NotificationBell />
            <div ref={profileMenuRef} className="relative">
              <button
                type="button"
                aria-haspopup="menu"
                aria-expanded={profileMenuOpen}
                aria-label="Buka menu profil"
                onClick={() => setProfileMenuOpen((current) => !current)}
                className="flex min-h-12 max-w-72 items-center gap-2.5 rounded-xl border border-slate-900/15 bg-white px-2.5 py-1 text-left hover:bg-slate-900/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700"
              >
                <span className="block h-10 w-10 shrink-0 overflow-hidden rounded-full border border-slate-900/10 bg-slate-100">
                  {auth.user.foto_profil_url ? (
                    <ProtectedImage path={auth.user.foto_profil_url} alt="" className="h-full w-full object-cover" showErrorMessage={false} />
                  ) : (
                    <span aria-hidden="true" className="flex h-full w-full items-center justify-center text-xs font-semibold uppercase text-slate-500">
                      {auth.user.nama?.[0] ?? "?"}
                    </span>
                  )}
                </span>
                <span className="hidden min-w-0 max-w-48 text-left sm:block">
                  <span className="block truncate text-sm font-semibold text-slate-900">{auth.user.nama}</span>
                  <span className="block truncate text-xs text-slate-500">{roleLabel[auth.role]}</span>
                </span>
                <svg aria-hidden="true" viewBox="0 0 20 20" className={`h-4 w-4 shrink-0 text-slate-950 transition ${profileMenuOpen ? "rotate-180" : ""}`} fill="currentColor">
                  <path fillRule="evenodd" d="M5.22 7.22a.75.75 0 0 1 1.06 0L10 10.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0L5.22 8.28a.75.75 0 0 1 0-1.06Z" clipRule="evenodd" />
                </svg>
              </button>
              {profileMenuOpen && (
                <div role="menu" className="absolute right-0 mt-2 w-48 rounded-xl border border-slate-900/10 bg-white p-2 shadow-xl">
                  <Link role="menuitem" to="/app/keamanan" onClick={() => setProfileMenuOpen(false)} className="block rounded-lg px-3 py-2 text-sm text-slate-700 hover:bg-slate-900/5">
                    Pengaturan Akun
                  </Link>
                  <button role="menuitem" type="button" onClick={handleLogout} disabled={isLoggingOut} className="block w-full rounded-lg px-3 py-2 text-left text-sm text-rose-700 hover:bg-rose-50 disabled:opacity-50">
                    {isLoggingOut ? "Keluar…" : "Keluar"}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>
      <div className="pt-16">
        <nav
          ref={sidebarRef}
          aria-label="Navigasi utama"
          className={`sticky top-16 z-30 border-b border-slate-900/10 bg-white px-4 py-2 transition-[width] sm:px-6 lg:fixed lg:inset-y-0 lg:top-16 lg:left-0 lg:z-30 lg:overflow-y-auto lg:border-b-0 lg:border-r lg:px-4 lg:py-6 ${sidebarCollapsed ? "lg:w-20" : "lg:w-60"}`}
        >
          {/* Satu-satunya kontrol yang boleh mengubah collapsed/expanded sidebar; klik di
              luar sidebar sengaja tidak lagi menutupnya secara otomatis. */}
          <div className={`mb-3 hidden lg:flex ${sidebarCollapsed ? "justify-center" : "justify-end"}`}>
            <button
              type="button"
              onClick={() => {
                setSidebarCollapsed((current) => !current);
                setCollapsedGroup(null);
              }}
              aria-expanded={!sidebarCollapsed}
              aria-label={sidebarCollapsed ? "Buka sidebar" : "Ciutkan sidebar"}
              className="flex h-8 w-8 items-center justify-center rounded-md text-slate-600 hover:bg-slate-900/5 hover:text-slate-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700"
            >
              <MenuIcon />
            </button>
          </div>
          <ul
            aria-label="Menu utama"
            className={`hidden lg:space-y-1 lg:overflow-visible lg:pb-0 ${sidebarCollapsed ? "lg:hidden" : "lg:block"}`}
          >
            {navigation.map((item) => {
              const groupVisible = item.children ? Boolean(openGroups[item.label]) : false;
              const groupActive = item.children
                ? item.children.some((child) => isNavItemActive(child, location.pathname, location.search))
                : false;
              return item.children ? (
                // Label pengelompokan UI-saja (mis. "Pengajuan"): tidak pernah menjadi link
                // sendiri, hanya menyorot hierarki anaknya. Menu ini hanya untuk sidebar
                // desktop yang diperluas; navbar mobile memakai compact icon list yang sama
                // dengan sidebar yang diciutkan (lihat "Menu ringkas" di bawah).
                <li key={item.label} className="contents lg:block">
                  <button
                    type="button"
                    aria-expanded={groupVisible}
                    onClick={() => setOpenGroups((current) => ({ ...current, [item.label]: !current[item.label] }))}
                    className={`hidden min-h-10 w-full items-center justify-between rounded-lg border px-3 text-sm font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700 lg:inline-flex ${
                      groupActive
                        ? "border-cyan-700 bg-cyan-700 text-white"
                        : "border-slate-900/15 bg-slate-900/[0.03] text-slate-800 hover:bg-slate-900/[0.07]"
                    }`}
                  >
                    <span className="flex items-center gap-2"><NavigationIcon label={item.label} />{item.label}</span>
                    <span aria-hidden="true" className="flex h-6 w-6 shrink-0 items-center justify-center">
                      <svg viewBox="0 0 20 20" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d={openGroups[item.label] ? "m5 8 5 5 5-5" : "m8 5 5 5-5 5"} />
                      </svg>
                    </span>
                  </button>
                  <ul className={`contents lg:mt-1 lg:space-y-0.5 ${groupVisible ? "lg:block" : "lg:hidden"}`}>
                    {item.children.map((child) => (
                      <li key={`${child.path}-${child.label}`} className="shrink-0">
                        <NavLink
                          to={navLinkTarget(child)}
                          className={navLinkClassName({
                            isActive: isNavItemActive(child, location.pathname, location.search),
                          })}
                          title={child.label}
                          aria-label={child.ariaLabel}
                        >
                          <span className="lg:pl-3">{child.label}</span>
                        </NavLink>
                      </li>
                    ))}
                  </ul>
                </li>
              ) : (
                <li key={item.path} className="shrink-0">
                  <NavLink
                    to={navLinkTarget(item)}
                    className={topLevelNavLinkClassName({
                      isActive: isNavItemActive(item, location.pathname, location.search),
                    })}
                    title={item.label}
                  >
                    <NavigationIcon label={item.label} />
                    <span>{item.label}</span>
                  </NavLink>
                </li>
              );
            })}
          </ul>
          {compactMenuVisible && (
            <ul
              aria-label="Menu ringkas"
              className={`mt-2 flex flex-wrap gap-2 border-t border-slate-900/10 pt-3 lg:mt-0 lg:gap-0 lg:space-y-2 lg:border-t-0 lg:pt-0 ${sidebarCollapsed ? "lg:block" : "lg:hidden"}`}
            >
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
                        to={navLinkTarget(item)}
                        aria-label={item.label}
                        title={item.label}
                        className={compactItemClassName(
                          isNavItemActive(item, location.pathname, location.search),
                        )}
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
        {compactMenuVisible && collapsedGroupItem && (
          <section
            ref={collapsedFlyoutRef}
            role="group"
            aria-label={`Submenu ${collapsedGroupItem.label}`}
            style={{ top: collapsedFlyoutTop }}
            className="fixed left-4 z-40 w-64 max-w-[calc(100vw-2rem)] rounded-xl border border-slate-900/15 bg-white p-3 shadow-xl lg:left-20"
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
                <li key={`${child.path}-${child.label}`}>
                  <NavLink
                    to={navLinkTarget(child)}
                    className={navLinkClassName({
                      isActive: isNavItemActive(child, location.pathname, location.search),
                    })}
                    aria-label={child.ariaLabel}
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
