"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { FileText, RefreshCw } from "lucide-react";
import api from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard-shell";

const reportStatuses = [
  "ready_for_review",
  "in_review",
  "needs_information",
  "ready_for_submission",
  "submitted",
  "completed",
] as const;

type RequestStatus = (typeof reportStatuses)[number];

type DashboardRequest = {
  id: number;
  request_number: string;
  payer_name: string;
  service_type: string;
  status: string;
  created_at: string;
};

type RequestListResponse = {
  data: DashboardRequest[];
  pagination: { total: number };
};

const statusLabels: Record<string, string> = {
  draft: "Draft",
  ready_for_review: "Siap direview",
  in_review: "Dalam review",
  needs_information: "Perlu informasi",
  ready_for_submission: "Siap diajukan",
  submitted: "Menunggu keputusan payer",
  completed: "Selesai",
  cancelled: "Dibatalkan",
  processing: "Processing",
  error: "Error",
};

const emptyStatusCounts: Record<RequestStatus, number> = {
  ready_for_review: 0,
  in_review: 0,
  needs_information: 0,
  ready_for_submission: 0,
  submitted: 0,
  completed: 0,
};

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Waktu tidak tersedia";

  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function statusStyle(status: string): string {
  if (status === "needs_information") {
    return "border-amber-300 bg-amber-50 text-amber-900";
  }
  if (status === "completed") {
    return "border-teal-200 bg-teal-50 text-teal-900";
  }
  if (status === "cancelled" || status === "error") {
    return "border-slate-300 bg-slate-100 text-slate-700";
  }
  return "border-slate-200 bg-white text-slate-700";
}

