"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  ClipboardList,
  FileText,
  LayoutDashboard,
  LogOut,
  Menu,
  QrCode,
  UserRoundCog,
  X,
} from "lucide-react";
import api from "@/lib/api";
import { clearSession, type AuthUser } from "@/lib/auth";
import { BrandLogo } from "@/components/brand-logo";

const DashboardUserContext = createContext<AuthUser | null>(null);

export function useDashboardUser(): AuthUser {
  const user = useContext(DashboardUserContext);
  if (!user) throw new Error("useDashboardUser must be used within DashboardShell");
  return user;
}

const roleLabels: Record<AuthUser["role"], string> = {
  admin: "Admin",
  staff: "Staff",
  reviewer: "Reviewer",
};

const navigation = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard, adminOnly: false },
  { href: "/requests", label: "Permohonan PA", icon: FileText, adminOnly: false },
  { href: "/checklists", label: "Kelola checklist", icon: ClipboardList, adminOnly: true },
  { href: "/users", label: "Kelola user", icon: UserRoundCog, adminOnly: true },
  { href: "/whatsapp", label: "WhatsApp Staff", icon: QrCode, adminOnly: false, staffOnly: true },
] as const;

export function DashboardShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const isWhatsAppPage = pathname === "/whatsapp" || pathname.startsWith("/whatsapp/");
  const sidebarRef = useRef<HTMLElement>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const firstNavigationRef = useRef<HTMLAnchorElement>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [checkingSession, setCheckingSession] = useState(true);
  const [menuOpen, setMenuOpen] = useState(false);
  const [desktopView, setDesktopView] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    const mediaQuery = window.matchMedia("(min-width: 1024px)");
    const updateDesktopView = () => setDesktopView(mediaQuery.matches);
    updateDesktopView();
    mediaQuery.addEventListener("change", updateDesktopView);
    return () => mediaQuery.removeEventListener("change", updateDesktopView);
  }, []);

  useEffect(() => {
    let active = true;

    async function verifySession() {
      const token = localStorage.getItem("paa_token");
      if (!token) {
        router.replace("/login");
        return;
      }

      const expiresAt = localStorage.getItem("paa_expires_at");
      if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
        clearSession();
        router.replace("/login");
        return;
      }

      try {
        const response = await api.get<{ user: AuthUser }>("/auth/me");
        if (!active) return;
        setUser(response.data.user);
        localStorage.setItem("paa_user", JSON.stringify(response.data.user));
      } catch {
        if (!active) return;
        clearSession();
        router.replace("/login");
      } finally {
        if (active) setCheckingSession(false);
      }
    }

    void verifySession();
    return () => {
      active = false;
    };
  }, [router]);

  useEffect(() => {
    if (!checkingSession && user && isWhatsAppPage && user.role !== "staff") {
      router.replace("/dashboard");
    }
  }, [checkingSession, isWhatsAppPage, router, user]);

  useEffect(() => {
    if (!menuOpen) return;

    firstNavigationRef.current?.focus();
    function handleMenuKeys(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setMenuOpen(false);
        menuButtonRef.current?.focus();
        return;
      }
      if (event.key === "Tab") {
        const controls = sidebarRef.current?.querySelectorAll<HTMLElement>(
          'a[href], button:not(:disabled)',
        );
        if (!controls?.length) return;
        const first = controls[0];
        const last = controls[controls.length - 1];
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first.focus();
        }
      }
    }
    window.addEventListener("keydown", handleMenuKeys);
    return () => window.removeEventListener("keydown", handleMenuKeys);
  }, [menuOpen]);

  const handleLogout = useCallback(async () => {
    setLoggingOut(true);
    try {
      await api.post("/auth/logout");
    } catch {
      // Clear the local session even if the API request fails.
    } finally {
      clearSession();
      router.replace("/login");
    }
  }, [router]);

  if (checkingSession || !user || (isWhatsAppPage && user.role !== "staff")) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-[#f5f7f6]">
        <div role="status" className="text-center">
          {checkingSession || !user ? (
            <>
              <div className="mx-auto h-9 w-9 animate-spin rounded-full border-4 border-slate-200 border-t-blue-600" />
              <p className="mt-4 text-sm text-slate-700">Memverifikasi sesi...</p>
            </>
          ) : (
            <p className="text-sm text-slate-700">Halaman WhatsApp hanya tersedia untuk akun Staff. Mengalihkan ke dashboard...</p>
          )}
        </div>
      </main>
    );
  }

  const firstName = user.full_name.trim().split(/\s+/)[0] || "Pengguna";

  return (
    <DashboardUserContext.Provider value={user}>
      <div className="min-h-screen bg-[#f5f7f6] text-slate-900">
        {menuOpen && (
          <button
            type="button"
            aria-label="Tutup menu navigasi"
            onClick={() => {
              setMenuOpen(false);
              menuButtonRef.current?.focus();
            }}
            className="fixed inset-0 z-40 bg-slate-950/40 lg:hidden"
          />
        )}

        <aside
          ref={sidebarRef}
          id="workspace-navigation"
          role={!desktopView && menuOpen ? "dialog" : undefined}
          aria-modal={!desktopView && menuOpen}
          aria-label="Navigasi workspace"
          aria-hidden={!desktopView && !menuOpen}
          inert={!desktopView && !menuOpen}
          className={`fixed inset-y-0 left-0 z-50 flex w-68 flex-col border-r border-[#dce7e3] bg-white transition-transform duration-200 motion-reduce:transition-none lg:translate-x-0 ${
            menuOpen ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <div className="flex min-h-20 items-center justify-between gap-3 border-b border-[#e7eeeb] px-5">
            <div className="flex min-w-0 items-center gap-3">
              <BrandLogo alt="" width={44} height={37} priority className="shrink-0 rounded-lg bg-white p-1 object-contain" />
              <div className="min-w-0">
                <p className="truncate font-semibold tracking-tight text-[#183336]">Ayuprama</p>
                <p className="text-xs text-slate-600">Ruang kerja PA dan klaim</p>
              </div>
            </div>
            <button
              type="button"
              aria-label="Tutup menu"
              onClick={() => {
                setMenuOpen(false);
                menuButtonRef.current?.focus();
              }}
              className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-lg text-slate-700 hover:bg-slate-100 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-blue-600 lg:hidden"
            >
              <X size={19} aria-hidden="true" />
            </button>
          </div>

          <nav aria-label="Navigasi utama" className="flex-1 space-y-1 px-3 py-6">
            <p className="px-3 pb-3 text-xs font-semibold uppercase tracking-wide text-slate-600">
              Workspace
            </p>
            {navigation
              .filter((item) =>
                (!item.adminOnly || user.role === "admin") &&
                (!("staffOnly" in item) || user.role === "staff"),
              )
              .map((item, index) => {
                const active =
                  pathname === item.href ||
                  (item.href === "/requests" && pathname.startsWith("/requests/"));
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    ref={index === 0 ? firstNavigationRef : undefined}
                    href={item.href}
                    onClick={() => setMenuOpen(false)}
                    aria-current={active ? "page" : undefined}
                    className={`flex min-h-11 items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-blue-600 ${
                      active
                        ? "bg-[#e8f1ef] font-semibold text-blue-700"
                        : "text-slate-700 hover:bg-slate-50 hover:text-blue-700"
                    }`}
                  >
                    <Icon size={19} aria-hidden="true" />
                    {item.label}
                  </Link>
                );
              })}
          </nav>

          <div className="border-t border-[#e7eeeb] p-4">
            <div className="mb-3 flex items-center gap-3 rounded-lg bg-[#f5f7f6] p-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#dcebe8] font-semibold text-blue-700">
                {firstName.charAt(0).toUpperCase()}
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">{user.full_name}</p>
                <p className="truncate text-xs text-slate-600">{roleLabels[user.role]}</p>
              </div>
            </div>
            <button
              type="button"
              onClick={handleLogout}
              disabled={loggingOut}
              className="flex min-h-11 w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-slate-700 transition hover:bg-red-50 hover:text-red-800 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-red-800 disabled:opacity-60"
            >
              <LogOut size={18} aria-hidden="true" />
              {loggingOut ? "Keluar..." : "Keluar"}
            </button>
          </div>
        </aside>

        <div className="lg:pl-68">
          <div className="sticky top-0 z-30 flex min-h-16 items-center gap-3 border-b border-[#dce7e3] bg-white px-4 sm:px-6 lg:hidden">
            <button
              ref={menuButtonRef}
              type="button"
              aria-label="Buka menu navigasi"
              aria-expanded={menuOpen}
              aria-controls="workspace-navigation"
              onClick={() => setMenuOpen(true)}
              className="inline-flex min-h-11 min-w-11 items-center justify-center gap-2 rounded-lg text-sm font-medium text-blue-700 hover:bg-blue-50 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-blue-600"
            >
              <Menu size={20} aria-hidden="true" />
              Menu
            </button>
            <BrandLogo alt="" width={34} height={29} priority className="shrink-0 rounded-md bg-white p-0.5 object-contain" />
            <span className="text-sm font-semibold text-[#183336]">Ayuprama</span>
          </div>
          <div>{children}</div>
        </div>
      </div>
    </DashboardUserContext.Provider>
  );
}
