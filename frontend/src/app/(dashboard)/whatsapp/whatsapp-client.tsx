"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import Image from "next/image";
import { MessageCircle, QrCode, RefreshCw, Smartphone, Unlink } from "lucide-react";
import { isAxiosError } from "axios";
import api from "@/lib/api";

type Pairing = {
  id: number;
  device_number: string;
  status: "pending" | "connected" | "disconnected";
};

type PairingResponse = {
  data: Pairing | null;
};

type QRResponse = {
  data: Pairing & { qr_code: string };
};

type PairingErrorResponse = {
  error?: string;
  data?: Pairing;
};

type IntakeActivity = {
  id: number;
  sender_masked: string;
  status: string;
  last_ai_status: "processed" | "failed" | "";
  last_ai_error_code: string;
  last_message_at: string | null;
  last_ai_attempt_at: string | null;
  pa_request_number: string;
};

type IntakeActivityResponse = {
  data: IntakeActivity[];
};

function statusLabel(status: Pairing["status"]): string {
  if (status === "connected") return "Terhubung";
  if (status === "disconnected") return "Terputus";
  return "Menunggu scan";
}

function intakeStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    awaiting_consent: "Menunggu persetujuan",
    collecting: "Mengumpulkan data PA",
    awaiting_confirmation: "Menunggu konfirmasi data",
    completed: "Draft PA dibuat",
    cancelled: "Pengumpulan dibatalkan",
  };
  return labels[status] ?? "Status percakapan tidak dikenal";
}

