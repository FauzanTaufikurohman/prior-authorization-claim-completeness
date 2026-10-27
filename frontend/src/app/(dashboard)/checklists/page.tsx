"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import axios from "axios";
import { ArrowLeft, Check, Plus, Save } from "lucide-react";
import api from "@/lib/api";

type Rule = { rule_key: string; description: string; required_field_key: string; is_required: boolean };
type Version = { id: number; name: string; payer_name: string; service_type: string; version_number: number; is_active: boolean; effective_date: string | null; rules: (Rule & { id: number })[] };
const blankRule = (): Rule => ({ rule_key: "", description: "", required_field_key: "", is_required: true });
const fieldChoices = [
  ["patient_name", "Nama pasien"],
  ["patient_reference", "ID atau referensi pasien"],
  ["payer_name", "Payer / health plan"],
  ["service_type", "Requested service"],
  ["provider_npi", "Provider NPI"],
  ["diagnosis", "Diagnosis or diagnosis code"],
  ["clinical_notes", "Ringkasan klinis pendukung"],
  ["prior_treatments", "Riwayat terapi yang tercatat"],
] as const;

function getErrorMessage(cause: unknown): string {
  if (axios.isAxiosError(cause)) {
    const data = cause.response?.data as { error?: string; message?: string } | undefined;
    return data?.error ?? data?.message ?? cause.message;
  }
  return cause instanceof Error ? cause.message : "Terjadi kesalahan saat memuat checklist.";
}

