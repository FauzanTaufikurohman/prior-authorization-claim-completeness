
"use client";

import {
  Suspense,
  use,
  useCallback,
  useEffect,
  useState,
} from "react";
import Link from "next/link";
import axios from "axios";
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  ClipboardCheck,
  FileText,
  LoaderCircle,
  RefreshCw,
  Save,
  Upload,
  AlertCircle,
  CircleHelp,
  Play,
  Clock3,
  UserRound,
  Building2,
  CalendarDays,
  ShieldCheck,
  FileUp,
  XCircle,
  Download,
  Trash2,
  Copy,
} from "lucide-react";

import api from "@/lib/api";
import type { AuthUser } from "@/lib/auth";

type PARequest = {
  id: number;
  request_number?: string;
  patient_name?: string;
  patient_reference?: string;
  patient_id?: string;
  payer_name?: string;
  insurance_name?: string;
  procedure_name?: string;
  service_name?: string;
  service_type?: string;
  diagnosis?: string;
  status?: string;
  checklist_version_id?: number | null;
  provider_npi?: string;
  ordering_provider_npi?: string;
  servicing_provider_npi?: string;
  priority?: string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
};

type ExtractedField = {
  id: number;
  pa_request_id?: number;
  field_key: string;
  field_value: string | null;
  verification_status: string;
  confidence?: number | null;
  evidence_text?: string | null;
  source_page?: number | null;
  created_at?: string;
  updated_at?: string;
};

type ChecklistResult = {
  id?: number;
  rule_id?: number;
  checklist_rule_id?: number;
  rule_name?: string;
  rule_key?: string;
  description?: string;
  status: string;
  message?: string;
  reason?: string;
  field_key?: string;
  [key: string]: unknown;
};

type UploadedDocument = {
  id: number;
  pa_request_id?: number;
  original_filename?: string;
  mime_type?: string;
  size_bytes?: number;
  parsing_status?: string;
  created_at?: string;
};

type RequestResponse = {
  data: PARequest;
};

type FieldsResponse = {
  data: {
    fields: ExtractedField[];
    count: number;
  };
};

type ChecklistResponse = {
  pa_request_id: number;
  total: number;
  results: ChecklistResult[];
};

type DocumentResponse = {
  data: UploadedDocument[];
};

type StatusHistory = { id: number; from_status: string | null; to_status: string; action: string; note: string | null; actor_name: string | null; created_at: string };
type ChecklistOption = { id: number; payer_name: string; service_type: string; rules: { required_field_key: string | null; is_required: boolean }[] };

type RequestDraft = {
  payer_name: string;
  service_type: string;
  ordering_provider_npi: string;
  servicing_provider_npi: string;
  patient_reference: string;
  diagnosis: string;
};

const workflowSteps = [
  {
    id: "overview",
    path: "",
    label: "Ringkasan",
    description: "Periksa konteks dan informasi utama kasus.",
  },
  {
    id: "documents",
    path: "documents",
    label: "Dokumen",
    description: "Unggah berkas pendukung dan jalankan pembacaan.",
  },
  {
    id: "fields",
    path: "fields",
    label: "Verifikasi informasi",
    description: "Tinjau hasil pembacaan sebelum evaluasi.",
  },
  {
    id: "checklist",
    path: "checklist",
    label: "Checklist",
    description: "Tinjau kelengkapan dan unduh ringkasan.",
  },
] as const;

type WorkflowStepID = (typeof workflowSteps)[number]["id"];

function errorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const responseData = error.response?.data as
      | { message?: string; error?: string }
      | undefined;

    if (responseData?.message) {
      return responseData.message;
    }

    if (responseData?.error) {
      return responseData.error;
    }

    if (error.response?.status === 401) {
      return "Sesi login berakhir. Silakan login kembali.";
    }

    if (error.response?.status === 403) {
      return "Anda tidak memiliki izin untuk melakukan tindakan ini.";
    }

    if (error.response?.status === 404) {
      return "Data yang diminta tidak ditemukan.";
    }

    if (!error.response) {
      return "Tidak dapat terhubung ke server. Periksa koneksi backend.";
    }
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "Terjadi kesalahan yang tidak diketahui.";
}

function workflowErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error) && error.response?.status === 404) {
    return "Rute workflow belum tersedia di backend yang sedang berjalan. Restart backend Go pada port 8081 dari source terbaru, lalu muat ulang halaman.";
  }
  if (axios.isAxiosError(error) && !error.response && (error.code === "ECONNABORTED" || error.code === "ETIMEDOUT")) {
    return "Pemeriksaan workflow memerlukan waktu lebih lama dari batas tunggu. Muat ulang status permohonan sebelum mencoba aksi lagi.";
  }
  if (axios.isAxiosError(error) && error.response?.status === 500) {
    const responseMessage = errorMessage(error);
    return responseMessage !== error.message
      ? responseMessage
      : "Backend mengembalikan 500. Periksa terminal backend Go untuk error database pada transisi permohonan.";
  }
  return errorMessage(error);
}

function formatDate(value?: string): string {
  if (!value) return "-";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function formatFileSize(bytes?: number): string {
  if (bytes === undefined || bytes === null || bytes < 0) {
    return "-";
  }

  if (bytes < 1024) {
    return `${bytes} B`;
  }

  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }

  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function humanize(value?: string): string {
  if (!value) return "-";

  return value
    .replaceAll("_", " ")
    .replaceAll("-", " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function requestStatusClass(status?: string): string {
  const value = status?.toLowerCase() ?? "";

  if (
    ["approved", "completed", "fulfilled", "approved_by_payer"].includes(
      value,
    )
  ) {
    return "bg-emerald-50 text-emerald-700 ring-emerald-600/20";
  }

  if (["rejected", "failed", "denied"].includes(value)) {
    return "bg-red-50 text-red-700 ring-red-600/20";
  }

  if (
    ["pending", "in_review", "needs_review", "processing"].includes(value)
  ) {
    return "bg-amber-50 text-amber-700 ring-amber-600/20";
  }

  if (["ready_for_review", "ready_for_submission"].includes(value)) {
    return "bg-blue-50 text-blue-700 ring-blue-600/20";
  }

  if (value === "submitted") {
    return "bg-violet-50 text-violet-700 ring-violet-600/20";
  }
  if (value === "payer_needs_information") {
    return "bg-amber-50 text-amber-800 ring-amber-600/20";
  }
  if (value === "appeal_submitted") {
    return "bg-violet-50 text-violet-700 ring-violet-600/20";
  }

  return "bg-slate-100 text-slate-700 ring-slate-500/20";
}

function checklistStatusClass(status?: string): string {
  switch (status?.toLowerCase()) {
    case "fulfilled":
    case "passed":
    case "complete":
    case "completed":
      return "bg-emerald-50 text-emerald-700 ring-emerald-600/20";

    case "missing":
    case "failed":
    case "not_fulfilled":
      return "bg-red-50 text-red-700 ring-red-600/20";

    case "needs_review":
    case "pending":
    case "review":
      return "bg-amber-50 text-amber-700 ring-amber-600/20";

    default:
      return "bg-slate-100 text-slate-700 ring-slate-500/20";
  }
}

function verificationStatusClass(status?: string): string {
  switch (status?.toLowerCase()) {
    case "verified":
    case "approved":
    case "corrected":
      return "bg-emerald-50 text-emerald-700";

    case "needs_review":
    case "pending":
    case "uncertain":
    default:
      return "bg-amber-50 text-amber-700";
  }
}

function displayValue(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }

  if (typeof value === "string" || typeof value === "number") {
    return String(value);
  }

  return "-";
}

function csvCell(value: unknown): string {
  let text = value === null || value === undefined ? "" : String(value);
  if (/^[\t\r ]*[=+\-@]/.test(text)) {
    text = `'${text}`;
  }
  return `"${text.replaceAll('"', '""')}"`;
}

function SectionHeading({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description?: string;
}) {
  return (
    <div className="mb-5 flex items-start gap-3">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-blue-700">
        {icon}
      </div>

      <div>
        <h2 className="text-base font-semibold text-slate-900">{title}</h2>
        {description && (
          <p className="mt-1 text-sm leading-5 text-slate-500">
            {description}
          </p>
        )}
      </div>
    </div>
  );
}

function InfoItem({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="min-w-0">
      <div className="flex items-center gap-2 text-xs font-medium text-slate-500">
        {icon}
        <span>{label}</span>
      </div>
      <p className="mt-2 break-words text-sm font-medium text-slate-900">
        {value}
      </p>
    </div>
  );
}

export default function PARequestDetailPage({
  params,
}: {
  params: Promise<{ id: string; step?: string[] }>;
}) {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-64 items-center justify-center">
          <LoaderCircle className="h-6 w-6 animate-spin text-blue-600" />
          <span className="ml-3 text-sm text-slate-500">
            Memuat detail permintaan...
          </span>
        </div>
      }
    >
      <PARequestDetailContent params={params} />
    </Suspense>
  );
}

