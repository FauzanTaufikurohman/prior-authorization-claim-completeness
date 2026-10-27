
"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import {
  AlertCircle,
  ArrowLeft,
  ArrowRight,
  Building2,
  ClipboardList,
  FilePlus2,
  LoaderCircle,
  Plus,
  RefreshCw,
  Search,
  X,
} from "lucide-react";
import api from "@/lib/api";
import { getStoredUser, type AuthUser } from "@/lib/auth";

type RequestStatus =
  | "draft"
  | "ready_for_review"
  | "in_review"
  | "needs_information"
  | "ready_for_submission"
  | "submitted"
  | "completed"
  | "cancelled"
  | "processing"
  | "error";

interface PARequest {
  id: number;
  request_number: string;
  created_by: number;
  payer_name: string;
  service_type: string;
  provider_npi: string;
  patient_reference: string;
  diagnosis: string;
  status: RequestStatus;
  checklist_version_id: number | null;
  created_at: string;
  updated_at: string;
}

interface ListResponse {
  data: PARequest[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

interface CreateResponse {
  data: PARequest;
}

interface ActiveChecklistOption {
  id: number;
  name: string;
  payer_name: string;
  service_type: string;
  version_number: number;
}

interface ProviderLookup {
  number: string;
  name: string;
  practice_address?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  registry_notice?: string;
}

interface RequestForm {
  payer_name: string;
  service_type: string;
  provider_npi: string;
  patient_reference: string;
  diagnosis: string;
}

const statuses: { value: RequestStatus | ""; label: string }[] = [
  { value: "", label: "Semua status" },
  { value: "draft", label: "Draft" },
  { value: "ready_for_review", label: "Siap ditinjau" },
  { value: "in_review", label: "Sedang ditinjau" },
  { value: "needs_information", label: "Perlu dilengkapi" },
  { value: "ready_for_submission", label: "Siap diajukan" },
  { value: "submitted", label: "Diajukan" },
  { value: "completed", label: "Selesai" },
  { value: "cancelled", label: "Dibatalkan" },
  { value: "processing", label: "Diproses" },
  { value: "error", label: "Error" },
];

const initialForm: RequestForm = {
  payer_name: "",
  service_type: "",
  provider_npi: "",
  patient_reference: "",
  diagnosis: "",
};

function statusStyle(status: RequestStatus) {
  switch (status) {
    case "completed":
      return "bg-emerald-50 text-emerald-700 ring-emerald-600/20";
    case "ready_for_review":
      return "bg-blue-50 text-blue-700 ring-blue-600/20";
    case "in_review":
      return "bg-violet-50 text-violet-700 ring-violet-600/20";
    case "processing":
      return "bg-amber-50 text-amber-700 ring-amber-600/20";
    case "error":
      return "bg-red-50 text-red-700 ring-red-600/20";
    default:
      return "bg-slate-100 text-slate-600 ring-slate-500/20";
  }
}

function statusLabel(status: RequestStatus) {
  const labels: Record<RequestStatus, string> = {
    draft: "Draft",
    ready_for_review: "Siap direview",
    in_review: "Dalam review",
    needs_information: "Perlu informasi",
    ready_for_submission: "Siap diajukan",
    submitted: "Diajukan",
    completed: "Selesai",
    cancelled: "Dibatalkan",
    processing: "Diproses",
    error: "Error",
  };
  return labels[status];
}

function ClinicIllustration({ offset }: { offset: { x: number; y: number } }) {
  return (
    <svg
      viewBox="0 0 640 380"
      aria-hidden="true"
      className="h-full w-full"
      fill="none"
    >
      <ellipse cx="320" cy="326" rx="254" ry="30" fill="#DCE9E7" />
      <g
        className="motion-safe:transition-transform motion-safe:duration-300"
        style={{ transform: `translate(${offset.x * -0.22}px, ${offset.y * -0.22}px)` }}
      >
        <circle cx="471" cy="84" r="42" fill="#F4D99A" />
        <path d="M80 293V157l91-58 91 58v136H80Z" fill="#D7E7E4" />
        <path d="M103 293V169l68-43 68 43v124h-136Z" fill="#F8FBF9" />
        <path d="M132 185h26v27h-26zm52 0h26v27h-26zm-52 48h26v27h-26zm52 0h26v27h-26z" fill="#A9C9C7" />
      </g>
      <g
        className="motion-safe:transition-transform motion-safe:duration-300"
        style={{ transform: `translate(${offset.x * 0.12}px, ${offset.y * 0.12}px)` }}
      >
        <rect x="238" y="72" width="238" height="238" rx="18" fill="#FFFFFF" stroke="#B8D1CE" strokeWidth="4" />
        <path d="M238 127h238" stroke="#B8D1CE" strokeWidth="4" />
        <rect x="278" y="159" width="37" height="40" rx="5" fill="#C9E2DF" />
        <rect x="337" y="159" width="37" height="40" rx="5" fill="#C9E2DF" />
        <rect x="397" y="159" width="37" height="40" rx="5" fill="#C9E2DF" />
        <rect x="278" y="218" width="37" height="40" rx="5" fill="#C9E2DF" />
        <rect x="397" y="218" width="37" height="40" rx="5" fill="#C9E2DF" />
        <path d="M337 310v-43a20 20 0 0 1 40 0v43" fill="#EAF3F1" stroke="#B8D1CE" strokeWidth="4" />
        <rect x="330" y="83" width="54" height="34" rx="8" fill="#0F766E" />
        <path d="M352 90v20m-10-10h20" stroke="white" strokeWidth="5" strokeLinecap="round" />
      </g>
      <g
        className="motion-safe:transition-transform motion-safe:duration-300"
        style={{ transform: `translate(${offset.x * 0.3}px, ${offset.y * 0.3}px)` }}
      >
        <path d="M91 313c-1-24 8-39 21-39s22 15 21 39m379 0c-1-24 8-39 21-39s22 15 21 39" fill="#7FAE9E" />
        <path d="M112 312v-28m399 28v-28" stroke="#477C70" strokeWidth="4" />
        <circle cx="288" cy="287" r="13" fill="#D9A17D" />
        <path d="M267 318c2-17 9-25 21-25s19 8 21 25" fill="#D98671" />
        <path d="M279 316v13m18-13v13" stroke="#355B63" strokeWidth="5" strokeLinecap="round" />
        <circle cx="401" cy="284" r="12" fill="#C98D6B" />
        <path d="M380 318c2-17 9-25 21-25s19 8 21 25" fill="#6B9D96" />
        <path d="M392 316v13m18-13v13" stroke="#355B63" strokeWidth="5" strokeLinecap="round" />
      </g>
    </svg>
  );
}

function formatDate(value: string) {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "-";

  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function getErrorMessage(err: unknown): string {
  if (
    typeof err === "object" &&
    err !== null &&
    "response" in err
  ) {
    const response = (
      err as {
        response?: { data?: { error?: string } };
      }
    ).response;

    if (response?.data?.error) return response.data.error;
  }

  return "Terjadi kesalahan. Silakan coba kembali.";
}

export default function PARequestsPage() {
  const [items, setItems] = useState<PARequest[]>([]);
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [status, setStatus] = useState<RequestStatus | "">("");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [pageError, setPageError] = useState("");
  const [formError, setFormError] = useState("");
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [form, setForm] = useState<RequestForm>(initialForm);
  const [checklistOptions, setChecklistOptions] = useState<ActiveChecklistOption[]>([]);
  const [checklistOptionsLoading, setChecklistOptionsLoading] = useState(true);
  const [pairSelection, setPairSelection] = useState("");
  const [providerLookup, setProviderLookup] = useState<ProviderLookup | null>(null);
  const [providerLookupError, setProviderLookupError] = useState("");
  const [providerLookupLoading, setProviderLookupLoading] = useState(false);
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  const [parallaxOffset, setParallaxOffset] = useState({ x: 0, y: 0 });
  const referralChecklistOptions = checklistOptions;

  const fetchRequests = useCallback(async () => {
    try {
      const response = await api.get<ListResponse>("/pa-requests", {
        params: {
          page,
          limit,
          ...(status ? { status } : {}),
        },
      });

      setPageError("");
      setItems(response.data.data);
      setTotal(response.data.pagination.total);
      setTotalPages(response.data.pagination.total_pages);
    } catch (err: unknown) {
      setPageError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [page, limit, status]);

  useEffect(() => {
    const pendingFetch = Promise.resolve().then(() => fetchRequests());
    void pendingFetch;
  }, [fetchRequests]);

  useEffect(() => {
    let active = true;
    async function loadCurrentUser() {
      try {
        const response = await api.get<{ user: AuthUser }>("/auth/me");
        if (!active) return;
        setCurrentUser(response.data.user);
        if (response.data.user.role === "reviewer") setStatus("ready_for_review");
        localStorage.setItem("paa_user", JSON.stringify(response.data.user));
      } catch {
        if (active) setCurrentUser(getStoredUser());
      }
    }
    void loadCurrentUser();
    return () => { active = false; };
  }, []);

  useEffect(() => {
    let active = true;
    api.get<{ data: ActiveChecklistOption[] }>("/checklist-options")
      .then((response) => { if (active) setChecklistOptions(response.data.data ?? []); })
      .catch(() => { if (active) setChecklistOptions([]); })
      .finally(() => { if (active) setChecklistOptionsLoading(false); });
    return () => { active = false; };
  }, []);

  function chooseChecklistPair(value: string) {
    setPairSelection(value);
    const option = checklistOptions.find((item) => String(item.id) === value);
    if (option) setForm({ ...form, payer_name: option.payer_name, service_type: option.service_type });
    else if (value === "manual") setForm({ ...form, payer_name: "", service_type: "" });
  }

  function fillSyntheticTestData() {
    const selected = checklistOptions.find((item) => String(item.id) === pairSelection) ?? checklistOptions[0];
    setPairSelection(selected ? String(selected.id) : "manual");
    setForm({
      ...form,
      payer_name: selected?.payer_name ?? "Synthetic Health Plan",
      service_type: selected?.service_type ?? "Outpatient MRI",
      provider_npi: "1234567893",
      patient_reference: "SYN-MEMBER-001",
      diagnosis: "Synthetic test diagnosis",
    });
    setFormError("");
  }

  function changeStatus(value: string) {
    setLoading(true);
    setStatus(value as RequestStatus | "");
    setPage(1);
  }

  async function lookupProviderNPI() {
    const npi = form.provider_npi.trim();
    setProviderLookup(null);
    setProviderLookupError("");
    if (!/^\d{10}$/.test(npi)) {
      setProviderLookupError("NPI must contain 10 digits.");
      return;
    }

    setProviderLookupLoading(true);
    try {
      const response = await api.get<{ data: ProviderLookup }>(`/providers/npi/${npi}`);
      setProviderLookup(response.data.data);
    } catch (error: unknown) {
      setProviderLookupError(getErrorMessage(error));
    } finally {
      setProviderLookupLoading(false);
    }
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setFormError("");

    const payload = {
      payer_name: form.payer_name.trim(),
      service_type: form.service_type.trim(),
      provider_npi: form.provider_npi.trim(),
      patient_reference: form.patient_reference.trim(),
      diagnosis: form.diagnosis.trim(),
    };

    if (
      !payload.payer_name ||
      !payload.service_type ||
      !payload.patient_reference ||
      !payload.diagnosis
    ) {
      setFormError("Lengkapi seluruh field wajib.");
      setSubmitting(false);
      return;
    }

    try {
      const createResponse = await api.post<CreateResponse>("/pa-requests", payload);

      setForm(initialForm);
      setPairSelection("");
      setShowCreateForm(false);
      setStatus("");
      setSearch("");
      setPage(1);

      // Refresh explicitly so newly created requests appear immediately.
      const response = await api.get<ListResponse>("/pa-requests", {
        params: { page: 1, limit },
      });

      setItems(response.data.data);
      setTotal(response.data.pagination.total);
      setTotalPages(response.data.pagination.total_pages);
      if (!createResponse.data.data.checklist_version_id) {
        setPageError("Draft tersimpan. Admin perlu menyiapkan dan mengaktifkan checklist yang cocok sebelum permohonan dapat dikirim untuk review.");
      }
    } catch (err: unknown) {
      setFormError(getErrorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  const filteredItems = items.filter((item) => {
    const query = search.trim().toLowerCase();

    if (!query) return true;

    return [
      item.request_number,
      item.payer_name,
      item.service_type,
      item.patient_reference,
      item.diagnosis,
      item.status,
    ].some((value) => value?.toLowerCase().includes(query));
  });
  const canCreateRequest = currentUser?.role === "staff";
  const roleView = {
    admin: {
      title: "Ikhtisar seluruh permohonan",
      description: "Kelola akun serta checklist payer dan layanan. Pantau permohonan tanpa mengubah kasus milik Staff.",
      listTitle: "Semua permohonan",
      listDescription: "Buka alur untuk melihat dokumen, field, dan checklist dari permohonan mana pun.",
    },
    staff: {
      title: "Permohonan saya",
      description: "Daftar ini hanya berisi permohonan Anda. Siapkan data, unggah dokumen, lalu kirim ke antrian Reviewer.",
      listTitle: "Permohonan milik saya",
      listDescription: "Buat permohonan, lalu lengkapi dokumen, verifikasi field, dan checklist pada alur yang sama.",
    },
    reviewer: {
      title: "Antrian peninjauan",
      description: "Antrian berisi permohonan yang siap ditinjau. Reviewer memverifikasi hasil ekstraksi dan checklist, lalu mengembalikan atau menyelesaikan review.",
      listTitle: "Permohonan untuk ditinjau",
      listDescription: "Pilih permohonan untuk memeriksa dokumen, memperbaiki field ekstraksi, atau mengevaluasi checklist.",
    },
  }[currentUser?.role ?? "staff"];

  return (
    <main className="min-h-screen bg-slate-50 text-slate-900">
      <header className="flex min-h-20 items-center justify-between border-b border-slate-200 bg-white px-5 sm:px-8">
        <div className="flex items-center gap-3">
          <Link
            href="/dashboard"
            aria-label="Kembali ke dashboard"
            className="rounded-lg border border-slate-200 p-2 text-slate-600 hover:bg-slate-50"
          >
            <ArrowLeft size={18} />
          </Link>
          <div>
            <p className="text-sm text-slate-500">Workspace / PA Requests</p>
            <h1 className="font-semibold">{roleView.listTitle}</h1>
          </div>
        </div>

        {canCreateRequest && (
          <button
            onClick={() => {
              setFormError("");
              setShowCreateForm(true);
            }}
            className="flex min-h-11 items-center gap-2 rounded-xl bg-blue-700 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
          >
            <Plus size={17} aria-hidden="true" />
            <span className="hidden sm:inline">Buat permohonan</span>
            <span className="sm:hidden">Buat baru</span>
          </button>
        )}
      </header>

      <div className="mx-auto max-w-7xl space-y-6 p-5 sm:p-8">
        <section>
          <h2 className="text-2xl font-semibold tracking-tight">
            {roleView.title}
          </h2>
          <p className="mt-2 text-sm text-slate-500">
            {roleView.description}
          </p>
          <p className="mt-3 inline-flex rounded-full border border-teal-200 bg-teal-50 px-3 py-1.5 text-xs font-semibold text-teal-900">
            Tampilan {currentUser?.role === "admin" ? "Admin" : currentUser?.role === "reviewer" ? "Reviewer" : "Staff"}
          </p>
        </section>

        <section className="grid gap-4 sm:grid-cols-2">
          <article className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="flex items-center justify-between">
              <p className="text-sm text-slate-500">Total permohonan</p>
              <ClipboardList size={20} className="text-blue-600" />
            </div>
            <p className="mt-3 text-3xl font-semibold tabular-nums">
              {loading ? "—" : total}
            </p>
            <p className="mt-1 text-xs text-slate-400">
              Sesuai akses akun saat ini
            </p>
          </article>

          <article className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="flex items-center justify-between">
              <p className="text-sm text-slate-500">Di halaman ini</p>
              <FilePlus2 size={20} className="text-violet-600" />
            </div>
            <p className="mt-3 text-3xl font-semibold tabular-nums">
              {loading ? "—" : items.length}
            </p>
            <p className="mt-1 text-xs text-slate-400">
              Maksimal {limit} request per halaman
            </p>
          </article>
        </section>

        <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white">
          <div className="flex flex-col gap-4 border-b border-slate-200 p-5 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h3 className="font-semibold">{roleView.listTitle}</h3>
              <p className="mt-1 text-sm text-slate-500">
                {roleView.listDescription}
              </p>
            </div>

            <div className="flex flex-col gap-3 sm:flex-row">
              <div className="relative">
                <Search
                  size={17}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                />
                <input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Cari di daftar ini..."
                  aria-label="Cari pada halaman ini"
                  className="w-full rounded-xl border border-slate-200 py-2.5 pl-9 pr-3 text-sm outline-none focus:border-blue-500 sm:w-56"
                />
              </div>

              <select
                value={status}
                onChange={(event) => changeStatus(event.target.value)}
                aria-label="Filter status"
                className="rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm outline-none focus:border-blue-500"
              >
                {statuses.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>

              <button
                onClick={() => {
                  setLoading(true);
                  void fetchRequests();
                }}
                disabled={loading}
                aria-label="Refresh request list"
                className="inline-flex items-center justify-center rounded-xl border border-slate-200 px-3 py-2.5 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
              >
                <RefreshCw
                  size={17}
                  className={loading ? "animate-spin" : ""}
                />
              </button>
            </div>
          </div>

          {pageError && (
            <div className="m-5 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
              <AlertCircle size={19} className="mt-0.5 shrink-0" />
              <div className="flex-1">
                <p className="font-medium">Gagal memuat request</p>
                <p className="mt-1">{pageError}</p>
              </div>
              <button
                onClick={() => {
                  setLoading(true);
                  void fetchRequests();
                }}
                className="font-semibold underline"
              >
                Coba lagi
              </button>
            </div>
          )}

          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-left text-sm">
              <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-5 py-4 font-medium">Request</th>
                  <th className="px-5 py-4 font-medium">Skema / Alur</th>
                  <th className="px-5 py-4 font-medium">Referensi pasien</th>
                  <th className="px-5 py-4 font-medium">Status</th>
                  <th className="px-5 py-4 font-medium">Created</th>
                  <th className="px-5 py-4 font-medium">Action</th>
                </tr>
              </thead>

              <tbody className="divide-y divide-slate-100">
                {loading ? (
                  <tr>
                    <td colSpan={6} className="px-5 py-16 text-center">
                      <LoaderCircle
                        size={24}
                        className="mx-auto animate-spin text-blue-600"
                      />
                      <p className="mt-3 text-slate-500">
                        Memuat kasus...
                      </p>
                    </td>
                  </tr>
                ) : filteredItems.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-5 py-16 text-center">
                      <ClipboardList
                        size={30}
                        className="mx-auto text-slate-300"
                      />
                      <p className="mt-3 font-medium text-slate-700">
                        Belum ada kasus
                      </p>
                      <p className="mt-1 text-sm text-slate-500">
                        Coba ubah pencarian atau filter status.
                      </p>
                    </td>
                  </tr>
                ) : (
                  filteredItems.map((item) => (
                    <tr
                      key={item.id}
                      className="transition hover:bg-slate-50/80"
                    >
                      <td className="px-5 py-4">
                        <p className="font-semibold text-slate-800">
                          {item.request_number}
                        </p>
                        <p className="mt-1 text-xs text-slate-400">
                          ID: {item.id}
                        </p>
                      </td>

                      <td className="px-5 py-4">
                        <p className="font-medium">{item.payer_name}</p>
                        <p className="mt-1 text-xs text-slate-500">
                          {item.service_type}
                        </p>
                      </td>

                      <td className="px-5 py-4 text-slate-600">
                        {item.patient_reference}
                      </td>

                      <td className="px-5 py-4">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium capitalize ring-1 ring-inset ${statusStyle(item.status)}`}
                        >
                          {statusLabel(item.status)}
                        </span>
                      </td>

                      <td className="px-5 py-4 text-slate-500">
                        {formatDate(item.created_at)}
                      </td>

                      <td className="px-5 py-4">
                        <div className="flex flex-wrap gap-x-3 gap-y-2">
                          <Link
                            href={`/requests/${item.id}`}
                            className="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-2 font-medium text-blue-800 transition hover:bg-blue-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
                          >
                            Buka alur <ArrowRight size={15} aria-hidden="true" />
                          </Link>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <div className="flex flex-col gap-3 border-t border-slate-200 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-slate-500">
              {total === 0
                ? "Tidak ada data"
                : `Halaman ${page} dari ${Math.max(totalPages, 1)} · ${total} request`}
            </p>

            <div className="flex items-center gap-2">
              <button
                onClick={() => {
                  setLoading(true);
                  setPage((current) => Math.max(1, current - 1));
                }}
                disabled={loading || page <= 1}
                className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-40"
              >
                <ArrowLeft size={15} /> Previous
              </button>

              <button
                onClick={() => {
                  setLoading(true);
                  setPage((current) =>
                    Math.min(Math.max(totalPages, 1), current + 1)
                  );
                }}
                disabled={loading || page >= totalPages || totalPages === 0}
                className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-40"
              >
                Next <ArrowRight size={15} />
              </button>
            </div>
          </div>
        </section>
      </div>

      {showCreateForm && (
        <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/50 p-4 sm:items-center">
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="create-request-title"
            className="my-4 w-full max-w-6xl rounded-2xl bg-white shadow-xl"
          >
            <div className="flex items-center justify-between border-b border-slate-200 px-6 py-5">
              <div>
                <h2 id="create-request-title" className="font-semibold">
                  Buat draft permohonan prior authorization
                </h2>
                <p className="mt-1 text-sm text-slate-500">
                  Kumpulkan informasi dan dokumen yang diperlukan sebelum diajukan ke payer.
                </p>
              </div>
              <button
                onClick={() => setShowCreateForm(false)}
                aria-label="Tutup form"
                className="rounded-lg p-2 text-slate-500 hover:bg-slate-100"
              >
                <X size={19} />
              </button>
            </div>

            <form onSubmit={handleCreate} className="p-6">
              <div className="grid gap-6 lg:grid-cols-[0.72fr_1.28fr]">
                <aside className="flex flex-col rounded-2xl border border-[#d5e5e2] bg-[#eef5f2] p-5 sm:p-6">
                  <div
                    className="relative -mx-2 h-48 overflow-hidden sm:h-56 lg:h-64"
                    onPointerMove={(event) => {
                      if (
                        event.pointerType !== "mouse" ||
                        window.matchMedia("(prefers-reduced-motion: reduce)").matches
                      ) return;

                      const bounds = event.currentTarget.getBoundingClientRect();
                      setParallaxOffset({
                        x: ((event.clientX - bounds.left) / bounds.width - 0.5) * 18,
                        y: ((event.clientY - bounds.top) / bounds.height - 0.5) * 14,
                      });
                    }}
                    onPointerLeave={() => setParallaxOffset({ x: 0, y: 0 })}
                  >
                    <ClinicIllustration offset={parallaxOffset} />
                  </div>
                  <div className="mt-auto pt-4">
                    <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-[#bfd7d2] bg-white/80 px-3 py-1 text-xs font-medium text-[#27635c]">
                      <Building2 size={14} aria-hidden="true" />
                      Prior authorization
                    </div>
                    <h3 className="max-w-sm text-xl font-semibold leading-snug text-[#173f45] sm:text-2xl">
                      Siapkan permohonan dengan informasi sumber yang bisa ditinjau.
                    </h3>
                    <p className="mt-2 max-w-sm text-sm leading-6 text-slate-600">
                      Unggah dokumen pendukung. Nilai ekstraksi perlu diperiksa oleh Reviewer.
                    </p>
                  </div>
                </aside>

                <div className="space-y-5">
              <div className="grid gap-5 sm:grid-cols-2">
                <div className="sm:col-span-2 rounded-lg bg-[#f3f8f6] p-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div><h3 className="text-sm font-semibold text-[#173f45]">Konteks permohonan</h3><p className="mt-1 text-xs text-slate-600">Gunakan data sintetis atau data yang sudah di-de-identifikasi untuk demo.</p></div>
                    <button type="button" onClick={fillSyntheticTestData} className="min-h-10 rounded-lg border border-[#205b55] bg-white px-3 text-sm font-semibold text-[#205b55] hover:bg-[#e6f1ee]">Isi data uji otomatis</button>
                  </div>
                </div>

                <div className="sm:col-span-2">
                  <label htmlFor="checklist-pair" className="mb-2 block text-sm font-medium">Checklist payer dan layanan</label>
                  <select id="checklist-pair" value={pairSelection} onChange={(event) => chooseChecklistPair(event.target.value)} className="min-h-11 w-full rounded-xl border border-slate-400 bg-white px-3 py-2.5 text-sm focus:border-blue-600 focus:ring-2 focus:ring-blue-100">
                    <option value="">{checklistOptionsLoading ? "Memuat checklist…" : "Pilih checklist atau masukkan manual"}</option>
                    <option value="manual">Belum ada checklist yang ditinjau</option>
                    {referralChecklistOptions.map((option) => <option key={option.id} value={String(option.id)}>{option.name} · v{option.version_number}</option>)}
                  </select>
                  <p className="mt-1 text-xs text-slate-600">Pilih checklist aktif yang cocok atau masukkan payer dan layanan secara manual. Checklist harus disiapkan Admin sebelum permohonan dapat dikirim untuk review.</p>
                </div>
                <label className="space-y-1.5 text-sm font-medium text-slate-800">Payer / health plan *<input required maxLength={150} value={form.payer_name} onChange={(event) => setForm({ ...form, payer_name: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 font-normal" /></label>
                <label className="space-y-1.5 text-sm font-medium text-slate-800">Requested service *<input required maxLength={100} value={form.service_type} onChange={(event) => setForm({ ...form, service_type: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 font-normal" /></label>

                <label className="space-y-1.5 text-sm font-medium text-slate-800">
                  Patient / member reference *
                  <input
                    required
                    maxLength={100}
                    value={form.patient_reference}
                    onChange={(event) => setForm({ ...form, patient_reference: event.target.value })}
                    placeholder="Synthetic member ID"
                    className="min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 font-normal focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]"
                  />
                  <span className="block text-xs font-normal text-slate-600">Use synthetic or de-identified data for testing.</span>
                </label>

                <div className="space-y-1.5 text-sm font-medium text-slate-800">
                  <label htmlFor="provider-npi">Provider NPI (optional)</label>
                  <div className="flex gap-2">
                    <input id="provider-npi" maxLength={10} inputMode="numeric" value={form.provider_npi} onChange={(event) => { setForm({ ...form, provider_npi: event.target.value.replace(/\D/g, "").slice(0, 10) }); setProviderLookup(null); setProviderLookupError(""); }} placeholder="10-digit NPI" className="min-h-11 min-w-0 flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2.5 font-normal" />
                    <button type="button" onClick={() => void lookupProviderNPI()} disabled={providerLookupLoading} className="min-h-11 rounded-lg border border-slate-300 px-3 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50">{providerLookupLoading ? "Searching…" : "Lookup"}</button>
                  </div>
                  <span className="block text-xs font-normal text-slate-600">Optional; 10 digits when provided. Lookup uses the CMS NPI Registry.</span>
                  {providerLookupError && <p role="status" className="text-xs font-normal text-amber-800">{providerLookupError}</p>}
                  {providerLookup && <p role="status" className="rounded-md bg-slate-50 p-2 text-xs font-normal text-slate-700">{providerLookup.name}{providerLookup.city && providerLookup.state ? ` · ${providerLookup.city}, ${providerLookup.state}` : ""}{providerLookup.registry_notice ? ` · ${providerLookup.registry_notice}` : ""}</p>}
                </div>

                <div className="sm:col-span-2">
                  <label className="mb-2 block text-sm font-medium">
                    Diagnosis / diagnosis code *
                  </label>
                  <textarea
                    required
                    maxLength={2000}
                    rows={3}
                    value={form.diagnosis}
                    onChange={(event) =>
                      setForm({ ...form, diagnosis: event.target.value })
                    }
                    placeholder="Enter the diagnosis documented for this request. Use synthetic details for testing."
                    className="w-full resize-y rounded-xl border border-slate-200 px-3 py-2.5 text-sm outline-none focus:border-blue-500"
                  />
                </div>


              </div>

              {formError && (
                <div
                  role="alert"
                  className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
                >
                  {formError}
                </div>
              )}

              <div className="flex justify-end gap-3 border-t border-slate-100 pt-5">
                <button
                  type="button"
                  onClick={() => setShowCreateForm(false)}
                  className="rounded-xl border border-slate-200 px-4 py-2.5 text-sm font-medium hover:bg-slate-50"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="inline-flex items-center gap-2 rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {submitting && (
                    <LoaderCircle size={16} className="animate-spin" />
                  )}
                  {submitting ? "Saving…" : "Save draft"}
                </button>
              </div>
                </div>
              </div>
            </form>
          </section>
        </div>
      )}
    </main>
  );
}
