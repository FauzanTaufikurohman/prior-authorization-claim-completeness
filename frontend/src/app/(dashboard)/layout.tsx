import { Suspense } from "react";
import type { ReactNode } from "react";
import { DashboardShell } from "@/components/dashboard-shell";

export default function DashboardLayout({ children }: { children: ReactNode }) {
  return (
    <Suspense
      fallback={
        <main className="flex min-h-screen items-center justify-center bg-[#f5f7f6]">
          <p role="status" className="text-sm text-slate-700">Memuat ruang kerja...</p>
        </main>
      }
    >
      <DashboardShell>{children}</DashboardShell>
    </Suspense>
  );
}
