# Rangkuman Aplikasi Prior Authorization Assistant

Dokumen ini menjelaskan fungsi produk, alur kerja, role, field data, parameter API, konfigurasi, dan batas kemampuan AI berdasarkan kode aplikasi.

## 1. Tujuan dan batas aplikasi

Aplikasi membantu Staff menyiapkan permohonan prior authorization (PA), menyimpan dokumen, mengekstrak informasi dari dokumen, membantu Reviewer memeriksa field terhadap checklist, dan mencatat status proses sampai keputusan payer diterima.

Aplikasi ini **tidak** mengirim permohonan ke payer, tidak menentukan kelayakan medis, dan tidak memutuskan persetujuan/penolakan. Staff tetap mengirim permohonan lewat kanal payer. Reviewer memeriksa kelengkapan dokumen dan bukti. Hasil AI adalah saran ekstraksi yang harus diverifikasi manusia.

## 2. Role dan kewenangannya

| Role | Tugas | Akses utama |
|---|---|---|
| **Admin** | Menyiapkan akun dan checklist payer/layanan; memantau permohonan. | Kelola user, kelola checklist, lihat semua permohonan dan riwayat. Tidak mengisi/mengubah field permohonan sebagai Staff dan tidak menjalankan verifikasi sebagai Reviewer. |
| **Staff** | Membuat dan melengkapi permohonan miliknya; mengunggah/mem-parsing dokumen; berkomunikasi dengan payer di luar aplikasi; mencatat pengajuan dan keputusan payer. | Hanya melihat/mengubah permohonan yang dibuatnya. Dapat mengubah data/dokumen hanya ketika `draft` atau `needs_information`. |
| **Reviewer** | Memeriksa permohonan masuk, meninjau bukti ekstraksi, menandai field, mengevaluasi checklist, meminta informasi, atau menyelesaikan review. | Melihat permohonan semua Staff; hanya mengubah verifikasi field/evaluasi saat permohonan `in_review`. Tidak mencatat keputusan payer. |

Pendaftaran publik membuat akun dengan role `staff`. Role lain hanya ditetapkan Admin melalui Kelola user.

## 3. Alur dari awal sampai akhir

1. **Admin menyiapkan checklist** untuk pasangan payer + jenis layanan. Setiap aturan menghubungkan satu `required_field_key` dengan deskripsi. Checklist harus diaktifkan sebelum Staff dapat mengirim permohonan.
2. **Staff membuat permohonan** (`draft`), memilih pasangan payer/layanan dari checklist aktif, mengisi referensi pasien dan diagnosis, serta NPI penyedia bila tersedia.
3. **Staff mengunggah dokumen** CSV atau PDF berbasis teks. Untuk PDF, Staff memberi konfirmasi bahwa data sintetis atau sudah de-identifikasi sebelum teks dikirim ke OpenAI.
4. **Parser membaca label dan nilai dari dokumen.** PDF berbasis teks dipindai untuk pasangan yang dipisahkan `:` atau `=`, serta pasangan dua kolom pada tabel. Ekstraksi AI dapat melengkapi field semantik yang dikonfigurasi; CSV diproses lokal, dengan dua kolom sebagai label/nilai atau tiga kolom lebih sebagai tabel ber-header.
5. **Staff memeriksa hasil**, membetulkan nilai yang salah/kosong, lalu mengirim permohonan untuk review. Syarat kirim: checklist aktif cocok, memiliki field wajib, sedikitnya satu dokumen diunggah, dan seluruh dokumen berstatus parsing `completed`.
6. Status menjadi **`ready_for_review`** dan masuk antrian Reviewer.
7. **Reviewer memulai review** (`in_review`), memeriksa field dan bukti. Reviewer menandai field `verified`, `corrected`, atau `uncertain`; lalu menjalankan evaluasi checklist.
8. Jika ada kekurangan, Reviewer memakai **Minta informasi** dan mengisi catatan. Status menjadi `needs_information`; Staff melengkapi data/dokumen dan mengirim kembali.
9. Jika semua field wajib memiliki nilai dan diverifikasi serta checklist seluruhnya terpenuhi, Reviewer memilih **Selesaikan review**. Status menjadi `ready_for_submission`.
10. **Staff mengirim permohonan ke payer di luar aplikasi**, lalu memilih **Catat pengajuan ke payer** (`submitted`) dan memasukkan nomor referensi payer jika ada.
11. Jika payer meminta informasi, Staff mencatat permintaan itu (`payer_needs_information`), melengkapi dokumen/field, lalu mengirim tambahan untuk review internal sebelum mengajukannya kembali.
12. Setelah payer menjawab, Staff mencatat outcome dan alasannya. Untuk penolakan atau persetujuan sebagian, Staff juga dapat mencatat pengajuan banding dan respons payer; status banding serta catatan tersimpan dalam riwayat.

