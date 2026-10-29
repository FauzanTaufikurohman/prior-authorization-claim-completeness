"use client";

import { useEffect, useState, useSyncExternalStore } from "react";
import axios from "axios";
import api from "@/lib/api";

function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { error?: string; message?: string } | undefined;
    if (data?.error) return data.error;
    if (data?.message) return data.message;
    if (!error.response) return "Server tidak dapat dihubungi. Periksa koneksi lalu coba lagi.";
  }
  if (error instanceof Error) return error.message;
  return "Terjadi kesalahan saat mengirim dokumen.";
}

function subscribeToLocationHash(onChange: () => void) {
  window.addEventListener("hashchange", onChange);
  return () => window.removeEventListener("hashchange", onChange);
}

function getLocationHash(): string | null {
  return window.location.hash.slice(1);
}

function getServerLocationHash(): string | null {
  return null;
}

export default function PatientUploadPage() {
  const token = useSyncExternalStore(
    subscribeToLocationHash,
    getLocationHash,
    getServerLocationHash,
  );
  const [linkCheck, setLinkCheck] = useState<{
    token: string;
    available: boolean;
    error: string;
  } | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [formError, setFormError] = useState("");

  useEffect(() => {
    if (!token) return;
    let active = true;
    api.get("/public/pa-upload", {
      headers: { "X-PA-Upload-Token": token },
    })
      .then(() => {
        if (active) setLinkCheck({ token, available: true, error: "" });
      })
      .catch((cause: unknown) => {
        if (!active) return;
        if (axios.isAxiosError(cause) && cause.response?.status === 404) {
          setLinkCheck({
            token,
            available: false,
            error: "Link upload tidak ditemukan atau sudah tidak aktif. Hubungi staf yang mengirimkan link.",
          });
          return;
        }
        setLinkCheck({
          token,
          available: false,
          error: getErrorMessage(cause),
        });
      });

    return () => {
      active = false;
    };
  }, [token]);

  const currentLinkCheck = token && linkCheck?.token === token ? linkCheck : null;
  const checkingLink = token === null || (token !== "" && currentLinkCheck === null);
  const linkAvailable = currentLinkCheck?.available ?? false;
  const linkError = currentLinkCheck?.error
    ?? (token === "" ? "Link upload tidak lengkap. Hubungi staf yang mengirimkan link." : "");

  async function submitDocument(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");

    const form = event.currentTarget;
    const fileInput = form.elements.namedItem("file");
    if (!(fileInput instanceof HTMLInputElement) || !fileInput.files?.[0]) {
      setFormError("Pilih dokumen yang akan dikirim.");
      return;
    }

    const file = fileInput.files[0];
    const fileName = file.name.toLowerCase();
    if (!fileName.endsWith(".csv") && !fileName.endsWith(".pdf")) {
      setFormError("Format dokumen belum didukung. Pilih file CSV atau PDF.");
      return;
    }
    if (file.size === 0 || file.size > 10 * 1024 * 1024) {
      setFormError("Ukuran file harus lebih dari 0 dan tidak melebihi 10 MB.");
      return;
    }

    setSubmitting(true);
    try {
      await api.post(
        "/public/pa-upload/documents",
        new FormData(form),
        {
          timeout: 30_000,
          headers: { "X-PA-Upload-Token": token },
        },
      );
      fileInput.value = "";
      setSubmitted(true);
    } catch (cause) {
      setFormError(getErrorMessage(cause));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen flex-1 bg-[#f3f6f3] px-4 py-8 sm:px-6 sm:py-12">
      <div className="mx-auto w-full max-w-xl">
        <header className="mb-7 flex items-center gap-3">
          <div>
            <p className="text-lg font-semibold leading-6 text-[#07171d]">Ayuprama</p>
            <p className="text-sm text-[#354e4a]">Pengiriman dokumen PA</p>
          </div>
        </header>

        <section className="rounded-2xl border border-[#c7d7d0] bg-white p-5 sm:p-8">
          <h1 className="text-2xl font-semibold leading-tight text-[#07171d] sm:text-3xl">
            Unggah dokumen
          </h1>
          <p className="mt-3 text-sm leading-6 text-[#354e4a]">
            Kirim dokumen yang diminta petugas melalui link ini. Anda tidak perlu login.
            Setelah dokumen diterima, petugas akan mengelola proses selanjutnya.
          </p>

          {checkingLink && (
            <p role="status" className="mt-6 rounded-lg bg-[#edf4f0] p-4 text-sm text-[#183336]">
              Memeriksa link upload...
            </p>
          )}

          {!checkingLink && !linkAvailable && (
            <div className="mt-6 rounded-lg border border-[#9d4b37] bg-[#fff4f0] p-4">
              <p role="alert" className="text-sm leading-6 text-[#692c1e]">
                {linkError || "Link upload tidak tersedia. Hubungi staf yang mengirimkan link."}
              </p>
              {token && (
                <button
                  type="button"
                  onClick={() => window.location.reload()}
                  className="mt-3 inline-flex min-h-11 items-center rounded-lg border border-[#9d4b37] bg-white px-4 text-sm font-semibold text-[#692c1e] hover:bg-[#fffaf8]"
                >
                  Periksa link lagi
                </button>
              )}
            </div>
          )}

          {linkAvailable && submitted && (
            <div className="mt-6 rounded-lg border border-[#205b55] bg-[#edf6f1] p-4">
              <p role="status" className="text-sm font-semibold text-[#174742]">
                Dokumen berhasil dikirim. Petugas akan memeriksa dan menindaklanjuti.
              </p>
              <button
                type="button"
                onClick={() => setSubmitted(false)}
                className="mt-4 inline-flex min-h-11 items-center justify-center rounded-lg border border-[#205b55] bg-white px-4 text-sm font-semibold text-[#174742] hover:bg-[#f3f8f6]"
              >
                Unggah dokumen lain
              </button>
            </div>
          )}

          {linkAvailable && !submitted && (
            <form onSubmit={submitDocument} className="mt-6 space-y-5">
              <label className="block text-sm font-semibold text-[#183336]">
                Nomor WhatsApp
                <input
                  type="tel"
                  name="whatsapp_number"
                  autoComplete="tel"
                  inputMode="tel"
                  required
                  placeholder="08XXXXXXXXXX atau +62 8XX..."
                  className="mt-2 min-h-12 w-full rounded-lg border border-[#657a73] bg-white px-3 text-base font-normal text-[#07171d] placeholder:text-[#53645f]"
                />
              </label>
              <p className="-mt-3 text-sm leading-5 text-[#354e4a]">
                Masukkan nomor aktif Anda. Nomor ini hanya digunakan untuk notifikasi perubahan status permohonan PA melalui WhatsApp, bukan dibuat otomatis.
              </p>

              <label className="flex min-h-11 items-start gap-3 text-sm leading-5 text-[#183336]">
                <input
                  type="checkbox"
                  name="whatsapp_consent"
                  value="true"
                  required
                  className="mt-0.5 h-5 w-5 shrink-0 accent-[#205b55]"
                />
                <span>Saya setuju menerima notifikasi WhatsApp terkait status permohonan PA ini.</span>
              </label>

              <label className="block text-sm font-semibold text-[#183336]">
                Dokumen
                <input
                  type="file"
                  name="file"
                  accept=".csv,.pdf,text/csv,application/pdf"
                  required
                  className="mt-2 block min-h-12 w-full rounded-lg border border-[#657a73] bg-white p-2 text-sm text-[#07171d] file:mr-3 file:min-h-9 file:rounded-md file:border-0 file:bg-[#205b55] file:px-3 file:text-sm file:font-semibold file:text-white"
                />
                <span className="mt-2 block text-sm font-normal leading-5 text-[#354e4a]">
                  Format CSV atau PDF, maksimum 10 MB per dokumen.
                </span>
              </label>

              {formError && (
                <p role="alert" className="rounded-lg border border-[#9d4b37] bg-[#fff4f0] p-3 text-sm leading-5 text-[#692c1e]">
                  {formError}
                </p>
              )}

              <button
                type="submit"
                disabled={submitting}
                className="inline-flex min-h-12 w-full items-center justify-center rounded-lg bg-[#205b55] px-5 text-sm font-semibold text-white hover:bg-[#174742] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {submitting ? "Mengirim dokumen..." : "Kirim dokumen"}
              </button>
            </form>
          )}
        </section>
      </div>
    </main>
  );
}