export default function DashboardPage() {
  const user = useDashboardUser();
  const [reportLoading, setReportLoading] = useState(false);
  const [reportError, setReportError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [totalRequests, setTotalRequests] = useState(0);
  const [statusCounts, setStatusCounts] = useState({ ...emptyStatusCounts });
  const [recentRequests, setRecentRequests] = useState<DashboardRequest[]>([]);

  useEffect(() => {
    let active = true;

    async function loadReport() {
      setReportLoading(true);
      setReportError("");

      try {
        const [recentResponse, ...statusResponses] = await Promise.all([
          api.get<RequestListResponse>("/pa-requests", {
            params: { page: 1, limit: 5 },
          }),
          ...reportStatuses.map((status) =>
            api.get<RequestListResponse>("/pa-requests", {
              params: { page: 1, limit: 1, status },
            }),
          ),
        ]);

        if (!active) return;

        const counts = { ...emptyStatusCounts };
        reportStatuses.forEach((status, index) => {
          counts[status] = statusResponses[index].data.pagination.total;
        });
        setRecentRequests(recentResponse.data.data);
        setTotalRequests(recentResponse.data.pagination.total);
        setStatusCounts(counts);
      } catch {
        if (active) setReportError("Ringkasan belum dapat dimuat.");
      } finally {
        if (active) setReportLoading(false);
      }
    }

    void loadReport();
    return () => {
      active = false;
    };
  }, [user, refreshKey]);

  const firstName = user.full_name.trim().split(/\s+/)[0] || "Pengguna";

  const summaryMetrics = [
    {
      label: "Proses review",
      value: statusCounts.ready_for_review + statusCounts.in_review,
      detail: "Siap dan sedang diperiksa",
    },
    {
      label: "Perlu tindak lanjut",
      value: statusCounts.needs_information,
      detail: "Perlu dilengkapi oleh Staff",
    },
    {
      label: "Review selesai",
      value: statusCounts.completed,
      detail: "Permohonan dengan review selesai",
    },
  ];

  const shortcuts = [
    {
      href: "/requests",
      label: user.role === "reviewer" ? "Buka antrian review" : "Buka permohonan PA",
      icon: FileText,
      primary: true,
    },
  ];
  return (
    <main className="min-h-screen bg-[#f5f7f6] text-slate-900">
      <div className="mx-auto max-w-6xl space-y-6 px-5 py-7 sm:px-8 sm:py-9">
          <section className="flex flex-col gap-4 border-b border-slate-200 pb-6 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-sm font-medium text-[#205b55]">Ringkasan sistem</p>
              <h1 className="mt-1 text-2xl font-semibold tracking-tight text-slate-950 sm:text-3xl">
                Selamat datang, {firstName}
              </h1>
              <p className="mt-2 text-sm text-slate-700">
                Ringkasan prior authorization dan kasus terbaru.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {shortcuts.map((shortcut) => {
                const Icon = shortcut.icon;
                return (
                  <Link
                    key={shortcut.label}
                    href={shortcut.href}
                    className={`inline-flex min-h-11 items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 ${
                      shortcut.primary
                        ? "bg-[#205b55] text-white hover:bg-[#174742]"
                        : "border border-slate-300 bg-white text-slate-800 hover:bg-slate-100"
                    }`}
                  >
                    <Icon size={17} aria-hidden="true" />
                    {shortcut.label}
                  </Link>
                );
              })}
              <button
                type="button"
                onClick={() => setRefreshKey((current) => current + 1)}
                disabled={reportLoading}
                aria-label="Muat ulang ringkasan"
                title="Muat ulang ringkasan"
                className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-lg border border-slate-300 bg-white text-slate-700 transition hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:cursor-wait disabled:opacity-60"
              >
                <RefreshCw
                  size={17}
                  className={reportLoading ? "animate-spin" : ""}
                  aria-hidden="true"
                />
              </button>
            </div>
          </section>

          {reportError && (
            <section
              role="alert"
              className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-400 bg-amber-50 px-4 py-3 text-sm text-amber-950"
            >
              <p>{reportError} Periksa koneksi backend atau muat ulang ringkasan.</p>
              <button
                type="button"
                onClick={() => setRefreshKey((current) => current + 1)}
                className="min-h-11 rounded-md border border-amber-500 px-3 font-semibold hover:bg-amber-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-800"
              >
                Coba lagi
              </button>
            </section>
          )}

          <section
            aria-label="Ringkasan review berkas"
            aria-busy={reportLoading}
            className="overflow-hidden rounded-xl border border-slate-200 bg-white"
          >
            <div className="grid divide-y divide-slate-200 sm:grid-cols-2 sm:divide-x sm:divide-y-0 xl:grid-cols-4">
              <div className="px-5 py-5 sm:px-6 sm:py-6">
                <p className="text-sm font-medium text-slate-600">Total kasus</p>
                <p className="mt-2 text-3xl font-semibold tabular-nums tracking-tight text-slate-950" aria-live="polite">
                  {reportLoading && totalRequests === 0 ? "—" : totalRequests}
                </p>
                <p className="mt-1 text-xs text-slate-500">Sesuai cakupan akun</p>
              </div>
              {summaryMetrics.map((metric) => (
                <div key={metric.label} className="px-5 py-5 sm:px-6 sm:py-6">
                  <p className="text-sm font-medium text-slate-600">{metric.label}</p>
                  <p className="mt-2 text-3xl font-semibold tabular-nums tracking-tight text-slate-950" aria-live="polite">
                    {reportLoading && totalRequests === 0 ? "—" : metric.value}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">{metric.detail}</p>
                </div>
              ))}
            </div>
          </section>

          <section>
            <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
              <div>
                <h3 className="text-lg font-semibold text-slate-950">Kasus terbaru</h3>
                <p className="mt-1 text-sm text-slate-600">Lima kasus terakhir yang dapat Anda akses.</p>
              </div>

            </div>

            <div className="overflow-hidden rounded-xl border border-slate-200 bg-white">
              {reportLoading && recentRequests.length === 0 ? (
                <p role="status" className="p-5 text-sm text-slate-700">Memuat permohonan...</p>
              ) : recentRequests.length === 0 ? (
                <div className="px-5 py-9 text-center">
                          <p className="font-medium text-slate-900">Belum ada kasus.</p>
                          <p className="mt-1 text-sm text-slate-600">Permohonan yang dibuat akan muncul di sini.</p>
                </div>
              ) : (
                <ul className="divide-y divide-slate-200" aria-live="polite">
                  {recentRequests.map((request) => (
                    <li key={request.id}>
                      <Link
                        href={`/requests/${request.id}`}
                        className="flex min-h-[72px] flex-col justify-center gap-2 px-4 py-3 transition hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-teal-700 sm:flex-row sm:items-center sm:justify-between sm:px-5"
                      >
                        <div className="min-w-0 sm:flex-1">
                          <p className="truncate text-sm font-semibold text-slate-950">
                            {request.request_number}
                          </p>
                          <p className="mt-1 truncate text-sm text-slate-600">
                            {request.payer_name} · {request.service_type}
                          </p>
                        </div>
                        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 sm:justify-end">
                          <span className={`inline-flex min-h-7 items-center rounded-md border px-2.5 py-1 text-xs font-medium ${statusStyle(request.status)}`}>
                            {statusLabels[request.status] ?? request.status}
                          </span>
                          <span className="text-xs text-slate-600 sm:min-w-36 sm:text-right">
                            {formatDate(request.created_at)}
                          </span>
                        </div>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </section>
        </div>
    </main>
  );
}