Staff dapat membatalkan hanya draft atau permohonan yang dikembalikan (`cancelled`). Draft juga dapat dihapus; penghapusan memakai soft-delete sehingga baris tidak langsung dihapus permanen.

## 4. Arti status

| Nilai API/database | Label di UI | Makna / tindakan berikutnya |
|---|---|---|
| `draft` | Draft | Staff sedang menyiapkan data dan dokumen. |
| `ready_for_review` | Siap direview | Menunggu Reviewer mengambil kasus. |
| `in_review` | Dalam review | Reviewer sedang memeriksa field dan checklist. |
| `needs_information` | Perlu informasi | Staff harus menindaklanjuti catatan Reviewer. |
| `ready_for_submission` | Siap diajukan | Review internal selesai; Staff mengajukan ke payer secara eksternal. |
| `submitted` | Diajukan | Pengajuan eksternal sudah dicatat; menunggu keputusan payer. |
| `payer_needs_information` | Payer meminta informasi | Staff melengkapi permintaan payer dan mengirim tambahan untuk review internal. |
| `appeal_submitted` | Banding diajukan | Banding sudah dikirim melalui kanal eksternal; Staff mencatat respons payer saat diterima. |
| `completed` | Selesai | Staff sudah mencatat outcome payer. Outcome tetap tersimpan pada `payer_decision`; respons banding tercatat di riwayat. |
| `cancelled` | Dibatalkan | Staff membatalkan draft/permohonan yang dikembalikan. |
| `processing`, `error` | Status lama/teknis | Dipertahankan untuk kompatibilitas data; bukan langkah alur baru. |

Transisi yang tersedia: `submit_for_review`, `start_review`, `request_information`, `complete_review`, `record_submission`, `request_payer_information`, `record_decision`, `record_appeal_submission`, `record_appeal_decision`, dan `cancel`. Semua perubahan status dicatat di `pa_request_status_history`.

Pengajuan PA, jawaban tambahan, dan banding tetap dikirim di luar aplikasi lalu dicatat oleh Staff. CMS-0057-F hanya mencakup QHP yang ditawarkan di Federally Facilitated Exchanges untuk payer komersial; aturan tersebut tidak otomatis mencakup semua employer-based atau commercial plans. Persyaratan tenggat, formulir, kanal, dan banding harus dikonfigurasi serta divalidasi untuk payer, produk, layanan, dan negara bagian tertentu sebelum aplikasi disebut patuh.

## 5. Apa yang AI kerjakan

Untuk PDF berbasis teks, backend mengambil teks PDF lalu mengirim teks dan daftar field yang diminta ke OpenAI Chat Completions. AI diminta mengembalikan JSON terstruktur. Sistem membatasi ekstraksi pada field yang diminta, meminta bukti berupa kutipan tepat dari dokumen, meminta confidence `0–1`, dan menyimpan field awal dengan status verifikasi manusia. AI tidak boleh mengarang nilai dan menggunakan `null` jika tidak ditemukan/tidak pasti.

Field bawaan extractor:

| Key | Fungsi |
|---|---|
| `patient_name` | Nama pasien jika tertulis di dokumen. |
| `patient_reference` | Nomor referensi pasien/member. |
| `payer_name` | Nama payer atau health plan. |
| `service_type` | Layanan/prosedur/obat yang diminta. |
| `provider_npi` | NPI penyedia 10 digit jika tertulis. |
| `diagnosis` | Diagnosis atau kode diagnosis yang tercantum. |
| `clinical_notes` | Catatan klinis pendukung yang tertulis. |
| `prior_treatments` | Terapi/perawatan sebelumnya yang tercatat. |

Checklist Admin menentukan field mana yang **wajib**. Hasil ekstraksi dengan sendirinya tidak membuat checklist fulfilled. Field wajib baru fulfilled jika memiliki nilai dan Reviewer memberi status `verified` atau `corrected`. `pending` atau `uncertain` menghasilkan `needs_review`; field hilang/kosong menghasilkan `missing`.

