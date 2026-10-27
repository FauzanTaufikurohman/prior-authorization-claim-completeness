"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { LockKeyhole, Mail, ShieldCheck, UserRound } from "lucide-react";
import api from "@/lib/api";

function errorMessage(error: unknown) {
  if (typeof error === "object" && error !== null && "response" in error) {
    const response = (error as { response?: { data?: { error?: string } } }).response;
    if (response?.data?.error) return response.data.error;
  }
  if (typeof error === "object" && error !== null && "request" in error) {
    return "API tidak merespons. Jalankan backend di port 8081, lalu muat ulang halaman dan coba lagi.";
  }
  return "Pendaftaran gagal. Periksa koneksi dan coba kembali.";
}

export default function RegisterPage() {
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [registered, setRegistered] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (password !== confirmation) { setError("Konfirmasi password belum sama."); return; }
    setLoading(true);
    try {
      await api.post("/auth/register", { full_name: fullName.trim(), email: email.trim(), password });
      setRegistered(true);
    } catch (err: unknown) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="grid min-h-screen bg-[#f4f7f5] lg:grid-cols-[0.9fr_1.1fr]">
      <section className="relative hidden overflow-hidden bg-[#143b3c] p-12 text-white lg:flex lg:flex-col lg:justify-between">
        <div className="absolute -right-28 -top-24 h-96 w-96 rounded-full border border-white/10" aria-hidden="true" />
        <div className="absolute -bottom-40 -left-24 h-[34rem] w-[34rem] rounded-full border border-white/10" aria-hidden="true" />
        <div className="relative flex items-center gap-3">
          <div className="rounded-xl bg-[#d9785c] p-3"><ShieldCheck size={25} /></div>
          <div><p className="text-lg font-semibold">Prior Authorization Assistant</p><p className="text-sm text-teal-100/70">Provider document workspace</p></div>
        </div>
        <div className="relative max-w-lg">
          <p className="mb-4 text-sm font-semibold uppercase tracking-[0.18em] text-[#f0b19d]">Mulai alur kerja</p>
          <h1 className="text-4xl font-semibold leading-tight">One workspace to prepare prior authorization requests.</h1>
          <p className="mt-5 max-w-md leading-7 text-teal-50/75">Akun baru mendapat role Staff. Admin dapat memberikan role Reviewer atau Admin dari halaman kelola user.</p>
        </div>
        <p className="relative text-sm text-teal-100/60">Prior authorization workflow for provider teams</p>
      </section>

      <section className="flex items-center justify-center px-5 py-10 sm:px-8">
        <div className="w-full max-w-md">
          <div className="mb-8 lg:hidden"><div className="mb-4 inline-flex rounded-xl bg-[#143b3c] p-3 text-white"><ShieldCheck size={26} /></div><h1 className="text-xl font-semibold">Prior Authorization Assistant</h1></div>
          {registered ? (
            <div role="status" className="rounded-2xl border border-emerald-300 bg-white p-6">
              <p className="text-sm font-semibold uppercase tracking-[0.12em] text-emerald-800">Pendaftaran selesai</p>
              <h2 className="mt-2 text-2xl font-semibold text-slate-950">Akun Staff berhasil dibuat.</h2>
              <p className="mt-3 text-sm leading-6 text-slate-700">Gunakan email yang didaftarkan dan password Anda untuk masuk.</p>
              <Link href="/login" className="mt-6 inline-flex min-h-11 items-center rounded-lg bg-[#143b3c] px-4 py-2.5 text-sm font-semibold text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2">Lanjut ke halaman masuk</Link>
            </div>
          ) : <>
          <p className="text-sm font-semibold text-[#a64f38]">AKUN BARU</p>
          <h2 className="mt-2 text-3xl font-semibold tracking-tight text-slate-950">Daftar sebagai Staff</h2>
          <p className="mt-3 text-sm leading-6 text-slate-600">Isi data akun. Setelah masuk, Anda dapat membuat dan mengelola permohonan milik sendiri.</p>

          <form onSubmit={handleSubmit} className="mt-8 space-y-5">
            <div><label htmlFor="full-name" className="mb-2 block text-sm font-medium text-slate-800">Nama lengkap</label><div className="relative"><UserRound size={18} aria-hidden="true" className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" /><input id="full-name" required maxLength={150} autoComplete="name" value={fullName} onChange={(event) => setFullName(event.target.value)} className="min-h-12 w-full rounded-xl border border-slate-400 bg-white py-3 pl-10 pr-4 text-sm text-slate-950 outline-none focus-visible:ring-2 focus-visible:ring-[#205b55]" /></div></div>
            <div><label htmlFor="register-email" className="mb-2 block text-sm font-medium text-slate-800">Email</label><div className="relative"><Mail size={18} aria-hidden="true" className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" /><input id="register-email" type="email" required maxLength={254} autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} className="min-h-12 w-full rounded-xl border border-slate-400 bg-white py-3 pl-10 pr-4 text-sm text-slate-950 outline-none focus-visible:ring-2 focus-visible:ring-[#205b55]" /></div></div>
            <div><label htmlFor="register-password" className="mb-2 block text-sm font-medium text-slate-800">Password</label><div className="relative"><LockKeyhole size={18} aria-hidden="true" className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" /><input id="register-password" type="password" required minLength={12} maxLength={128} autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} className="min-h-12 w-full rounded-xl border border-slate-400 bg-white py-3 pl-10 pr-4 text-sm text-slate-950 outline-none focus-visible:ring-2 focus-visible:ring-[#205b55]" /></div><p className="mt-1.5 text-xs text-slate-600">Minimal 12 karakter.</p></div>
            <div><label htmlFor="confirm-password" className="mb-2 block text-sm font-medium text-slate-800">Ulangi password</label><input id="confirm-password" type="password" required minLength={12} maxLength={128} autoComplete="new-password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} className="min-h-12 w-full rounded-xl border border-slate-400 bg-white px-4 py-3 text-sm text-slate-950 outline-none focus-visible:ring-2 focus-visible:ring-[#205b55]" /></div>
            {error && <p role="alert" className="rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-900">{error}</p>}
            <button type="submit" disabled={loading} className="min-h-12 w-full rounded-xl bg-[#143b3c] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-[#205b55] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#143b3c] disabled:cursor-wait disabled:opacity-60">{loading ? "Membuat akun…" : "Buat akun Staff"}</button>
          </form>
          <p className="mt-7 text-center text-sm text-slate-700">Sudah punya akun? <Link href="/login" className="font-semibold text-[#205b55] underline underline-offset-4 focus-visible:outline focus-visible:outline-2">Masuk</Link></p>
          </>}
        </div>
      </section>
    </main>
  );
}