function activityDate(value: string | null): string {
  if (!value) return "Belum ada";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Waktu tidak tersedia";
  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function aiErrorMessage(code: string): string {
  const messages: Record<string, string> = {
    missing_api_key: "OPENAI_API_KEY belum dikonfigurasi pada backend.",
    timeout: "OpenAI tidak merespons sampai batas waktu. Coba lagi.",
    authentication: "OpenAI menolak API key atau akses project. Periksa konfigurasi key.",
    model_not_found: "Model OpenAI tidak ditemukan atau belum tersedia untuk project.",
    rate_limited: "Batas permintaan atau kredit OpenAI tercapai. Periksa kuota dan billing.",
    request_rejected: "OpenAI menolak permintaan. Periksa nama model dan akses project.",
    provider_error: "Layanan OpenAI sedang bermasalah. Coba lagi beberapa saat lagi.",
    request_failed: "Permintaan OpenAI gagal. Periksa konfigurasi model.",
    connection_error: "Backend tidak dapat terhubung ke OpenAI. Periksa koneksi server.",
    invalid_response: "Respons OpenAI tidak dapat dibaca. Coba lagi atau periksa konfigurasi model.",
  };
  return messages[code] ?? "Periksa konfigurasi dan koneksi OpenAI pada backend.";
}

function qrImageSource(value: string): string {
  if (value.startsWith("data:image/") || value.startsWith("https://") || value.startsWith("http://")) {
    return value;
  }
  return `data:image/png;base64,${value}`;
}

export default function WhatsAppPairingClient() {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [phoneNumber, setPhoneNumber] = useState("");
  const [qrCode, setQrCode] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [checkingStatus, setCheckingStatus] = useState(false);
  const [configuringWebhook, setConfiguringWebhook] = useState(false);
  const [deletingPairing, setDeletingPairing] = useState(false);
  const [error, setError] = useState("");
  const [statusError, setStatusError] = useState("");
  const [activity, setActivity] = useState<IntakeActivity[]>([]);
  const [activityLoading, setActivityLoading] = useState(true);
  const [activityRefreshing, setActivityRefreshing] = useState(false);
  const [activityError, setActivityError] = useState("");

  const loadActivity = useCallback(async () => {
    try {
      const response = await api.get<IntakeActivityResponse>("/whatsapp/intake-activity");
      setActivity(response.data.data);
      setActivityError("");
    } catch (requestError) {
      const message =
        isAxiosError<PairingErrorResponse>(requestError) && requestError.response?.data?.error
          ? requestError.response.data.error
          : "Aktivitas chatbot tidak dapat dimuat. Periksa koneksi lalu coba lagi.";
      setActivityError(message);
    } finally {
      setActivityLoading(false);
      setActivityRefreshing(false);
    }
  }, []);

  useEffect(() => {
    let active = true;

    async function loadPairing() {
      try {
        const response = await api.get<PairingResponse>("/whatsapp/device-pairing");
        if (!active) return;
        const currentPairing = response.data.data;
        setPairing(currentPairing);
        if (currentPairing) setPhoneNumber(currentPairing.device_number);
      } catch {
        if (active) setError("Status pairing tidak dapat dimuat. Periksa koneksi lalu muat ulang halaman.");
      } finally {
        if (active) setLoading(false);
      }
    }

    void loadPairing();
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const initialLoad = window.setTimeout(() => void loadActivity(), 0);
    const interval = window.setInterval(() => void loadActivity(), 15000);
    return () => {
      window.clearTimeout(initialLoad);
      window.clearInterval(interval);
    };
  }, [loadActivity]);

  async function refreshActivity() {
    setActivityRefreshing(true);
    await loadActivity();
  }

  const checkStatus = useCallback(async (pairingId: number) => {
    const response = await api.get<{ data: Pairing }>(
      `/whatsapp/device-pairing/${pairingId}/status`,
    );
    setPairing(response.data.data);
    setStatusError("");
    if (response.data.data.status === "connected") setQrCode("");
  }, []);

  const pairingId = pairing?.id;
  const pairingStatus = pairing?.status;

  useEffect(() => {
    if (!pairingId || pairingStatus === "connected") return;

    let active = true;
    const refreshStatus = async () => {
      try {
        const response = await api.get<{ data: Pairing }>(
          `/whatsapp/device-pairing/${pairingId}/status`,
        );
        if (!active) return;
        setPairing(response.data.data);
        setStatusError("");
        if (response.data.data.status === "connected") setQrCode("");
      } catch {
        if (active) setStatusError("Status belum dapat diperiksa. Coba muat ulang status.");
      }
    };

    void refreshStatus();
    const interval = window.setInterval(() => void refreshStatus(), 10000);
    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [pairingId, pairingStatus]);

  async function createPairing(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreating(true);
    setError("");
    setStatusError("");
    try {
      const response = await api.post<QRResponse>("/whatsapp/device-pairing", {
        device_number: phoneNumber,
        confirm_create: confirmed,
      });
      setPairing(response.data.data);
      setPhoneNumber(response.data.data.device_number);
      setQrCode(response.data.data.qr_code);
    } catch (requestError) {
      let message = "QR belum dapat dibuat. Periksa koneksi dan pengaturan Fonnte.";
      if (isAxiosError<PairingErrorResponse>(requestError)) {
        const responseData = requestError.response?.data;
        if (responseData?.data) {
          setPairing(responseData.data);
          setPhoneNumber(responseData.data.device_number);
        }
        if (responseData?.error) message = responseData.error;
      }
      setError(message);
    } finally {
      setCreating(false);
    }
  }

  async function refreshQRCode() {
    setRefreshing(true);
    setError("");
    try {
      const response = await api.post<QRResponse>("/whatsapp/device-pairing/qr");
      setPairing(response.data.data);
      setQrCode(response.data.data.qr_code);
    } catch (requestError) {
      let message = "QR belum dapat dimuat. Periksa koneksi device Fonnte lalu coba lagi.";
      if (isAxiosError<PairingErrorResponse>(requestError) && requestError.response?.data?.error) {
        message = requestError.response.data.error;
      }
      setError(message);
    } finally {
      setRefreshing(false);
    }
  }

  async function refreshStatus() {
    if (!pairing) return;
    setCheckingStatus(true);
    try {
      await checkStatus(pairing.id);
    } catch {
      setStatusError("Status belum dapat diperiksa. Coba lagi beberapa saat.");
    } finally {
      setCheckingStatus(false);
    }
  }

  async function configureWebhook() {
    setConfiguringWebhook(true);
    setStatusError("");
    try {
      await api.post("/whatsapp/device-pairing/webhook");
      setStatusError("Webhook Fonnte berhasil diperbarui. Kirim pesan WhatsApp baru untuk menguji chatbot.");
    } catch (requestError) {
      const message =
        isAxiosError<PairingErrorResponse>(requestError) && requestError.response?.data?.error
          ? requestError.response.data.error
          : "Webhook Fonnte tidak dapat diperbarui. Periksa koneksi backend dan akun Fonnte.";
      setStatusError(message);
    } finally {
      setConfiguringWebhook(false);
    }
  }

  async function deletePairing() {
    const confirmedDelete = window.confirm(
      "Hapus catatan pairing dari aplikasi? Device di akun Fonnte tidak akan dihapus.",
    );
    if (!confirmedDelete) return;

    setDeletingPairing(true);
    setError("");
    try {
      await api.delete("/whatsapp/device-pairing");
      setPairing(null);
      setQrCode("");
      setPhoneNumber("");
      setConfirmed(false);
      setStatusError("");
    } catch (requestError) {
      let message = "Catatan pairing belum dapat dihapus. Coba lagi.";
      if (isAxiosError<PairingErrorResponse>(requestError)) {
        const response = requestError.response;
        if (response?.data?.error) {
          message = response.data.error;
        } else if (response?.status === 404) {
          message = "Endpoint penghapusan belum tersedia. Restart backend lalu muat ulang halaman.";
        } else if (response?.status === 401 || response?.status === 403) {
          message = "Sesi atau izin akun tidak cukup untuk menghapus catatan pairing.";
        } else if (response) {
          message = `Backend gagal menghapus catatan pairing (HTTP ${response.status}). Periksa log backend.`;
        } else {
          message = "Backend tidak dapat dihubungi. Periksa koneksi lalu restart backend jika baru diperbarui.";
        }
      }
      setError(message);
    } finally {
      setDeletingPairing(false);
    }
  }

  return (
    <main className="min-h-screen bg-[#f5f7f6] text-slate-900">
      <div className="mx-auto max-w-6xl space-y-6 px-5 py-7 sm:px-8 sm:py-9">
        <header className="border-b border-slate-200 pb-6">
          <p className="text-sm font-medium text-[#205b55]">Intake WhatsApp</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight text-slate-950 sm:text-3xl">
            Hubungkan WhatsApp Staff
          </h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-700">
            Pasangkan nomor WhatsApp Staff agar chatbot dapat mengumpulkan data awal permohonan PA.
            Staff tetap dapat membuat dan mengubah permohonan secara manual.
          </p>
        </header>

        {error && (
          <div
            role="alert"
            className="rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-900"
          >
            {error}
          </div>
        )}

        {loading ? (
          <section className="rounded-xl border border-[#dce7e3] bg-white p-6">
            <p role="status" className="text-sm text-slate-700">
              Memuat status koneksi WhatsApp...
            </p>
          </section>
        ) : (
          <div className="grid gap-5 lg:grid-cols-[minmax(0,0.9fr)_minmax(20rem,1.1fr)]">
            <section className="rounded-xl border border-[#dce7e3] bg-white p-5 sm:p-6">
              <div className="flex items-start gap-3">
                <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-[#e8f1ef] text-[#205b55]">
                  <Smartphone size={21} aria-hidden="true" />
                </span>
                <div>
                  <h2 className="text-lg font-semibold text-slate-950">Nomor WhatsApp Staff</h2>
                  <p className="mt-1 text-sm leading-6 text-slate-700">
                    Nomor akan ditambahkan atau dicocokkan sebagai device pada akun Fonnte.
                  </p>
                </div>
              </div>

              {pairing ? (
                <div className="mt-6 space-y-4">
                  <div className="rounded-lg border border-slate-300 bg-[#f5f7f6] p-4">
                    <p className="text-sm text-slate-700">Nomor yang dipasangkan</p>
                    <p className="mt-1 font-semibold text-slate-950">{pairing.device_number}</p>
                    <p className="mt-3 text-sm text-slate-700">
                      Status: <span className="font-semibold">{statusLabel(pairing.status)}</span>
                    </p>
                  </div>
                  {statusError && (
                    <p role="status" className="text-sm text-amber-900">
                      {statusError}
                    </p>
                  )}
                  <div className="flex flex-wrap gap-3">
                    <button
                      type="button"
                      onClick={refreshQRCode}
                      disabled={refreshing || pairing.status === "connected"}
                      className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-[#205b55] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#174742] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      <QrCode size={17} aria-hidden="true" />
                      {refreshing ? "Memuat QR..." : "Tampilkan QR lagi"}
                    </button>
                    <button
                      type="button"
                      onClick={refreshStatus}
                      disabled={checkingStatus}
                      className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-slate-500 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 transition hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:cursor-wait disabled:opacity-60"
                    >
                      <RefreshCw size={16} aria-hidden="true" />
                      {checkingStatus ? "Memeriksa..." : "Periksa status"}
                    </button>
                    <button
                      type="button"
                      onClick={configureWebhook}
                      disabled={configuringWebhook}
                      className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-slate-500 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 transition hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:cursor-wait disabled:opacity-60"
                    >
                      <RefreshCw size={16} aria-hidden="true" />
                      {configuringWebhook ? "Memperbarui..." : "Perbarui webhook Fonnte"}
                    </button>
                  </div>
                  <p className="text-sm leading-6 text-slate-700">
                    Jika device sudah dihapus dari Fonnte atau nomor ini salah, hapus catatan pairing di aplikasi
                    sebelum memasangkan nomor Staff yang benar. Tindakan ini tidak menghapus device Fonnte.
                  </p>
                  <button
                    type="button"
                    onClick={deletePairing}
                    disabled={deletingPairing}
                    className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-red-700 bg-white px-4 py-2.5 text-sm font-semibold text-red-800 transition hover:bg-red-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-700 disabled:cursor-wait disabled:opacity-60"
                  >
                    <Unlink size={17} aria-hidden="true" />
                    {deletingPairing ? "Menghapus catatan..." : "Hapus catatan pairing"}
                  </button>
                  {pairing.status === "connected" && (
                    <p role="status" className="text-sm leading-6 text-[#174742]">
                      WhatsApp terhubung. Pastikan tunnel publik menuju backend tetap berjalan dan tekan
                      Perbarui webhook Fonnte setelah URL tunnel berubah.
                    </p>
                  )}
                </div>
              ) : (
                <form onSubmit={createPairing} className="mt-6 space-y-5">
                  <div>
                    <label htmlFor="staff-whatsapp-number" className="mb-2 block text-sm font-medium text-slate-900">
                      Nomor WhatsApp
                    </label>
                    <input
                      id="staff-whatsapp-number"
                      type="tel"
                      autoComplete="tel"
                      inputMode="tel"
                      value={phoneNumber}
                      onChange={(event) => setPhoneNumber(event.target.value)}
                      placeholder="08xxxxxxxxxx"
                      required
                      className="min-h-11 w-full rounded-lg border border-slate-500 bg-white px-3 py-2 text-base text-slate-950 placeholder:text-slate-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700"
                    />
                    <p className="mt-2 text-sm text-slate-700">
                      Gunakan nomor WhatsApp Staff yang akan terhubung ke Fonnte.
                    </p>
                  </div>

                  <div className="rounded-lg border border-amber-400 bg-amber-50 p-4 text-sm leading-6 text-amber-950">
                    Fonnte dapat menambahkan device baru. Batas device dan biaya mengikuti paket akun Fonnte.
                  </div>

                  <label className="flex min-h-11 items-start gap-3 text-sm leading-6 text-slate-800">
                    <input
                      type="checkbox"
                      checked={confirmed}
                      onChange={(event) => setConfirmed(event.target.checked)}
                      className="mt-1 h-5 w-5 shrink-0 border-2 border-slate-500 accent-[#205b55] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700"
                    />
                    Saya memahami Fonnte akan menambahkan atau memakai device untuk nomor ini.
                  </label>

                  <button
                    type="submit"
                    disabled={creating || !confirmed || phoneNumber.trim() === ""}
                    className="inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg bg-[#205b55] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#174742] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:cursor-not-allowed disabled:opacity-60 sm:w-auto"
                  >
                    <QrCode size={17} aria-hidden="true" />
                    {creating ? "Menyiapkan device..." : "Buat QR pairing"}
                  </button>
                </form>
              )}
            </section>

            <section className="rounded-xl border border-[#dce7e3] bg-white p-5 sm:p-6">
              <div className="flex items-start gap-3">
                <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-[#e8f1ef] text-[#205b55]">
                  <QrCode size={21} aria-hidden="true" />
                </span>
                <div>
                  <h2 className="text-lg font-semibold text-slate-950">Scan untuk memasangkan</h2>
                  <p className="mt-1 text-sm leading-6 text-slate-700">
                    Buka WhatsApp di ponsel Staff, pilih Perangkat tertaut, lalu Tautkan perangkat.
                  </p>
                </div>
              </div>

              <div className="mt-6 flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed border-slate-400 bg-[#f8faf8] p-5 text-center">
                {qrCode ? (
                  <>
                    <Image
                      src={qrImageSource(qrCode)}
                      alt="Kode QR pairing WhatsApp Fonnte"
                      width={224}
                      height={224}
                      unoptimized
                      className="h-56 w-56 max-w-full object-contain"
                    />
                    <p className="mt-4 max-w-sm text-sm text-slate-700">
                      Pindai QR dari menu Perangkat tertaut di WhatsApp Staff.
                    </p>
                  </>
                ) : pairing?.status === "connected" ? (
                  <p role="status" className="max-w-sm text-sm font-medium text-[#174742]">
                    Device sudah terhubung. QR tidak lagi diperlukan.
                  </p>
                ) : (
                  <div className="max-w-sm">
                    <p className="text-sm font-medium text-slate-900">
                      QR akan tampil setelah device disiapkan.
                    </p>
                    <p className="mt-2 text-sm leading-6 text-slate-700">
                      Masukkan nomor dan konfirmasi penambahan device untuk membuat QR.
                    </p>
                  </div>
                )}

                <section
                  aria-labelledby="chatbot-activity-title"
                  className="rounded-xl border border-[#dce7e3] bg-white p-5 sm:p-6"
                >
                  <div className="flex flex-col gap-4 border-b border-slate-200 pb-4 sm:flex-row sm:items-start sm:justify-between">
                    <div className="flex items-start gap-3">
                      <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-[#e8f1ef] text-[#205b55]">
                        <MessageCircle size={20} aria-hidden="true" />
                      </span>
                      <div>
                        <h2 id="chatbot-activity-title" className="text-lg font-semibold text-slate-950">
                          Aktivitas chatbot
                        </h2>
                        <p className="mt-1 text-sm leading-6 text-slate-700">
                          Status percakapan diperbarui otomatis. Isi pesan dan nomor lengkap tidak ditampilkan.
                        </p>
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={refreshActivity}
                      disabled={activityRefreshing}
                      className="inline-flex min-h-11 items-center justify-center gap-2 self-start rounded-lg border border-slate-500 bg-white px-4 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-700 disabled:cursor-wait disabled:opacity-60"
                    >
                      <RefreshCw size={16} aria-hidden="true" />
                      {activityRefreshing ? "Memuat..." : "Muat ulang aktivitas"}
                    </button>
                  </div>

                  {activityLoading ? (
                    <p role="status" className="py-6 text-sm text-slate-700">
                      Memuat aktivitas chatbot...
                    </p>
                  ) : activityError ? (
                    <div role="alert" className="mt-5 rounded-lg border border-red-300 bg-red-50 p-4">
                      <p className="text-sm text-red-900">{activityError}</p>
                    </div>
                  ) : activity.length === 0 ? (
                    <p className="py-6 text-sm leading-6 text-slate-700">
                      Belum ada pesan webhook yang tersimpan.
                    </p>
                  ) : (
                    <ul className="divide-y divide-slate-200">
                      {activity.map((item) => (
                        <li key={item.id} className="grid gap-3 py-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                          <div className="min-w-0">
                            <p className="font-semibold text-slate-950">Pengirim {item.sender_masked}</p>
                            <p className="mt-1 text-sm text-slate-800">{intakeStatusLabel(item.status)}</p>
                            {item.pa_request_number && (
                              <p className="mt-1 text-sm font-medium text-[#205b55]">
                                Draft: {item.pa_request_number}
                              </p>
                            )}
                          </div>
                          <div className="min-w-0 text-sm leading-6 text-slate-700 sm:text-right">
                            <p aria-live="polite">
                              {item.last_ai_status === "processed"
                                ? "Pemrosesan AI terakhir berhasil"
                                : item.last_ai_status === "failed"
                                  ? "Pemrosesan AI terakhir gagal"
                                  : "AI belum pernah memproses jawaban"}
                            </p>
                            {item.last_ai_status === "failed" && (
                              <p className="text-sm text-red-800">
                                {aiErrorMessage(item.last_ai_error_code)}
                              </p>
                            )}
                            <p>Pesan terakhir: {activityDate(item.last_message_at)}</p>
                            {item.last_ai_attempt_at && (
                              <p>
                                Pemrosesan AI terakhir: {activityDate(item.last_ai_attempt_at)}
                              </p>
                            )}
                          </div>
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
              </div>
            </section>
          </div>
        )}
      </div>
    </main>
  );
}