PDF hasil scan tidak di-OCR. Batas PDF: 100 halaman dan teks hasil ekstraksi 100.000 karakter. File upload maksimal 10 MiB. Jangan mengirim PHI/data pasien nyata untuk uji coba; gunakan data sintetis atau de-identifikasi.

## 6. Field bisnis utama

### Permohonan (`pa_requests`)

| Field JSON/database | Fungsi |
|---|---|
| `id` | ID internal; dipakai sebagai `:id` pada URL. |
| `request_number` | Nomor permohonan unik yang dibuat backend. |
| `created_by` | ID Staff pemilik permohonan. |
| `payer_name` | Nama payer; harus cocok dengan pasangan checklist. |
| `service_type` | Jenis layanan; harus cocok dengan pasangan checklist. |
| `provider_npi` | NPI penyedia opsional; jika diisi harus tepat 10 digit angka. |
| `patient_reference` | Identitas/referensi pasien, bukan nama wajib. Gunakan ID sintetis saat testing. |
| `diagnosis` | Diagnosis/kode yang dimasukkan Staff. Untuk testing gunakan label sintetis, bukan diagnosis nyata. |
| `status` | Status workflow pada tabel di atas. |
| `checklist_version_id` | Versi checklist yang terpasang pada permohonan. Dipilih otomatis berdasarkan payer + layanan aktif. |
| `payer_reference` | Nomor/rujukan yang diberikan payer saat Staff mencatat submission; opsional. |
| `submitted_at` | Waktu Staff mencatat pengajuan ke payer. |
| `payer_decision` | Outcome: `approved`, `denied`, `partially_approved`, `withdrawn`. |
| `payer_decision_at` | Waktu Staff mencatat outcome. |
| `payer_decision_note` | Catatan keputusan payer; wajib saat mencatat outcome. |
| `created_at`, `updated_at` | Waktu pembuatan dan perubahan terakhir. |
| `deleted_at` | Waktu soft-delete draft; baris yang terhapus tidak tampil di alur aktif. |

Input create/update permohonan: `payer_name` wajib maks 150 karakter; `service_type` wajib maks 100; `provider_npi` opsional 10 digit; `patient_reference` wajib maks 100; `diagnosis` wajib maks 2.000; `checklist_version_id` opsional karena backend akan mencocokkan checklist aktif.

### Dokumen (`documents`)

`id` identitas dokumen; `pa_request_id` induk permohonan; `original_filename` nama file asli; `storage_key` nama penyimpanan acak yang dibuat server; `mime_type` hanya `application/pdf` atau `text/csv`; `size_bytes` ukuran; `parsing_status` = `pending`, `processing`, `completed`, atau `failed`; `uploaded_by` user pengunggah; `created_at` waktu unggah. File tersimpan pada `DOCUMENT_STORAGE_DIR` (default `backend/var/private-documents`), sedangkan database menyimpan metadata dan storage key.

### Field hasil ekstraksi (`extracted_fields`)

`id`; `pa_request_id`; `document_id`; `field_key`; `field_value`; `confidence` (0 sampai 1, bisa kosong); `evidence_text` (kutipan sumber); `verification_status` (`pending`, `verified`, `corrected`, `uncertain`); `reviewed_by`; `reviewed_at`; `created_at`.

Staff dapat mengubah `field_value` pada draft/needs-information. Reviewer dapat mengubah nilai dan `verification_status` hanya saat `in_review`. `corrected` berarti Reviewer mengoreksi nilai dan memverifikasinya.

### Checklist

`checklist_versions`: `id`, `name`, `payer_name`, `service_type`, `version_number`, `source_url`, `effective_date`, `reviewed_by`, `is_active`, `created_at`. Payer + service + version unik. Checklist baru tidak aktif sampai Admin mengaktifkannya. Aktivasi menonaktifkan versi lain untuk pasangan payer/service yang sama.

`checklist_rules`: `id`, `checklist_version_id`, `rule_key` (unik di dalam versi), `description`, `required_field_key` (field hasil ekstraksi yang menjadi dasar aturan), `is_required`, `created_at`. Aturan wajib harus terhubung ke sebuah field.

`checklist_results`: `id`, `pa_request_id`, `checklist_rule_id`, `status` (`fulfilled`, `missing`, `needs_review`), `reason`, `evidence`, `reviewed_by`, `reviewed_at`, `updated_at`. Ini pemeriksaan kelengkapan berdasar aturan yang disusun Admin, bukan keputusan payer.

### User, sesi, dan riwayat

