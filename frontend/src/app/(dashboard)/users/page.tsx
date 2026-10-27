"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, LoaderCircle, Plus, RefreshCw, ShieldCheck, UserRoundCog } from "lucide-react";
import api from "@/lib/api";
import type { AuthUser } from "@/lib/auth";

type Role = AuthUser["role"];
interface ManagedUser extends AuthUser { is_active: boolean; created_at: string }
interface UserListResponse { data: ManagedUser[] }
interface CreateUserForm { full_name: string; email: string; password: string; role: Role }
const emptyForm: CreateUserForm = { full_name: "", email: "", password: "", role: "staff" };
const roleNames: Record<Role, string> = { admin: "Admin", staff: "Staff", reviewer: "Reviewer" };

function getMessage(error: unknown) {
  if (typeof error === "object" && error !== null && "response" in error) {
    const response = (error as { response?: { data?: { error?: string } } }).response;
    if (response?.data?.error) return response.data.error;
  }
  return "Permintaan gagal diproses. Periksa koneksi dan coba kembali.";
}

export default function UsersPage() {
  const router = useRouter();
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  const [users, setUsers] = useState<ManagedUser[]>([]);
  const [roleDrafts, setRoleDrafts] = useState<Record<number, Role>>({});
  const [form, setForm] = useState<CreateUserForm>(emptyForm);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [savingUserId, setSavingUserId] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const fetchUsers = useCallback(async () => {
    try {
      const response = await api.get<UserListResponse>("/users");
      setUsers(response.data.data);
      setRoleDrafts(Object.fromEntries(response.data.data.map((user) => [user.id, user.role])));
      setError("");
    } catch (err: unknown) {
      setError(getMessage(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let active = true;
    async function load() {
      try {
        const response = await api.get<{ user: AuthUser }>("/auth/me");
        if (!active) return;
        setCurrentUser(response.data.user);
        if (response.data.user.role !== "admin") {
          router.replace("/dashboard");
          return;
        }
        await fetchUsers();
      } catch {
        if (active) router.replace("/login");
      }
    }
    void load();
    return () => { active = false; };
  }, [fetchUsers, router]);

  async function createUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true); setError(""); setNotice("");
    try {
      await api.post("/users", { ...form, full_name: form.full_name.trim(), email: form.email.trim() });
      setForm(emptyForm);
      setNotice("Akun berhasil dibuat. Sampaikan email dan password awal kepada pemilik akun secara aman.");
      setLoading(true);
      await fetchUsers();
    } catch (err: unknown) { setError(getMessage(err)); }
    finally { setSaving(false); }
  }

  async function saveRole(user: ManagedUser) {
    const role = roleDrafts[user.id];
    if (role === user.role) return;
    setSavingUserId(user.id); setError(""); setNotice("");
    try {
      await api.patch(`/users/${user.id}`, { role });
      setNotice(`Role ${user.full_name} diperbarui menjadi ${roleNames[role]}.`);
      await fetchUsers();
    } catch (err: unknown) { setError(getMessage(err)); }
    finally { setSavingUserId(null); }
  }

  async function toggleActive(user: ManagedUser) {
    const nextActive = !user.is_active;
    if (!nextActive && !window.confirm(`Nonaktifkan akun ${user.full_name}? Sesi aktifnya akan dihentikan.`)) return;
    setSavingUserId(user.id); setError(""); setNotice("");
    try {
      await api.patch(`/users/${user.id}`, { is_active: nextActive });
      setNotice(`Akun ${user.full_name} sekarang ${nextActive ? "aktif" : "nonaktif"}.`);
      await fetchUsers();
    } catch (err: unknown) { setError(getMessage(err)); }
    finally { setSavingUserId(null); }
  }

  if (!currentUser || currentUser.role !== "admin") {
    return <main className="flex min-h-screen items-center justify-center bg-[#f4f7f5] text-sm text-slate-700">Memeriksa akses Admin…</main>;
  }

  return (
    <main className="min-h-screen bg-[#f4f7f5] text-slate-950">
      <header className="flex min-h-20 items-center justify-between border-b border-slate-200 bg-white px-5 sm:px-8">
        <div className="flex items-center gap-3"><Link href="/dashboard" aria-label="Kembali ke dashboard" className="rounded-lg border border-slate-300 p-2 text-slate-700 hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"><ArrowLeft size={18} /></Link><div><p className="text-sm text-slate-600">Workspace / Admin</p><h1 className="font-semibold">Kelola user</h1></div></div>
        <span className="inline-flex items-center gap-2 rounded-lg bg-[#143b3c] px-3 py-2 text-xs font-semibold text-white"><ShieldCheck size={15} /> Admin</span>
      </header>

      <div className="mx-auto max-w-7xl space-y-7 p-5 sm:p-8">
        <section className="max-w-3xl"><p className="text-sm font-semibold uppercase tracking-[0.12em] text-[#a64f38]">Akses dan keanggotaan</p><h2 className="mt-2 text-3xl font-semibold tracking-tight">Akun menentukan tanggung jawab kerja.</h2><p className="mt-3 text-sm leading-6 text-slate-700">Buat akun, tetapkan role, atau nonaktifkan akses. Akun nonaktif tidak dapat masuk; riwayat permohonannya tetap tersimpan.</p></section>

        {error && <div role="alert" className="rounded-xl border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-900">{error}</div>}
        {notice && <p role="status" className="rounded-xl border border-emerald-300 bg-emerald-50 px-4 py-3 text-sm text-emerald-950">{notice}</p>}

        <section className="grid gap-7 xl:grid-cols-[minmax(18rem,0.75fr)_minmax(0,1.5fr)]">
          <form onSubmit={createUser} className="h-fit space-y-4 rounded-2xl border border-slate-300 bg-white p-5 sm:p-6">
            <div><div className="mb-3 inline-flex rounded-lg bg-[#edf4f1] p-2 text-[#143b3c]"><Plus size={18} /></div><h3 className="text-lg font-semibold">Buat akun</h3><p className="mt-1 text-sm leading-5 text-slate-600">Password awal hanya digunakan untuk masuk pertama kali. Berikan melalui kanal yang aman.</p></div>
            <div><label htmlFor="user-name" className="mb-1.5 block text-sm font-medium">Nama lengkap</label><input id="user-name" required maxLength={150} value={form.full_name} onChange={(event) => setForm({ ...form, full_name: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-400 px-3 py-2 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /></div>
            <div><label htmlFor="user-email" className="mb-1.5 block text-sm font-medium">Email</label><input id="user-email" type="email" required maxLength={254} value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-400 px-3 py-2 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /></div>
            <div><label htmlFor="user-password" className="mb-1.5 block text-sm font-medium">Password awal</label><input id="user-password" type="password" required minLength={12} maxLength={128} autoComplete="new-password" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-400 px-3 py-2 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /><p className="mt-1 text-xs text-slate-600">Minimal 12 karakter.</p></div>
            <div><label htmlFor="user-role" className="mb-1.5 block text-sm font-medium">Role</label><select id="user-role" value={form.role} onChange={(event) => setForm({ ...form, role: event.target.value as Role })} className="min-h-11 w-full rounded-lg border border-slate-400 bg-white px-3 py-2 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]"><option value="staff">Staff</option><option value="reviewer">Reviewer</option><option value="admin">Admin</option></select></div>
            <button type="submit" disabled={saving} className="inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg bg-[#143b3c] px-4 py-2.5 text-sm font-semibold text-white hover:bg-[#205b55] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#143b3c] disabled:opacity-60"><Plus size={16} />{saving ? "Membuat akun…" : "Buat akun"}</button>
          </form>

          <section className="overflow-hidden rounded-2xl border border-slate-300 bg-white">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-5"><div className="flex items-center gap-3"><span className="rounded-lg bg-[#edf4f1] p-2 text-[#143b3c]"><UserRoundCog size={19} /></span><div><h3 className="font-semibold">Akun terdaftar</h3><p className="text-sm text-slate-600">{users.length} akun · role dan status akses</p></div></div><button type="button" onClick={() => { setLoading(true); void fetchUsers(); }} disabled={loading} aria-label="Muat ulang daftar user" className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-slate-400 px-3 text-sm font-medium hover:bg-slate-50 disabled:opacity-50"><RefreshCw size={15} className={loading ? "animate-spin" : ""} />Muat ulang</button></div>
            {loading ? <div role="status" className="p-12 text-center text-sm text-slate-700"><LoaderCircle size={22} className="mx-auto animate-spin" /><p className="mt-3">Memuat daftar akun…</p></div> : users.length === 0 ? <p className="p-10 text-center text-sm text-slate-700">Belum ada akun untuk ditampilkan.</p> : <div className="divide-y divide-slate-200">{users.map((user) => <article key={user.id} className="grid gap-4 p-5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"><div className="min-w-0"><div className="flex flex-wrap items-center gap-2"><h4 className="font-semibold">{user.full_name}</h4><span className={`rounded px-2 py-0.5 text-xs font-semibold ${user.is_active ? "bg-emerald-100 text-emerald-900" : "bg-slate-200 text-slate-800"}`}>{user.is_active ? "Aktif" : "Nonaktif"}</span>{user.id === currentUser.id && <span className="text-xs text-slate-600">Akun Anda</span>}</div><p className="mt-1 truncate text-sm text-slate-700">{user.email}</p><p className="mt-1 text-xs text-slate-600">Dibuat {new Intl.DateTimeFormat("id-ID", { dateStyle: "medium" }).format(new Date(user.created_at))}</p></div><div className="flex flex-wrap items-center gap-2"><label htmlFor={`role-${user.id}`} className="sr-only">Role {user.full_name}</label><select id={`role-${user.id}`} value={roleDrafts[user.id] ?? user.role} onChange={(event) => setRoleDrafts({ ...roleDrafts, [user.id]: event.target.value as Role })} className="min-h-10 rounded-lg border border-slate-400 bg-white px-3 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]"><option value="staff">Staff</option><option value="reviewer">Reviewer</option><option value="admin">Admin</option></select><button type="button" onClick={() => void saveRole(user)} disabled={savingUserId === user.id || roleDrafts[user.id] === user.role} className="min-h-10 rounded-lg border border-slate-400 px-3 text-sm font-semibold text-slate-900 hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55] disabled:cursor-not-allowed disabled:opacity-50">{savingUserId === user.id ? "Menyimpan…" : "Simpan role"}</button><button type="button" onClick={() => void toggleActive(user)} disabled={savingUserId === user.id || user.id === currentUser.id} className="min-h-10 rounded-lg border border-slate-400 px-3 text-sm font-semibold text-slate-900 hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55] disabled:cursor-not-allowed disabled:opacity-50">{user.is_active ? "Nonaktifkan" : "Aktifkan"}</button></div></article>)}</div>}
          </section>
        </section>
      </div>
    </main>
  );
}
