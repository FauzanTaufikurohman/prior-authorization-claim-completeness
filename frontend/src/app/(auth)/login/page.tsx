"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { LockKeyhole, Mail } from "lucide-react";
import api from "@/lib/api";
import { saveSession, type LoginResponse } from "@/lib/auth";
import { BrandLogo } from "@/components/brand-logo";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      const response = await api.post<LoginResponse>("/auth/login", {
        email: email.trim(),
        password,
      });

      saveSession(response.data);
      router.replace("/dashboard");
    } catch (err: unknown) {
      if (typeof err === "object" && err !== null && "response" in err) {
        const response = (
          err as {
            response?: {
              data?: { error?: string };
              status?: number;
            };
          }
        ).response;

        setError(
          response?.data?.error ??
            (response?.status === 401
              ? "Email atau kata sandi salah."
              : "Login gagal. Periksa koneksi dan coba kembali."),
        );
      } else {
        setError("Tidak dapat menghubungi server. Coba kembali.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="grid min-h-screen bg-[#f5f7f6] lg:grid-cols-2">
      <section className="relative isolate flex h-60 flex-col justify-between overflow-hidden bg-[#143b3c] p-6 text-white sm:h-72 sm:p-8 lg:sticky lg:top-0 lg:h-screen lg:p-12">
        <Image
          src="/images/hospital-login-photo.png"
          alt=""
          fill
          priority
          sizes="(min-width: 1024px) 50vw, 100vw"
          className="object-cover"
        />
        <div
          aria-hidden="true"
          className="absolute inset-0 bg-gradient-to-b from-[#102f30]/65 via-[#143b3c]/45 to-[#0d302d]/90"
        />

        <div className="relative z-10 flex items-center gap-3">
          <BrandLogo alt="" width={48} height={40} priority className="shrink-0 rounded-lg bg-white p-1 object-contain" />
          <div>
            <p className="text-lg font-semibold">Ayuprama</p>
            <p className="text-sm text-teal-100/85">Ruang kerja PA dan kelengkapan klaim</p>
          </div>
        </div>

        <div className="relative z-10 max-w-lg">
          <p className="mb-2 text-xs font-semibold uppercase tracking-[0.1em] text-[#a9dfbf] sm:mb-3 sm:text-sm">
            PRIOR AUTHORIZATION DAN KELENGKAPAN KLAIM
          </p>
          <h1 className="text-xl font-semibold leading-tight sm:text-2xl lg:text-4xl">
            Siapkan permohonan dan tinjau dokumen pendukung.
          </h1>
          <p className="mt-4 hidden max-w-md leading-7 text-teal-50/90 lg:block">
            Catat informasi permohonan, susun bukti pendukung, dan ikuti status tinjauan.
          </p>
        </div>

        <p className="relative z-10 hidden text-sm text-teal-100/80 lg:block">
          Keputusan tetap berada pada reviewer dan payer.
        </p>
      </section>

      <section className="flex items-center justify-center px-6 py-10 sm:px-8 sm:py-12 lg:min-h-screen">
        <div className="w-full max-w-md">
          <p className="text-sm font-medium text-blue-700">SELAMAT DATANG DI AYUPRAMA</p>
          <h2 className="mt-2 text-3xl font-semibold tracking-tight text-slate-900">
            Masuk ke akun
          </h2>
          <p className="mt-3 text-sm leading-6 text-slate-600">
            Masukkan email dan kata sandi untuk melanjutkan.
          </p>

          <form onSubmit={handleSubmit} className="mt-7 space-y-5 sm:mt-8">
            <div>
              <label
                htmlFor="email"
                className="mb-2 block text-sm font-medium text-slate-800"
              >
                Alamat email
              </label>
              <div className="relative">
                <Mail
                  size={18}
                  aria-hidden="true"
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
                />
                <input
                  id="email"
                  type="email"
                  autoComplete="email"
                  required
                  maxLength={254}
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="nama@instansi.id"
                  className="min-h-12 w-full rounded-lg border border-slate-300 bg-white py-3 pl-10 pr-4 text-sm text-slate-900 outline-none transition placeholder:text-slate-500 focus:border-blue-600 focus:ring-4 focus:ring-blue-100"
                />
              </div>
            </div>

            <div>
              <label
                htmlFor="password"
                className="mb-2 block text-sm font-medium text-slate-800"
              >
                Kata sandi
              </label>
              <div className="relative">
                <LockKeyhole
                  size={18}
                  aria-hidden="true"
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
                />
                <input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="Masukkan kata sandi"
                  className="min-h-12 w-full rounded-lg border border-slate-300 bg-white py-3 pl-10 pr-4 text-sm text-slate-900 outline-none transition placeholder:text-slate-500 focus:border-blue-600 focus:ring-4 focus:ring-blue-100"
                />
              </div>
            </div>

            {error && (
              <div
                role="alert"
                className="rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-900"
              >
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="min-h-12 w-full rounded-lg bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-blue-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? "Sedang masuk..." : "Masuk"}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-slate-700">
            Belum punya akun?{" "}
            <Link
              href="/register"
              className="font-semibold text-blue-800 underline underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700"
            >
              Daftar sebagai Staff
            </Link>
          </p>

          <p className="mt-8 text-center text-xs leading-5 text-slate-600">
            Akses hanya untuk akun terdaftar. Role akun menentukan data yang dapat Anda buka.
          </p>
        </div>
      </section>
    </main>
  );
}