`users`: `id`, `full_name`, `email`, `password_hash` (hash, bukan password asli), `role`, `is_active`, `created_at`, `updated_at`.

`user_sessions`: `id`, `user_id`, `token_hash` (hash sesi yang disimpan server), `expires_at` (sesi login 12 jam), `created_at`, `revoked_at`. Frontend menyimpan token sesi di `localStorage` dengan key `paa_token`, data user `paa_user`, dan expiry `paa_expires_at`; request API mengirim `Authorization: Bearer <token>`.

`pa_request_status_history`: `id`, `pa_request_id`, `actor_id`, `from_status`, `to_status`, `action`, `note`, `details`, `created_at`.

`audit_logs`: jejak aksi sistem/user; `actor_id`, `pa_request_id`, `action`, `entity_type`, `entity_id`, `details`, `created_at`.

`checklist_version_audit_logs`: jejak pembuatan/aktivasi checklist; `actor_id`, `checklist_version_id`, `action` (`created`/`activated`), `details`, `created_at`.

## 7. Endpoint API dan parameternya

Semua endpoint diawali `/api`. Kecuali health check dan login/register, endpoint memerlukan header `Authorization: Bearer <token>`. Backend secara default memakai port `8080`, tetapi contoh konfigurasi lokal menetapkan `8081`. Frontend mengakses `/api/...` pada port `3000`; Next.js meneruskan request itu ke backend. Jadi URL browser `localhost:3000/api/...` adalah normal; target backend sebenarnya `localhost:8081/api/...`.

| Method + path | Role | Parameter/body penting | Fungsi |
|---|---|---|---|
| `GET /health` | Publik | — | Liveness backend. |
| `GET /health/ready` | Publik | — | Memeriksa readiness/database. |
| `POST /auth/register` | Publik | JSON `full_name`, `email`, `password` (12–128 karakter) | Membuat akun Staff. |
| `POST /auth/login` | Publik | JSON `email`, `password` | Membuat sesi; respons token + user + `expires_at`. |
| `POST /auth/logout` | Login | — | Mencabut sesi aktif. |
| `GET /auth/me` | Login | — | Mengembalikan user dari sesi. |
| `GET /providers/npi/:number` | Login | URL `number`, 10 digit | Cari data penyedia di CMS NPI Registry. Hasil lookup bukan validasi lisensi/coverage/PA. |
| `GET /users` | Admin | — | Daftar user. |
| `POST /users` | Admin | `full_name`, `email`, `password` (12–128), `role` (`admin`/`staff`/`reviewer`) | Membuat user dan memberi role. |
| `PATCH /users/:id` | Admin | URL `id`; salah satu `full_name`, `role`, `is_active` | Mengubah nama/role/status aktif; Admin tidak boleh menurunkan/nonaktifkan dirinya sendiri. |
| `GET /checklist-options` | Login | — | Mengambil checklist aktif beserta aturan untuk dropdown Staff. |
| `GET /admin/checklist-versions` | Admin | — | Daftar checklist dan aturan. |
| `POST /admin/checklist-versions` | Admin | `name`, `payer_name`, `service_type`, `version_number`, `source_url?`, `effective_date?` (`YYYY-MM-DD`), `rules[]` | Membuat versi checklist nonaktif. Setiap rule: `rule_key`, `description`, `required_field_key?`, `is_required`. Minimal satu aturan; aturan wajib perlu field key. |
| `POST /admin/checklist-versions/:id/activate` | Admin | URL `id` | Mengaktifkan checklist; harus punya minimal satu field wajib. |
| `GET /pa-requests` | Login | Query `page` (default 1), `limit` (default 10, maks 100), `status?` | Daftar berhalaman. Staff hanya melihat miliknya; Admin/Reviewer melihat semua. |
| `POST /pa-requests` | Staff | `payer_name`, `service_type`, `provider_npi?`, `patient_reference`, `diagnosis`, `checklist_version_id?` | Membuat draft. |
| `GET /pa-requests/:id` | Login | URL `id` | Detail; Staff hanya pemilik. |
| `PUT /pa-requests/:id` | Staff pemilik | Body sama seperti create | Ubah data hanya draft/needs-information. |
| `DELETE /pa-requests/:id` | Staff pemilik | URL `id` | Soft-delete draft. |
| `GET /pa-requests/:id/documents` | Login | URL `id` | Daftar metadata dokumen. |
| `POST /pa-requests/:id/documents` | Staff pemilik | multipart `file` | Upload PDF/CSV maksimal 10 MiB saat draft/needs-information. |
| `POST /pa-requests/:id/documents/:documentId/parse` | Staff pemilik | PDF: JSON `{ "confirm_synthetic_data": true }`; CSV tidak perlu AI | Parsing PDF/CSV dan simpan hasil field. PDF mengirim teks ke OpenAI setelah konfirmasi. |
| `GET /pa-requests/:id/fields` | Login dengan akses request | URL `id` | Daftar hasil ekstraksi, confidence, bukti, dan status verifikasi. |
| `PATCH /pa-requests/:id/fields/:fieldId` | Staff pemilik atau Reviewer sesuai status | `field_value?`, `verification_status?` | Koreksi nilai/verifikasi. Nilai status yang diterima: `pending`, `verified`, `corrected`, `uncertain`. |
| `POST /pa-requests/:id/checklist/evaluate` | Reviewer, status `in_review` | URL `id` | Evaluasi ulang seluruh field wajib dan simpan hasil checklist. |
| `GET /pa-requests/:id/checklist/results` | Login dengan akses request | URL `id` | Baca hasil evaluasi. |
| `POST /pa-requests/:id/transitions` | Login; hak aksi divalidasi | `action` wajib, `note?`, `payer_reference?`, `payer_decision?`, `payer_decision_note?` | Menjalankan transisi workflow di bawah. |
| `GET /pa-requests/:id/status-history` | Login dengan akses request | URL `id` | Riwayat aktor, perubahan status, aksi, catatan, waktu. |
| `GET /admin/access-check` | Admin | — | Endpoint pengecekan akses Admin. |