export default function ChecklistsPage() {
  const [versions, setVersions] = useState<Version[]>([]);
  const [form, setForm] = useState({ name: "", payer_name: "", service_type: "", version_number: "1", source_url: "", effective_date: "" });
  const [rules, setRules] = useState<Rule[]>([blankRule()]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function load() {
    setLoading(true);
    try {
      const response = await api.get<{ data: Version[] }>("/admin/checklist-versions");
      setVersions(response.data.data);
      setError("");
    } catch (cause) {
      setError(getErrorMessage(cause));
    } finally { setLoading(false); }
  }
  useEffect(() => { const pendingLoad = Promise.resolve().then(() => load()); void pendingLoad; }, []);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(""); setNotice("");
    try {
      await api.post("/admin/checklist-versions", {
        ...form, version_number: Number(form.version_number), source_url: form.source_url || null,
        effective_date: form.effective_date || null,
        rules: rules.map((rule) => ({ ...rule, required_field_key: rule.required_field_key || null })),
      });
      setForm({ name: "", payer_name: "", service_type: "", version_number: "1", source_url: "", effective_date: "" });
      setRules([blankRule()]); setNotice("Versi disimpan sebagai draft. Tinjau lalu aktifkan agar bisa dipasangkan otomatis.");
      await load();
    } catch (cause) { setError(getErrorMessage(cause)); }
    finally { setBusy(false); }
  }

  async function activate(id: number) {
    setBusy(true); setError(""); setNotice("");
    try { await api.post(`/admin/checklist-versions/${id}/activate`); setNotice("Checklist aktif untuk pasangan payer dan layanan ini."); await load(); }
    catch (cause) { setError(getErrorMessage(cause)); }
    finally { setBusy(false); }
  }

  return <main className="mx-auto max-w-6xl space-y-8 px-4 py-8 sm:px-6">
    <header className="flex items-center gap-3">
      <Link href="/dashboard" aria-label="Kembali ke dashboard" className="rounded-lg border border-slate-300 p-2 text-slate-700 hover:bg-slate-100"><ArrowLeft size={18} /></Link>
      <div><p className="text-sm text-slate-600">Workspace / Admin</p><h1 className="text-2xl font-semibold text-slate-900">Kelola checklist</h1></div>
    </header>
    <p className="max-w-3xl text-sm leading-6 text-slate-600">Checklist menentukan informasi pendukung yang perlu dikumpulkan per payer dan layanan. Tinjau sumber kebijakan dan masa berlaku setiap checklist sebelum diaktifkan. Checklist ini membantu pemeriksaan kelengkapan dan tidak menentukan keputusan klinis atau payer.</p>
    {error && <div role="alert" className="rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-800">{error}</div>}
    {notice && <div role="status" className="rounded-lg border border-emerald-300 bg-emerald-50 p-3 text-sm text-emerald-800">{notice}</div>}

    <section className="rounded-xl border border-slate-200 bg-white p-5 sm:p-6">
      <h2 className="text-lg font-semibold">Buat versi checklist</h2>
      <form onSubmit={create} className="mt-5 space-y-5">
        <div className="grid gap-4 sm:grid-cols-2">
          {([["name", "Nama checklist"], ["payer_name", "Skema / penjamin"], ["service_type", "Alur layanan"], ["version_number", "Nomor versi"]] as const).map(([key, label]) => <label key={key} className="space-y-1.5 text-sm font-medium text-slate-800">{label}<input required min={key === "version_number" ? 1 : undefined} type={key === "version_number" ? "number" : "text"} value={form[key]} onChange={(event) => setForm({ ...form, [key]: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-normal focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#205b55]" /></label>)}
          <label className="space-y-1.5 text-sm font-medium text-slate-800">Tanggal berlaku<input type="date" value={form.effective_date} onChange={(event) => setForm({ ...form, effective_date: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal" /></label>
          <label className="space-y-1.5 text-sm font-medium text-slate-800">Policy source URL<input type="url" value={form.source_url} onChange={(event) => setForm({ ...form, source_url: event.target.value })} className="min-h-11 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal" /><span className="block text-xs font-normal text-slate-600">Use payer policy documentation and record the effective date.</span></label>
        </div>
        <div className="space-y-3"><div className="flex items-center justify-between"><h3 className="font-semibold">Aturan pemeriksaan</h3><button type="button" onClick={() => setRules([...rules, blankRule()])} className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-slate-300 px-3 text-sm hover:bg-slate-50"><Plus size={16} />Tambah aturan</button></div>
          {rules.map((rule, index) => <div key={index} className="grid gap-3 rounded-lg bg-slate-50 p-4 sm:grid-cols-3">
            <label className="space-y-1 text-sm">Data yang harus diperiksa<select required value={rule.required_field_key} onChange={(event) => { const key = event.target.value; const label = fieldChoices.find(([fieldKey]) => fieldKey === key)?.[1] ?? "Data"; setRules(rules.map((item, i) => i === index ? { ...item, required_field_key: key, rule_key: key, description: item.description || `${label} tercantum pada dokumen pendukung` } : item)); }} className="min-h-10 w-full rounded-md border border-slate-300 bg-white px-3"><option value="">Pilih data</option>{fieldChoices.map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select><span className="block text-xs text-slate-600">Aplikasi mencari data ini dari CSV atau PDF.</span></label>
            <label className="space-y-1 text-sm">Deskripsi<textarea required value={rule.description} onChange={(event) => setRules(rules.map((item, i) => i === index ? { ...item, description: event.target.value } : item))} rows={2} className="w-full rounded-md border border-slate-300 px-3 py-2" /></label>
          </div>)}
          <p className="text-xs leading-5 text-slate-600">Field yang dipilih dicocokkan dengan hasil ekstraksi dokumen. Minimal satu field wajib harus terisi dan diverifikasi Reviewer sebelum review selesai.</p>
        </div>
        <button disabled={busy} className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-[#205b55] px-4 font-medium text-white hover:bg-[#174742] disabled:opacity-60"><Save size={17} />Simpan sebagai draft</button>
      </form>
    </section>

    <section className="space-y-3"><h2 className="text-lg font-semibold">Versi yang tersedia</h2>
      {loading ? <p role="status" className="text-sm text-slate-600">Memuat checklist…</p> : versions.length === 0 ? <p className="rounded-lg border border-slate-200 bg-white p-5 text-sm text-slate-600">Belum ada versi checklist. Tambahkan checklist untuk payer dan layanan yang dipakai Staff.</p> : versions.map((version) => <article key={version.id} className="rounded-xl border border-slate-200 bg-white p-5">
        <div className="flex flex-wrap items-start justify-between gap-3"><div><h3 className="font-semibold">{version.name} <span className="font-normal text-slate-600">· v{version.version_number}</span></h3><p className="mt-1 text-sm text-slate-600">{version.payer_name} / {version.service_type}{version.effective_date ? ` · berlaku ${version.effective_date}` : ""}</p></div>{version.is_active ? <span className="inline-flex items-center gap-1 rounded-md bg-emerald-100 px-2.5 py-1 text-sm font-medium text-emerald-800"><Check size={15} />Aktif</span> : <button disabled={busy} onClick={() => void activate(version.id)} className="min-h-10 rounded-lg border border-[#205b55] px-3 text-sm font-medium text-[#205b55] hover:bg-emerald-50 disabled:opacity-50">Aktifkan</button>}</div>
        <ul className="mt-4 grid gap-2 text-sm sm:grid-cols-2">{version.rules.map((rule) => <li key={rule.id} className="rounded-md bg-slate-50 p-3"><span className="font-medium">{rule.description}</span><span className="mt-1 block text-slate-600">Field: {rule.required_field_key ?? "tidak ditautkan"} · {rule.is_required ? "wajib" : "opsional"}</span></li>)}</ul>
      </article>)}
    </section>
  </main>;
}