function PARequestDetailContent({
  params,
}: {
  params: Promise<{ id: string; step?: string[] }>;
}) {
  const { id, step: routeSteps } = use(params);
  const requestId = id;
  const requestedStep = routeSteps?.[0] ?? "overview";
  const isKnownStep =
    (routeSteps?.length ?? 0) <= 1 &&
    workflowSteps.some((step) => step.id === requestedStep);
  const activeStep: WorkflowStepID = isKnownStep
    ? (requestedStep as WorkflowStepID)
    : "overview";
  const activeStepIndex = workflowSteps.findIndex(
    (step) => step.id === activeStep,
  );
  const currentStep = workflowSteps[activeStepIndex];

  const [request, setRequest] = useState<PARequest | null>(null);
  const [requestDraft, setRequestDraft] = useState<RequestDraft>({ payer_name: "", service_type: "", ordering_provider_npi: "", servicing_provider_npi: "", patient_reference: "", diagnosis: "" });
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  const [fields, setFields] = useState<ExtractedField[]>([]);
  const [checklist, setChecklist] = useState<ChecklistResult[]>([]);
  const [documents, setDocuments] = useState<UploadedDocument[]>([]);
  const [statusHistory, setStatusHistory] = useState<StatusHistory[]>([]);
  const [sampleFieldKeys, setSampleFieldKeys] = useState<string[]>([]);
  const [historyUnavailable, setHistoryUnavailable] = useState(false);
  const [patientUploadLink, setPatientUploadLink] = useState("");

  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [creatingUploadLink, setCreatingUploadLink] = useState(false);
  const [parsingDocumentID, setParsingDocumentID] = useState<number | null>(
    null,
  );
  const [deletingDocumentID, setDeletingDocumentID] = useState<number | null>(null);
  const [confirmedPDFDocumentIDs, setConfirmedPDFDocumentIDs] = useState<
    number[]
  >([]);
  const [evaluating, setEvaluating] = useState(false);
  const [savingField, setSavingField] = useState<number | null>(null);

  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    api.get<{ user: AuthUser }>("/auth/me").then((response) => setCurrentUser(response.data.user)).catch(() => setCurrentUser(null));
  }, []);

  const loadData = useCallback(
    async (options?: { silent?: boolean }) => {
      const silent = options?.silent ?? false;
      let loadedRequest: PARequest | null = null;

      try {
        const requestResponse = await api.get<RequestResponse>(
          `/pa-requests/${requestId}`,
        );
        setError("");
        loadedRequest = requestResponse.data.data;
        setRequest(loadedRequest);
        setRequestDraft({
          payer_name: String(loadedRequest.payer_name ?? ""),
          service_type: String(loadedRequest.service_type ?? ""),
          ordering_provider_npi: String(loadedRequest.ordering_provider_npi ?? loadedRequest.provider_npi ?? ""),
          servicing_provider_npi: String(loadedRequest.servicing_provider_npi ?? ""),
          patient_reference: String(loadedRequest.patient_reference ?? ""),
          diagnosis: String(loadedRequest.diagnosis ?? ""),
        });
        try {
          const history = await api.get<{ data: StatusHistory[] }>(`/pa-requests/${requestId}/status-history`);
          setStatusHistory(history.data.data ?? []);
          setHistoryUnavailable(false);
        } catch (cause) {
          setStatusHistory([]);
          setHistoryUnavailable(axios.isAxiosError(cause) && cause.response?.status === 404);
        }
      } catch (err) {
        setRequest(null);
        setError(errorMessage(err));
        if (!silent) {
          setLoading(false);
        }
        return;
      }

      try {
        switch (activeStep) {
          case "documents": {
            setDocuments([]);
            const response = await api.get<DocumentResponse>(`/pa-requests/${requestId}/documents`);
            setDocuments(response.data.data ?? []);
            try {
              const optionsResponse = await api.get<{ data: ChecklistOption[] }>("/checklist-options");
              const assigned = optionsResponse.data.data.find((item) => item.id === loadedRequest?.checklist_version_id);
              setSampleFieldKeys((assigned?.rules ?? []).filter((rule) => rule.is_required && rule.required_field_key).map((rule) => rule.required_field_key as string));
            } catch { setSampleFieldKeys([]); }
            break;
          }
          case "fields": {
            setFields([]);
            const response = await api.get<FieldsResponse>(
              `/pa-requests/${requestId}/fields`,
            );
            setFields(response.data.data.fields ?? []);
            break;
          }
          case "checklist": {
            setChecklist([]);
            const response = await api.get<ChecklistResponse>(
              `/pa-requests/${requestId}/checklist/results`,
            );
            setChecklist(response.data.results ?? []);
            break;
          }
        }
      } catch (err) {
        setError(errorMessage(err));
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [activeStep, requestId],
  );

  useEffect(() => {
    const pendingLoad = Promise.resolve().then(() => loadData());
    void pendingLoad;
  }, [loadData]);

  async function uploadDocument(
    event: React.FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    setError("");
    setNotice("");

    const form = event.currentTarget;
    const fileInput = form.elements.namedItem(
      "document",
    ) as HTMLInputElement | null;

    const file = fileInput?.files?.[0];

    if (!file) {
      setError("Pilih file CSV atau PDF terlebih dahulu.");
      return;
    }

    const allowedExtensions = [".csv", ".pdf"];
    const fileName = file.name.toLowerCase();

    if (!allowedExtensions.some((extension) => fileName.endsWith(extension))) {
      setError("Format file tidak didukung. Gunakan file CSV atau PDF.");
      return;
    }

    const maxFileSize = 10 * 1024 * 1024;

    if (file.size > maxFileSize) {
      setError("Ukuran file maksimal adalah 10 MB.");
      return;
    }

    const formData = new FormData();
    formData.append("file", file);

    setBusy(true);

    try {
      const response = await api.post(
        `/pa-requests/${requestId}/documents`,
        formData,
      );

      const uploadedDocument: UploadedDocument = response.data.data;

      if (uploadedDocument?.id) {
        setDocuments((current) => [
          uploadedDocument,
          ...current.filter((item) => item.id !== uploadedDocument.id),
        ]);
      } else {
        await loadData({ silent: true });
      }

      setNotice(
        "Dokumen berhasil diunggah. Lanjutkan ke proses parsing untuk mengekstrak field.",
      );
      form.reset();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function parseDocument(document: UploadedDocument) {
    setError("");
    setNotice("");

    const isPDF = document.mime_type === "application/pdf";
    if (isPDF && !confirmedPDFDocumentIDs.includes(document.id)) {
      setError(
        "Konfirmasi data sintetis atau data yang sudah di-de-identifikasi sebelum mengirim teks ke OpenAI.",
      );
      return;
    }

    setParsingDocumentID(document.id);

    try {
      const response = await api.post(
        `/pa-requests/${requestId}/documents/${document.id}/parse`,
        isPDF ? { confirm_synthetic_data: true } : undefined,
        { timeout: isPDF ? 80_000 : 15_000 },
      );

      const responseData = response.data?.data ?? response.data;

      setDocuments((current) =>
        current.map((item) =>
          item.id === document.id
            ? {
                ...item,
                parsing_status:
                  responseData?.parsing_status ?? "completed",
              }
            : item,
        ),
      );

      const extractedCount = responseData?.extracted_count;

      setNotice(
        typeof extractedCount === "number"
          ? `Parsing berhasil. ${extractedCount} field berhasil diproses.`
          : "Parsing dokumen berhasil dijalankan. Periksa hasil ekstraksi di bawah.",
      );

      await loadData({ silent: true });
    } catch (err) {
      setError(errorMessage(err));
      setDocuments((current) =>
        current.map((item) =>
          item.id === document.id
            ? { ...item, parsing_status: "failed" }
            : item,
        ),
      );
    } finally {
      setParsingDocumentID(null);
    }
  }

  async function deleteDocument(document: UploadedDocument) {
    const filename = document.original_filename ?? `dokumen #${document.id}`;
    if (!window.confirm(`Hapus ${filename}? File dan field hasil ekstraksinya akan dihapus, lalu hasil checklist perlu dievaluasi ulang.`)) return;
    setError("");
    setNotice("");
    setDeletingDocumentID(document.id);
    try {
      await api.delete(`/pa-requests/${requestId}/documents/${document.id}`);
      setDocuments((current) => current.filter((item) => item.id !== document.id));
      setConfirmedPDFDocumentIDs((current) => current.filter((id) => id !== document.id));
      setChecklist([]);
      setNotice(`Dokumen ${filename} dan hasil ekstraksinya dihapus.`);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setDeletingDocumentID(null);
    }
  }

  async function updateField(
    field: ExtractedField,
    fieldValue: string,
    verificationStatus: string,
  ) {
    setError("");
    setNotice("");
    setSavingField(field.id);

    try {
      await api.patch(
        `/pa-requests/${requestId}/fields/${field.id}`,
        currentUser?.role === "reviewer"
          ? { field_value: fieldValue, verification_status: verificationStatus }
          : { field_value: fieldValue },
      );

      await loadData({ silent: true });

      setNotice(
        `Field "${field.field_key}" berhasil diperbarui. Jalankan evaluasi checklist kembali untuk memperbarui hasil berdasarkan nilai terbaru.`,
      );
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSavingField(null);
    }
  }

  async function evaluateChecklist() {
    setError("");
    setNotice("");
    setEvaluating(true);

    try {
      await api.post(
        `/pa-requests/${requestId}/checklist/evaluate`,
      );

      const response = await api.get<ChecklistResponse>(
        `/pa-requests/${requestId}/checklist/results`,
      );

      setChecklist(response.data.results ?? []);

      setNotice(
        "Evaluasi checklist berhasil dijalankan menggunakan data field terbaru.",
      );
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setEvaluating(false);
    }
  }

  async function transition(action: string, extra: Record<string, string> = {}) {
    setError(""); setNotice(""); setBusy(true);
    try {
      await api.post(
        `/pa-requests/${requestId}/transitions`,
        { action, ...extra },
        { timeout: 100_000 },
      );
      await loadData({ silent: true });
      setNotice("Status permohonan berhasil disimpan.");
    } catch (cause) { setError(workflowErrorMessage(cause)); }
    finally { setBusy(false); }
  }

  async function createPatientUploadLink() {
    setError("");
    setNotice("");
    setCreatingUploadLink(true);
    try {
      const response = await api.post<{ data: { token: string } }>(
        `/pa-requests/${requestId}/upload-link`,
      );
      const link = new URL("/upload-user", window.location.origin);
      link.hash = response.data.data.token;
      setPatientUploadLink(link.toString());
      setNotice("Link upload pasien berhasil dibuat.");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setCreatingUploadLink(false);
    }
  }

  async function copyPatientUploadLink() {
    if (!patientUploadLink) return;
    try {
      await navigator.clipboard.writeText(patientUploadLink);
      setNotice("Link upload pasien berhasil disalin.");
      setError("");
    } catch {
      setError("Link tidak dapat disalin otomatis. Pilih dan salin link dari kolom yang tersedia.");
    }
  }

  async function saveRequestDetails(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(""); setNotice(""); setBusy(true);
    try {
      await api.put(`/pa-requests/${requestId}`, {
        ...requestDraft,
        provider_npi: requestDraft.ordering_provider_npi,
      });
      setNotice("Informasi kasus diperbarui.");
      await loadData({ silent: true });
    } catch (cause) { setError(errorMessage(cause)); }
    finally { setBusy(false); }
  }

  function downloadSyntheticCSV() {
    if (!request || sampleFieldKeys.length === 0) {
      setError("Field wajib checklist belum dapat dimuat. Pastikan checklist aktif dan backend sudah direstart.");
      return;
    }
    const sampleValues: Record<string, string> = {
      patient_reference: String(request.patient_reference ?? "SYN-PA-TEST-001"),
      payer_name: String(request.payer_name ?? "DATA UJI SINTETIS"),
      service_type: String(request.service_type ?? "DATA UJI SINTETIS"),
      provider_npi: "DATA UJI SINTETIS - NPI tidak diisi",
      ordering_provider_npi: "DATA UJI SINTETIS - NPI tidak diisi",
      servicing_provider_npi: "DATA UJI SINTETIS - NPI tidak diisi",
      diagnosis: String(request.diagnosis ?? "DATA UJI SINTETIS - bukan diagnosis pasien"),
      clinical_notes: "DATA UJI SINTETIS - bukan catatan klinis pasien",
      prior_treatments: "DATA UJI SINTETIS - contoh untuk menguji kelengkapan field, bukan informasi terapi pasien",
      diagnosis_code: "SYN-DIAG-001 - bukan kode klinis",
      requested_service: "Layanan uji sintetis",
      supporting_document_type: "Dokumen uji sintetis",
    };
    const rows = [["field", "value"], ...sampleFieldKeys.map((key) => [key, sampleValues[key] ?? `DATA UJI SINTETIS - contoh isi untuk ${key}`])];
    const csv = `\uFEFF${rows.map((row) => row.map(csvCell).join(",")).join("\r\n")}`;
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const link = window.document.createElement("a");
    link.href = url;
    link.download = `synthetic-pa-${request.id}.csv`;
    link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    setNotice("CSV data sintetis dibuat sesuai field wajib checklist untuk menguji alur.");
  }

  async function downloadSummary() {
    if (!request) return;

    setError("");
    try {
      const fieldsResponse = await api.get<FieldsResponse>(
        `/pa-requests/${requestId}/fields`,
      );
      const summaryFields = fieldsResponse.data.data.fields ?? [];
      const rows: unknown[][] = [
        ["Prior Authorization summary"],
        ["Request number", request.request_number],
        ["Patient reference", request.patient_reference],
        ["Payer", request.payer_name],
        ["Service", request.service_type],
        ["Ordering provider NPI", request.ordering_provider_npi ?? request.provider_npi],
        ["Servicing provider NPI", request.servicing_provider_npi],
        ["Diagnosis", request.diagnosis],
        ["Status", request.status],
        ["Checklist version ID", request.checklist_version_id],
        ["Generated at", new Date().toISOString()],
        [],
        ["Extracted fields"],
        ["Field", "Value", "Verification status", "Evidence"],
        ...summaryFields.map((field) => [
          field.field_key,
          field.field_value,
          field.verification_status,
          field.evidence_text,
        ]),
        [],
        ["Checklist results"],
        ["Rule", "Field", "Status", "Reason"],
        ...checklist.map((item) => [
          item.rule_key ?? item.description,
          item.field_key,
          item.status,
          item.reason ?? item.message,
        ]),
      ];

      const csv = `\uFEFF${rows.map((row) => row.map(csvCell).join(",")).join("\r\n")}`;
      const url = URL.createObjectURL(
        new Blob([csv], { type: "text/csv;charset=utf-8" }),
      );
      const link = window.document.createElement("a");
      const safeRequestNumber = (request.request_number ?? `request-${id}`)
        .replace(/[^a-zA-Z0-9_-]/g, "_");
      link.href = url;
      link.download = `${safeRequestNumber}-summary.csv`;
      link.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      setNotice("Ringkasan berkas berhasil diunduh.");
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  const fulfilledCount = checklist.filter((item) =>
    ["fulfilled", "passed", "complete", "completed"].includes(
      item.status?.toLowerCase(),
    ),
  ).length;

  const missingCount = checklist.filter((item) =>
    ["missing", "failed", "not_fulfilled"].includes(
      item.status?.toLowerCase(),
    ),
  ).length;

  const reviewCount = checklist.filter((item) =>
    ["needs_review", "pending", "review"].includes(
      item.status?.toLowerCase(),
    ),
  ).length;
  const previousStep = workflowSteps[activeStepIndex - 1];
  const nextStep = workflowSteps[activeStepIndex + 1];

  if (loading) {
    return (
      <div className="flex min-h-64 items-center justify-center">
        <LoaderCircle className="h-6 w-6 animate-spin text-blue-600" />
        <span className="ml-3 text-sm text-slate-500">
          Memuat detail permintaan...
        </span>
      </div>
    );
  }

  if (!request) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-12">
        <div className="rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm">
          <AlertCircle className="mx-auto h-10 w-10 text-red-500" />

          <h1 className="mt-4 text-lg font-semibold text-slate-900">
            Detail permintaan tidak tersedia
          </h1>

          <p className="mt-2 text-sm text-slate-500">
            {error || "Data permintaan tidak ditemukan."}
          </p>

          <div className="mt-6 flex justify-center gap-3">
            <button
              type="button"
              onClick={() => {
                setLoading(true);
                void loadData();
              }}
              className="inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            >
              <RefreshCw className="h-4 w-4" />
              Coba lagi
            </button>

            <Link
              href="/requests"
              className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <ArrowLeft className="h-4 w-4" />
              Kembali ke daftar
            </Link>
          </div>
        </div>
      </div>
    );
  }

  const requestNumber = displayValue(request.request_number);
  const patientName = displayValue(
    request.patient_name ?? request.patient_reference,
  );
  const payerName = displayValue(
    request.payer_name ?? request.insurance_name,
  );
  const procedureName = displayValue(
    request.procedure_name ?? request.service_name ?? request.service_type,
  );
  const requestStatusLabel = request.status === "payer_needs_information"
    ? "Payer meminta informasi"
    : request.status === "appeal_submitted"
      ? "Banding diajukan"
      : humanize(request.status);

  return (
    <main className="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <Link
            href="/requests"
            className="mb-3 inline-flex items-center gap-2 text-sm font-medium text-slate-500 transition hover:text-blue-700"
          >
            <ArrowLeft className="h-4 w-4" />
            Kembali ke daftar permintaan
          </Link>

          <h1 className="text-2xl font-bold tracking-tight text-slate-900">
            {currentStep.label}
          </h1>

          <div className="mt-1 flex flex-wrap items-center gap-2 text-sm text-slate-500">
            <span>{requestNumber}</span>
            {request.status && (
              <span
                className={`rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${requestStatusClass(request.status)}`}
              >
                {requestStatusLabel}
              </span>
            )}
            <span>{currentStep.description}</span>
          </div>
        </div>

        <div className="flex flex-wrap gap-2 self-start sm:self-auto">
          {activeStep === "checklist" && (
            <button
              type="button"
              onClick={() => void downloadSummary()}
              className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 transition hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
            >
              <Download className="h-4 w-4" />
              Unduh ringkasan
            </button>
          )}

          <button
            type="button"
            onClick={() => {
              setLoading(true);
              void loadData();
            }}
            disabled={loading}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <RefreshCw
              className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
            />
            Refresh data
          </button>
        </div>
      </div>

      <nav
        aria-label="Tahapan permohonan"
        className="-mx-4 overflow-x-auto border-y border-slate-200 bg-white px-4 py-3 sm:mx-0 sm:rounded-xl sm:border"
      >
        <ol className="flex min-w-max items-center gap-2 sm:min-w-0 sm:gap-3">
          {workflowSteps.map((step, index) => {
            const isActive = step.id === activeStep;
            const href = step.path
              ? `/requests/${requestId}/${step.path}`
              : `/requests/${requestId}`;

            return (
              <li key={step.id} className="flex items-center gap-2 sm:flex-1">
                <Link
                  href={href}
                  aria-current={isActive ? "step" : undefined}
                  className={`flex min-h-11 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700 ${
                    isActive
                      ? "bg-blue-700 text-white shadow-sm"
                      : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                  }`}
                >
                  <span
                    className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs ${
                      isActive
                        ? "bg-white/20 text-white"
                        : "bg-white text-slate-600"
                    }`}
                  >
                    {index + 1}
                  </span>
                  <span>{step.label}</span>
                </Link>
                {index < workflowSteps.length - 1 && (
                  <span
                    aria-hidden="true"
                    className="hidden h-px flex-1 bg-slate-300 sm:block"
                  />
                )}
              </li>
            );
          })}
        </ol>
      </nav>

      {!isKnownStep && routeSteps && (
        <div
          role="alert"
          className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900"
        >
          Tahap tidak dikenal. Pilih salah satu tahap pada navigasi di atas.
        </div>
      )}

      {error && (
        <div
          role="alert"
          className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800"
        >
          <AlertCircle className="mt-0.5 h-5 w-5 shrink-0" />
          <div className="min-w-0 flex-1">
            <p className="font-semibold">Terjadi kesalahan</p>
            <p className="mt-1 break-words">{error}</p>
          </div>
          <button
            type="button"
            aria-label="Tutup pesan error"
            onClick={() => setError("")}
            className="rounded p-1 hover:bg-red-100"
          >
            <XCircle className="h-4 w-4" />
          </button>
        </div>
      )}

      {notice && (
        <div
          role="status"
          className="flex items-start gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800"
        >
          <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0" />
          <p className="flex-1">{notice}</p>
          <button
            type="button"
            aria-label="Tutup notifikasi"
            onClick={() => setNotice("")}
            className="rounded p-1 hover:bg-emerald-100"
          >
            <XCircle className="h-4 w-4" />
          </button>
        </div>
      )}

      {request && currentUser && (
        <section aria-label="Aksi tahap permohonan" className="rounded-xl border border-slate-200 bg-white p-4 sm:p-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div><h2 className="font-semibold text-slate-900">Tahap berikutnya</h2><p className="mt-1 text-sm text-slate-600">Status saat ini: {requestStatusLabel}</p></div>
            <div className="flex flex-wrap gap-2">
              {currentUser.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") && <>
                <button disabled={busy} onClick={() => void transition("submit_for_review")} className="min-h-10 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white hover:bg-[#174742] disabled:opacity-50">{request.status === "payer_needs_information" ? "Kirim jawaban payer untuk review" : "Kirim untuk review"}</button>
                {(request.status === "draft" || request.status === "needs_information") && <button disabled={busy} onClick={() => { const note = window.prompt("Alasan membatalkan permohonan:"); if (note?.trim()) void transition("cancel", { note }); }} className="ml-1 min-h-10 rounded-lg border border-slate-300 px-4 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50">Batalkan</button>}
              </>}
              {currentUser.role === "reviewer" && request.status === "ready_for_review" && <button disabled={busy} onClick={() => void transition("start_review")} className="min-h-10 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white disabled:opacity-50">Mulai review</button>}
              {currentUser.role === "reviewer" && request.status === "in_review" && <>
                <button disabled={busy} onClick={() => { const note = window.prompt("Sebutkan informasi atau dokumen yang harus dilengkapi Staff:"); if (note?.trim()) void transition("request_information", { note }); }} className="min-h-10 rounded-lg border border-amber-400 px-4 text-sm font-semibold text-amber-900 hover:bg-amber-50 disabled:opacity-50">Minta informasi</button>
                <button disabled={busy} onClick={() => void transition("complete_review")} className="min-h-10 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white disabled:opacity-50">Selesaikan review</button>
              </>}
              {currentUser.role === "staff" && request.status === "ready_for_submission" && <button disabled={busy} onClick={() => { const submission_channel = window.prompt("Kanal yang digunakan: payer_portal, clearinghouse, phone, fax, mail, api_manual, atau other", "payer_portal") ?? ""; if (!submission_channel.trim()) return; const payer_reference = window.prompt("Referensi payer (opsional):") ?? ""; void transition("record_submission", { submission_channel, payer_reference }); }} className="min-h-10 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white disabled:opacity-50">{statusHistory.some((event) => event.action === "request_payer_information") ? "Catat pengiriman jawaban payer" : "Catat pengajuan ke payer"}</button>}
              {currentUser.role === "staff" && request.status === "submitted" && <>
                <button disabled={busy} onClick={() => { const note = window.prompt("Catat informasi atau dokumen tambahan yang diminta payer:"); if (note?.trim()) void transition("request_payer_information", { note }); }} className="min-h-10 rounded-lg border border-amber-400 px-4 text-sm font-semibold text-amber-900 hover:bg-amber-50 disabled:opacity-50">Payer meminta informasi</button>
                <button disabled={busy} onClick={() => { const payer_decision = window.prompt("Keputusan payer: approved, denied, partially_approved, atau withdrawn") ?? ""; const payer_decision_note = window.prompt("Catatan keputusan dan alasan payer:") ?? ""; if (payer_decision && payer_decision_note.trim()) void transition("record_decision", { payer_decision, payer_decision_note }); }} className="min-h-10 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white disabled:opacity-50">Catat keputusan payer</button>
              </>}
              {currentUser.role === "staff" && request.status === "completed" && ["denied", "partially_approved"].includes(String(request.payer_decision ?? "")) && <button disabled={busy} onClick={() => { const submission_channel = window.prompt("Kanal banding: payer_portal, clearinghouse, phone, fax, mail, api_manual, atau other", "payer_portal") ?? ""; if (!submission_channel.trim()) return; const note = window.prompt("Ringkasan alasan dan dokumen pendukung banding:"); if (note?.trim()) { const appeal_reference = window.prompt("Referensi banding (opsional):") ?? ""; void transition("record_appeal_submission", { submission_channel, note, appeal_reference }); } }} className="min-h-10 rounded-lg border border-amber-400 px-4 text-sm font-semibold text-amber-900 hover:bg-amber-50 disabled:opacity-50">Catat pengajuan banding</button>}
              {currentUser.role === "staff" && request.status === "appeal_submitted" && <button disabled={busy} onClick={() => { const payer_decision = window.prompt("Keputusan banding: approved, denied, partially_approved, atau withdrawn") ?? ""; const payer_decision_note = window.prompt("Catatan respons atas banding:") ?? ""; if (payer_decision && payer_decision_note.trim()) void transition("record_appeal_decision", { payer_decision, payer_decision_note }); }} className="min-h-10 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white disabled:opacity-50">Catat respons banding</button>}
            </div>
          </div>
          {currentUser.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") && !request.checklist_version_id && <p className="mt-3 rounded-lg bg-amber-50 p-3 text-sm text-amber-900">Checklist aktif untuk <strong>{String(request.payer_name ?? "skema belum diisi")}</strong> / <strong>{String(request.service_type ?? "alur belum diisi")}</strong> belum tersedia. Minta Admin meninjau sumber dan aturan checklist sebelum mengaktifkannya.</p>}
          {request.status === "needs_information" && <p className="mt-3 rounded-lg bg-amber-50 p-3 text-sm text-amber-900">Permohonan dikembalikan Reviewer. <Link className="font-semibold underline underline-offset-2" href={`/requests/${requestId}/fields`}>Buka Verifikasi field</Link> untuk memperbaiki field sesuai catatan di riwayat, lalu kirim ulang sebagai Staff.</p>}
          {request.status === "payer_needs_information" && <p className="mt-3 rounded-lg bg-amber-50 p-3 text-sm text-amber-900">Payer meminta informasi tambahan. Lengkapi dokumen atau field, lalu kirim jawaban untuk review internal sebelum disampaikan kembali ke payer.</p>}
          {(request.status === "completed" || request.status === "appeal_submitted") && <p className="mt-3 text-sm text-slate-600">Keputusan payer: {humanize(String(request.payer_decision ?? "belum dicatat"))}. {String(request.payer_decision_note ?? "")}</p>}
          {!!request.appeal_decision && <p className="mt-2 text-sm text-slate-600">Respons banding: {humanize(String(request.appeal_decision))}. {String(request.appeal_decision_note ?? "")}</p>}
        </section>
      )}

      {activeStep === "overview" && historyUnavailable && <p role="status" className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">Riwayat belum tersedia karena backend yang berjalan belum memuat endpoint workflow terbaru. Restart backend Go pada port 8081, lalu muat ulang halaman.</p>}
      {activeStep === "overview" && statusHistory.length > 0 && <section className="rounded-xl border border-slate-200 bg-white p-5"><h2 className="font-semibold text-slate-900">Riwayat permohonan</h2><ol className="mt-4 space-y-3">{statusHistory.map((event) => <li key={event.id} className="border-l-2 border-slate-200 pl-4"><p className="text-sm font-medium text-slate-800">{event.from_status ? `${humanize(event.from_status)} → ` : ""}{humanize(event.to_status)}</p><p className="mt-1 text-xs text-slate-600">{event.actor_name ?? "Akun yang dinonaktifkan"} · {formatDate(event.created_at)} · {humanize(event.action)}</p>{event.note && <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700">{event.note}</p>}</li>)}</ol></section>}

      {activeStep === "overview" && (
      <section
        id="pa-request"
        className="scroll-mt-24 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"
      >
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded-md bg-blue-50 px-2.5 py-1 text-xs font-semibold text-blue-700">
                {requestNumber}
              </span>

              {request.status && (
                <span
                  className={`rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset ${requestStatusClass(request.status)}`}
                >
                  {humanize(request.status)}
                </span>
              )}
            </div>

            <h2 className="mt-3 text-xl font-bold text-slate-900">
              {patientName}
            </h2>

            <p className="mt-1 text-sm text-slate-500">
              Ringkasan informasi permintaan otorisasi.
            </p>
          </div>

          {request.priority && (
            <div className="rounded-lg border border-slate-200 px-3 py-2">
              <p className="text-xs text-slate-500">Prioritas</p>
              <p className="mt-1 text-sm font-semibold text-slate-900">
                {humanize(request.priority)}
              </p>
            </div>
          )}
        </div>

        <div className="mt-6 grid grid-cols-1 gap-5 border-t border-slate-100 pt-5 sm:grid-cols-2 lg:grid-cols-4">
          <InfoItem
            icon={<UserRound className="h-4 w-4" />}
            label="Pasien / referensi"
            value={patientName}
          />

          <InfoItem
            icon={<Building2 className="h-4 w-4" />}
            label="Payer / Asuransi"
            value={payerName}
          />

          <InfoItem
            icon={<FileText className="h-4 w-4" />}
            label="Prosedur / Layanan"
            value={procedureName}
          />

          <InfoItem
            icon={<UserRound className="h-4 w-4" />}
            label="Ordering provider NPI"
            value={String(request.ordering_provider_npi ?? request.provider_npi ?? "Belum diisi")}
          />

          <InfoItem
            icon={<UserRound className="h-4 w-4" />}
            label="Servicing provider NPI"
            value={String(request.servicing_provider_npi ?? "Belum diisi")}
          />

          <InfoItem
            icon={<CalendarDays className="h-4 w-4" />}
            label="Tanggal dibuat"
            value={formatDate(request.created_at)}
          />
        </div>

        {request.diagnosis && (
          <div className="mt-5 rounded-xl bg-slate-50 p-4">
            <p className="text-xs font-medium text-slate-500">Diagnosis</p>
            <p className="mt-1 text-sm text-slate-800">
              {request.diagnosis}
            </p>
          </div>
        )}
        {currentUser?.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") && <form onSubmit={saveRequestDetails} className="mt-6 space-y-4 border-t border-slate-100 pt-5">
          <h3 className="font-semibold text-slate-900">Edit request information</h3>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="space-y-1 text-sm font-medium text-slate-700">Patient / member reference<input value={requestDraft.patient_reference} onChange={(event) => setRequestDraft({ ...requestDraft, patient_reference: event.target.value })} maxLength={100} className="min-h-10 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /></label>
            {([ ["payer_name", "Payer"], ["service_type", "Requested service"], ["ordering_provider_npi", "Ordering provider NPI"], ["servicing_provider_npi", "Servicing provider NPI"] ] as const).map(([key, label]) => <label key={key} className="space-y-1 text-sm font-medium text-slate-700">{label}<input value={requestDraft[key]} onChange={(event) => setRequestDraft({ ...requestDraft, [key]: key.endsWith("_npi") ? event.target.value.replace(/\D/g, "").slice(0, 10) : event.target.value })} maxLength={key.endsWith("_npi") ? 10 : 150} inputMode={key.endsWith("_npi") ? "numeric" : undefined} className="min-h-10 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /></label>)}
            <label className="space-y-1 text-sm font-medium text-slate-700 sm:col-span-2">Diagnosis / kode sebagaimana tercantum<textarea value={requestDraft.diagnosis} onChange={(event) => setRequestDraft({ ...requestDraft, diagnosis: event.target.value })} maxLength={2000} rows={3} className="w-full rounded-lg border border-slate-300 px-3 py-2 font-normal focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /></label>
          </div>
          <div className="flex justify-end"><button disabled={busy} className="min-h-10 rounded-lg border border-slate-300 px-4 text-sm font-semibold text-slate-800 hover:bg-slate-50 disabled:opacity-50">Simpan informasi</button></div>
        </form>}
      </section>
      )}

      {activeStep === "documents" && (
      <section
        id="document-review"
        className="scroll-mt-24 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"
      >
        <SectionHeading
          icon={<FileUp className="h-5 w-5" />}
          title="Upload dokumen"
          description="Unggah CSV atau PDF berbasis teks (maksimum 10 MB). PDF hasil scan/OCR belum didukung."
        />

        {currentUser?.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") && (
          <div className="mb-5 rounded-xl border border-[#b8d5ca] bg-[#f3f8f6] p-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <h3 className="text-sm font-semibold text-[#173f45]">Link upload pasien</h3>
                <p className="mt-1 max-w-2xl text-sm leading-5 text-slate-700">
                  Pasien dapat mengirim dokumen tanpa login. Link baru akan menonaktifkan link sebelumnya.
                </p>
              </div>
              <button
                type="button"
                onClick={() => void createPatientUploadLink()}
                disabled={creatingUploadLink || busy}
                className="inline-flex min-h-11 shrink-0 items-center justify-center gap-2 rounded-lg border border-[#205b55] bg-white px-4 text-sm font-semibold text-[#174742] hover:bg-[#e6f1ee] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {creatingUploadLink ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                {patientUploadLink ? "Buat link baru" : "Buat link upload"}
              </button>
            </div>
            {patientUploadLink && (
              <div className="mt-4 flex flex-col gap-2 sm:flex-row">
                <input
                  aria-label="Link halaman upload pasien"
                  readOnly
                  value={patientUploadLink}
                  className="min-h-11 min-w-0 flex-1 rounded-lg border border-slate-400 bg-white px-3 text-sm text-slate-900"
                  onFocus={(event) => event.currentTarget.select()}
                />
                <button
                  type="button"
                  onClick={() => void copyPatientUploadLink()}
                  className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg bg-[#205b55] px-4 text-sm font-semibold text-white hover:bg-[#174742]"
                >
                  <Copy className="h-4 w-4" />
                  Salin link
                </button>
              </div>
            )}
          </div>
        )}

        {currentUser?.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") && <div className="mb-5 rounded-lg bg-[#f3f8f6] p-4"><div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="text-sm font-semibold text-[#173f45]">Tidak punya dokumen untuk uji?</h3><p className="mt-1 max-w-2xl text-xs leading-5 text-slate-600">Unduh CSV contoh yang mengikuti field wajib checklist permohonan ini. Isinya data sintetis, bukan fakta medis; gunakan hanya untuk latihan.</p></div><button type="button" disabled={sampleFieldKeys.length === 0} onClick={downloadSyntheticCSV} className="min-h-10 rounded-lg border border-[#205b55] bg-white px-3 text-sm font-semibold text-[#205b55] hover:bg-[#e6f1ee] disabled:cursor-not-allowed disabled:opacity-50">Unduh CSV data uji</button></div></div>}

        {currentUser?.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") ? <form
          onSubmit={uploadDocument}
          className="space-y-4"
        >
          <label
            htmlFor="document"
            className="flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 px-5 py-8 text-center transition hover:border-blue-400 hover:bg-blue-50/40"
          >
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-white text-blue-600 shadow-sm">
              <Upload className="h-6 w-6" />
            </div>

            <span className="mt-3 text-sm font-semibold text-slate-800">
              Pilih dokumen untuk diunggah
            </span>

            <span className="mt-1 text-xs text-slate-500">
              Format CSV atau PDF, maksimum 10 MB
            </span>

            <input
              id="document"
              name="document"
              type="file"
              accept=".csv,.pdf,text/csv,application/pdf"
              required
              className="mt-4 block w-full max-w-xs text-sm text-slate-600 file:mr-3 file:rounded-lg file:border-0 file:bg-blue-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-blue-700"
            />
          </label>

          <div className="flex justify-end">
            <button
              type="submit"
              disabled={busy}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {busy ? (
                <LoaderCircle className="h-4 w-4 animate-spin" />
              ) : (
                <Upload className="h-4 w-4" />
              )}
              {busy ? "Mengunggah..." : "Upload dokumen"}
            </button>
          </div>
        </form> : <p className="mb-4 rounded-lg bg-slate-50 p-3 text-sm text-slate-600">Dokumen hanya dapat ditambah atau diproses oleh Staff pemilik saat draft atau saat melengkapi permintaan informasi.</p>}

        {documents.length > 0 && (
          <div className="mt-5 space-y-3">
            {documents.map((document) => {
              const isParsing = parsingDocumentID === document.id;
              const isPDF = document.mime_type === "application/pdf";
              const isConfirmed = confirmedPDFDocumentIDs.includes(
                document.id,
              );
              const parseFailed = document.parsing_status === "failed";

              return (
                <article
                  key={document.id}
                  className="rounded-xl border border-blue-200 bg-blue-50/60 p-4"
                >
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                    <div className="flex min-w-0 items-start gap-3">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-white text-blue-700">
                        <FileText className="h-5 w-5" />
                      </div>

                      <div className="min-w-0">
                        <p className="break-all text-sm font-semibold text-slate-900">
                          {document.original_filename ??
                            `Dokumen #${document.id}`}
                        </p>

                        <p className="mt-1 text-xs text-slate-500">
                          {formatFileSize(document.size_bytes)}
                          {" · "}
                          Status parsing:{" "}
                          {humanize(document.parsing_status ?? "pending")}
                        </p>

                        {document.created_at && (
                          <p className="mt-1 text-xs text-slate-500">
                            Diunggah: {formatDate(document.created_at)}
                          </p>
                        )}

                        {parseFailed && (
                          <p className="mt-1 text-xs text-red-700">
                            Pemrosesan gagal. Periksa format dan konfigurasi,
                            lalu coba proses ulang.
                          </p>
                        )}
                      </div>
                    </div>

                    <div className="flex flex-col gap-3 lg:items-end">
                      {isPDF && (
                        <label className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs leading-5 text-amber-900 lg:max-w-sm">
                          <input
                            type="checkbox"
                            checked={isConfirmed}
                            onChange={(event) =>
                              setConfirmedPDFDocumentIDs((current) =>
                                event.target.checked
                                  ? [...current, document.id]
                                  : current.filter(
                                      (id) => id !== document.id,
                                    ),
                              )
                            }
                            className="mt-0.5"
                          />
                          <span>
                            Saya memastikan dokumen berisi data sintetis atau
                            telah di-de-identifikasi. Teks PDF akan dikirim ke
                            OpenAI untuk ekstraksi. Jangan gunakan data pasien
                            nyata sebelum tinjauan privasi dan kontrak selesai.
                          </span>
                        </label>
                      )}

                      <button
                        type="button"
                        onClick={() => void parseDocument(document)}
                        disabled={
                          parsingDocumentID !== null || currentUser?.role !== "staff" || (request.status !== "draft" && request.status !== "needs_information" && request.status !== "payer_needs_information") ||
                          (isPDF && !isConfirmed)
                        }
                        className="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {isParsing ? (
                          <LoaderCircle className="h-4 w-4 animate-spin" />
                        ) : parseFailed ? (
                          <RefreshCw className="h-4 w-4" />
                        ) : (
                          <Play className="h-4 w-4" />
                        )}
                        {isParsing
                          ? "Memproses..."
                          : parseFailed
                            ? "Coba proses ulang"
                            : document.parsing_status === "completed"
                              ? "Proses ulang"
                              : "Parse dokumen"}
                      </button>
                      {currentUser?.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information") && <button type="button" onClick={() => void deleteDocument(document)} disabled={isParsing || parsingDocumentID !== null || deletingDocumentID !== null || document.parsing_status === "processing"} className="inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border border-red-300 px-3 text-sm font-semibold text-red-800 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"><Trash2 className="h-4 w-4" />{deletingDocumentID === document.id ? "Menghapus…" : "Hapus dokumen"}</button>}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        )}

        {documents.length === 0 && (
          <div className="mt-4 flex items-start gap-2 rounded-lg bg-slate-50 p-3 text-xs leading-5 text-slate-500">
            <CircleHelp className="mt-0.5 h-4 w-4 shrink-0" />
            <p>
              Setelah dokumen berhasil diunggah, gunakan tombol parsing pada
              dokumen tersebut untuk memproses dan menampilkan field yang
              diekstrak.
            </p>
          </div>
        )}
      </section>
      )}

      {activeStep === "fields" && (
      <section
        id="extracted-fields"
        className="scroll-mt-24 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"
      >
        <SectionHeading
          icon={<FileText className="h-5 w-5" />}
          title="Hasil ekstraksi"
          description="Tinjau nilai yang diekstrak dari dokumen. Periksa bukti sumber dan status verifikasi sebelum evaluasi checklist."
        />

        <div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 p-4">
            <p className="text-xs text-slate-500">Jumlah field</p>
            <p className="mt-1 text-2xl font-bold text-slate-900">
              {fields.length}
            </p>
          </div>

          <div className="rounded-xl border border-slate-200 p-4">
            <p className="text-xs text-slate-500">Terverifikasi</p>
            <p className="mt-1 text-2xl font-bold text-emerald-700">
              {
                fields.filter((field) =>
                  ["verified", "approved"].includes(
                    field.verification_status?.toLowerCase(),
                  ),
                ).length
              }
            </p>
          </div>

          <div className="rounded-xl border border-slate-200 p-4">
            <p className="text-xs text-slate-500">Perlu ditinjau</p>
            <p className="mt-1 text-2xl font-bold text-amber-700">
              {
                fields.filter((field) =>
                  ["needs_review", "pending"].includes(
                    field.verification_status?.toLowerCase(),
                  ),
                ).length
              }
            </p>
          </div>
        </div>

        {fields.length === 0 ? (
          <div className="rounded-xl border border-dashed border-slate-300 px-5 py-12 text-center">
            <FileText className="mx-auto h-8 w-8 text-slate-400" />

            <p className="mt-3 text-sm font-semibold text-slate-800">
              Belum ada extracted fields
            </p>

            <p className="mx-auto mt-1 max-w-md text-sm leading-6 text-slate-500">
              Unggah dan proses dokumen terlebih dahulu. Hasil ekstraksi
              akan ditampilkan di bagian ini.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {fields.map((field) => (
              <FieldEditor
                key={`${field.id}-${field.field_value ?? ""}-${field.verification_status}`}
                field={field}
                saving={savingField === field.id}
                editable={currentUser?.role === "reviewer" ? request.status === "in_review" : currentUser?.role === "staff" && (request.status === "draft" || request.status === "needs_information" || request.status === "payer_needs_information")}
                allowStatus={currentUser?.role === "reviewer" && request.status === "in_review"}
                onSave={(value, status) =>
                  updateField(field, value, status)
                }
              />
            ))}
          </div>
        )}
      </section>
      )}

      {activeStep === "checklist" && (
      <section
        id="checklist-evaluation"
        className="scroll-mt-24 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"
      >
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <SectionHeading
            icon={<ClipboardCheck className="h-5 w-5" />}
            title="Evaluasi checklist"
            description="Evaluasi persyaratan berdasarkan extracted fields yang tersedia."
          />

          <button
            type="button"
            onClick={() => void evaluateChecklist()}
            disabled={evaluating || currentUser?.role !== "reviewer" || request.status !== "in_review"}
            className="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {evaluating ? (
              <LoaderCircle className="h-4 w-4 animate-spin" />
            ) : (
              <ShieldCheck className="h-4 w-4" />
            )}
            {evaluating ? "Mengevaluasi..." : "Evaluasi checklist"}
          </button>
        </div>

        <div className="mb-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div className="rounded-xl border border-emerald-200 bg-emerald-50/60 p-4">
            <div className="flex items-center gap-2 text-emerald-700">
              <CheckCircle2 className="h-4 w-4" />
              <span className="text-sm font-medium">Fulfilled</span>
            </div>
            <p className="mt-2 text-2xl font-bold text-emerald-800">
              {fulfilledCount}
            </p>
          </div>

          <div className="rounded-xl border border-red-200 bg-red-50/60 p-4">
            <div className="flex items-center gap-2 text-red-700">
              <XCircle className="h-4 w-4" />
              <span className="text-sm font-medium">Missing</span>
            </div>
            <p className="mt-2 text-2xl font-bold text-red-800">
              {missingCount}
            </p>
          </div>

          <div className="rounded-xl border border-amber-200 bg-amber-50/60 p-4">
            <div className="flex items-center gap-2 text-amber-700">
              <Clock3 className="h-4 w-4" />
              <span className="text-sm font-medium">Needs review</span>
            </div>
            <p className="mt-2 text-2xl font-bold text-amber-800">
              {reviewCount}
            </p>
          </div>
        </div>

        {checklist.length === 0 ? (
          <div className="rounded-xl border border-dashed border-slate-300 px-5 py-12 text-center">
            <ClipboardCheck className="mx-auto h-8 w-8 text-slate-400" />

            <p className="mt-3 text-sm font-semibold text-slate-800">
              Belum ada hasil checklist
            </p>

            <p className="mx-auto mt-1 max-w-md text-sm leading-6 text-slate-500">
              Jalankan evaluasi untuk membandingkan data yang tersedia
              dengan aturan persyaratan.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {checklist.map((item, index) => {
              const title =
                displayValue(item.rule_name) !== "-"
                  ? displayValue(item.rule_name)
                  : displayValue(item.rule_key) !== "-"
                    ? displayValue(item.rule_key)
                    :                     `Checklist rule #${item.rule_id ?? item.checklist_rule_id ?? index + 1}`;

              const detail =
                displayValue(item.message) !== "-"
                  ? displayValue(item.message)
                  : displayValue(item.reason) !== "-"
                    ? displayValue(item.reason)
                    : displayValue(item.description);

              return (
                <div
                  key={
                    item.id ??
                    item.rule_id ??
                    item.checklist_rule_id ??
                    `${title}-${index}`
                  }
                  className="rounded-xl border border-slate-200 p-4"
                >
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <h3 className="break-words text-sm font-semibold text-slate-900">
                        {title}
                      </h3>

                      {detail !== "-" && (
                        <p className="mt-1.5 break-words text-sm leading-6 text-slate-600">
                          {detail}
                        </p>
                      )}

                      {item.field_key && (
                        <p className="mt-2 text-xs text-slate-500">
                          Field terkait:{" "}
                          <span className="font-medium text-slate-700">
                            {item.field_key}
                          </span>
                        </p>
                      )}
                    </div>

                    <span
                      className={`inline-flex w-fit shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset ${checklistStatusClass(item.status)}`}
                    >
                      {humanize(item.status)}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {checklist.length > 0 && (
          <p className="mt-4 text-xs leading-5 text-slate-500">
            Hasil checklist mencerminkan evaluasi terakhir. Jika extracted
            fields berubah, jalankan evaluasi kembali untuk memperbarui
            hasil.
          </p>
        )}
      </section>
      )}

      <footer className="flex flex-col gap-3 border-t border-slate-200 pt-5 sm:flex-row sm:items-center sm:justify-between">
        {previousStep ? (
          <Link
            href={
              previousStep.path
                ? `/requests/${requestId}/${previousStep.path}`
                : `/requests/${requestId}`
            }
            className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-slate-300 px-4 py-2.5 text-sm font-medium text-slate-700 transition hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
          >
            <ArrowLeft className="h-4 w-4" />
            Sebelumnya: {previousStep.label}
          </Link>
        ) : (
          <Link
            href="/requests"
            className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-slate-300 px-4 py-2.5 text-sm font-medium text-slate-700 transition hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
          >
            <ArrowLeft className="h-4 w-4" />
            Kembali ke daftar permintaan
          </Link>
        )}

        {nextStep ? (
          <Link
            href={`/requests/${requestId}/${nextStep.path}`}
            className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg bg-blue-700 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
          >
            Lanjut ke {nextStep.label}
            <ArrowRight className="h-4 w-4" />
          </Link>
        ) : (
          <p className="text-sm font-medium text-slate-600">
            Tahap terakhir dalam alur
          </p>
        )}
      </footer>
    </main>
  );
}

function FieldEditor({
  field,
  saving,
  onSave,
  editable,
  allowStatus,
}: {
  field: ExtractedField;
  saving: boolean;
  onSave: (value: string, status: string) => Promise<void>;
  editable: boolean;
  allowStatus: boolean;
}) {
  const [value, setValue] = useState(field.field_value ?? "");
  const [verificationStatus, setVerificationStatus] = useState(
    field.verification_status || "pending",
  );

  const hasChanges =
    value !== (field.field_value ?? "") ||
    verificationStatus !== (field.verification_status || "pending");

  return (
    <article className="rounded-xl border border-slate-200 p-4 transition hover:border-slate-300">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h3 className="break-words text-sm font-semibold text-slate-900">
            {humanize(field.field_key)}
          </h3>

          <p className="mt-1 break-all text-xs text-slate-500">
            Key: {field.field_key}
          </p>
        </div>

        <span
          className={`inline-flex w-fit shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-medium ${verificationStatusClass(field.verification_status)}`}
        >
          {humanize(field.verification_status)}
        </span>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <label
            htmlFor={`field-value-${field.id}`}
            className="mb-1.5 block text-xs font-medium text-slate-600"
          >
            Field value
          </label>

          <textarea
            id={`field-value-${field.id}`}
            value={value}
            onChange={(event) => setValue(event.target.value)}
            rows={2}
            disabled={!editable}
            className="w-full resize-y rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            placeholder="Masukkan atau perbaiki nilai field..."
          />
        </div>

        {allowStatus && <div>
          <label
            htmlFor={`field-status-${field.id}`}
            className="mb-1.5 block text-xs font-medium text-slate-600"
          >
            Verification status
          </label>

          <select
            id={`field-status-${field.id}`}
            value={verificationStatus}
            onChange={(event) =>
              setVerificationStatus(event.target.value)
            }
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
          >
            <option value="pending">Pending review</option>
            <option value="verified">Verified</option>
            <option value="corrected">Corrected</option>
            <option value="uncertain">Uncertain</option>
          </select>
        </div>}
      </div>

      {(field.evidence_text ||
        field.source_page !== null && field.source_page !== undefined ||
        field.confidence !== null && field.confidence !== undefined) && (
        <div className="mt-4 rounded-lg bg-slate-50 p-3">
          <p className="text-xs font-semibold text-slate-700">
            Extraction evidence
          </p>

          {field.evidence_text && (
            <p className="mt-1.5 whitespace-pre-wrap break-words text-sm leading-6 text-slate-600">
              {field.evidence_text}
            </p>
          )}

          <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
            {field.source_page !== null &&
              field.source_page !== undefined && (
                <span>Source page: {field.source_page}</span>
              )}

            {field.confidence !== null &&
              field.confidence !== undefined && (
                <span>
                  Confidence:{" "}
                  {field.confidence <= 1
                    ? `${(field.confidence * 100).toFixed(1)}%`
                    : `${field.confidence.toFixed(1)}%`}
                </span>
              )}
          </div>
        </div>
      )}

      {editable && <div className="mt-4 flex flex-col gap-3 border-t border-slate-100 pt-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-slate-500">
          {hasChanges
            ? "Ada perubahan yang belum disimpan."
            : "Tidak ada perubahan yang belum disimpan."}
        </p>

        <button
          type="button"
          disabled={saving || !hasChanges}
          onClick={() => void onSave(value, verificationStatus)}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? (
            <LoaderCircle className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          {saving ? "Menyimpan..." : "Simpan perubahan"}
        </button>
      </div>}
    </article>
  );
}