Payload untuk `POST /pa-requests/:id/transitions`:

| `action` | Role + status asal | Field tambahan dan hasil |
|---|---|---|
| `submit_for_review` | Staff pemilik; `draft`, `needs_information`, atau `payer_needs_information` | Tidak ada field tambahan; syarat kelengkapan pengiriman harus terpenuhi. Menjadi `ready_for_review`. |
| `start_review` | Reviewer; `ready_for_review` | Tidak ada; menjadi `in_review`. |
| `request_information` | Reviewer; `in_review` | `note` wajib; menjadi `needs_information`. |
| `complete_review` | Reviewer; `in_review` | Tidak ada; checklist harus memiliki aturan dan semua hasil wajib `fulfilled`; menjadi `ready_for_submission`. |
| `record_submission` | Staff pemilik; `ready_for_submission` | `submission_channel` wajib; `payer_reference` opsional; menjadi `submitted`, waktu dan kanal dicatat di riwayat. |
| `request_payer_information` | Staff pemilik; `submitted` | `note` wajib; menjadi `payer_needs_information`. |
| `record_decision` | Staff pemilik; `submitted` | `payer_decision` wajib salah satu 4 outcome; `payer_decision_note` wajib; menjadi `completed`. |
| `record_appeal_submission` | Staff pemilik; `completed` setelah ditolak/disetujui sebagian | `note` wajib; menjadi `appeal_submitted`. |
| `record_appeal_decision` | Staff pemilik; `appeal_submitted` | `payer_decision` dan `payer_decision_note` wajib; menjadi `completed`, respons banding dicatat di riwayat. |
| `cancel` | Staff pemilik; `draft`/`needs_information` | `note` wajib sebagai alasan; menjadi `cancelled`. |

Untuk semua path `:id`, ID permohonan harus bilangan bulat positif. `:documentId`, `:fieldId`, dan checklist/user `:id` juga harus ID valid.

## 8. Konfigurasi lingkungan

Backend membaca file `backend/.env` saat dijalankan dari folder backend. Jangan membagikan atau menaruh key rahasia di dokumentasi.

| Variabel | Fungsi | Default/ketentuan |
|---|---|---|
| `APP_PORT` | Port HTTP Go API. | Default kode `8080`; contoh `.env.example` `8080`. Untuk setup yang digunakan frontend lokal, set `8081`. |
| `APP_ENV` | Tercantum di `.env.example`. | Saat ini belum dibaca oleh kode backend. |
| `DB_HOST` | Host PostgreSQL. | Wajib. |
| `DB_PORT` | Port PostgreSQL. | Wajib; integer 1–65535, biasanya `5432`. |
| `DB_USER` | User PostgreSQL. | Wajib. |
| `DB_PASSWORD` | Password PostgreSQL. | Wajib, simpan hanya di `.env`. |
| `DB_NAME` | Nama database. | Wajib. |
| `DB_SSLMODE` | Mode TLS PostgreSQL (`disable`, `require`, dan seterusnya sesuai deployment). | Wajib; contoh lokal `disable`. |
| `OPENAI_API_KEY` | Kredensial untuk ekstraksi PDF via OpenAI. | Tanpa key, parsing AI PDF gagal; CSV tetap bisa diparse tanpa OpenAI. |
| `OPENAI_MODEL` | Nama model OpenAI. | Jika kosong, kode memakai `gpt-4o-mini`; `.env.example` saat ini memberi `gpt-4.1-mini`, jadi nilai dari `.env` yang menentukan. |
| `OPENAI_TIMEOUT_SECONDS` | Batas waktu panggilan OpenAI. | Default 30 detik; nilai kode hanya diterima bila 5–60. |
| `DOCUMENT_STORAGE_DIR` | Direktori file dokumen lokal. | Tidak ditampilkan pada contoh env; default `backend/var/private-documents` jika backend dijalankan dari folder `backend`. |
| `BACKEND_API_URL` | Tujuan rewrite `/api/*` di Next.js menuju Go API. | Frontend `.env.example`: `http://localhost:8081/api`; diubah perlu restart Next dev server. |

Backend menjalankan migrasi SQL yang belum tercatat pada startup. Frontend menggunakan Axios dengan `baseURL: /api`, jadi browser tidak perlu memanggil port backend secara langsung.

## 9. Cara menjalankan lokal

1. PostgreSQL harus aktif dan database sesuai `DB_*` sudah tersedia.
2. Terminal backend: `cd backend`; isi `.env` sesuai `backend/.env.example`; jalankan `go run .`. Pastikan log menyebut port yang diinginkan (`8081` untuk setup contoh frontend).
3. Terminal frontend kedua: `cd frontend`; isi `.env.local` berdasarkan `frontend/.env.example`; jalankan `npm run dev`. Aplikasi biasanya dibuka di `http://localhost:3000`.
4. Jika mengubah `BACKEND_API_URL` atau konfigurasi Next, restart frontend. Jika mengubah port/env Go, restart backend.
5. Pastikan `/api/health/ready` merespons dan terminal backend tidak menunjukkan error koneksi database/migrasi.

## 10. Masalah umum dan cara membaca error

- **Checklist version belum tersedia**: Admin perlu membuat checklist untuk payer + service yang dipilih, menghubungkan aturan wajib ke field, lalu mengaktifkannya.
- **422 saat parse PDF**: biasanya PDF berupa hasil scan tanpa teks, rusak, terlalu panjang, atau teks tidak dapat diekstrak. Gunakan PDF teks seperti sample sintetis yang disediakan.
- **422 saat evaluasi checklist**: permohonan tidak memiliki checklist version aktif/terpasang atau versi itu tidak punya aturan field wajib. Periksa setup Admin dan pasangan payer/layanan.
- **500 saat transisi/status history**: lihat error paling pertama di terminal backend Go; penyebab database ada di log backend, bukan URL port browser saja. Pastikan migrasi terbaru terpasang dan database dapat diakses.
- **Browser menampilkan `localhost:3000/api/...`**: itu jalur proxy frontend. Jika `BACKEND_API_URL` benar, Next.js meneruskan ke backend port 8081. Periksa Network tab dan `next.config.ts` jika respons 404; restart Next setelah mengubah `.env.local`.
- **NPI tidak ditemukan**: NPI opsional di form, harus 10 digit jika diisi. Lookup memakai CMS NPI Registry eksternal; lookup yang gagal tidak menghalangi input manual.
- **Field checklist tidak fulfilled**: pastikan key di `required_field_key` identik dengan `field_key` hasil ekstraksi, nilai tidak kosong, dan Reviewer menandai `verified`/`corrected` sebelum evaluasi ulang.

## 11. Batas dokumentasi variabel

“Variabel” dalam panduan ini berarti konfigurasi runtime, parameter endpoint, field payload/database, key field hasil AI, dan nilai status yang perlu dipahami saat menggunakan/mengintegrasikan aplikasi. Variabel lokal sementara di dalam fungsi Go/React tidak didaftarkan satu per satu karena nilainya hanya berlaku pada implementasi internal dan bukan kontrak aplikasi. Sumber kontrak teknis: `backend/internal/routes/routes.go`, handler/service backend, `backend/migrations/*.sql`, `frontend/src/lib/api.ts`, dan `frontend/next.config.ts`.
