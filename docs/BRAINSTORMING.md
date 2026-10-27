# BRAINSTORMING.md
# Prior Authorization Assistant — Brainstorming & Development Direction

**Status:** Draft awal untuk handoff ke AI coding agent  
**Tujuan:** Menetapkan arah pengembangan, prioritas, dan batasan kerja berdasarkan kondisi proyek yang diketahui.  
**Catatan:** Detail implementasi yang belum diverifikasi harus diperiksa langsung dari repository; jangan dianggap sebagai fakta teknis final.

---

## 1. Visi Produk

Prior Authorization Assistant adalah aplikasi untuk membantu proses administrasi prior authorization melalui alur pengelolaan permintaan, pemrosesan dokumen, ekstraksi informasi, verifikasi field, dan evaluasi checklist persyaratan.

Aplikasi harus membantu pengguna memahami:
- Informasi apa yang berhasil diekstrak dari dokumen.
- Informasi apa yang belum tersedia atau perlu diverifikasi.
- Persyaratan apa yang telah terpenuhi, belum terpenuhi, atau memerlukan peninjauan.
- Bukti sumber yang mendukung informasi dan hasil evaluasi, apabila tersedia.

AI berfungsi sebagai alat bantu pemrosesan dan peninjauan informasi. AI tidak boleh dianggap sebagai sumber kebenaran yang tidak perlu diverifikasi atau sebagai pengganti keputusan akhir pihak berwenang.

## 2. Tujuan Brainstorming

Dokumen ini digunakan untuk:
1. Menentukan urutan pengembangan yang masuk akal.
2. Memisahkan stabilisasi fitur yang ada dari penambahan fitur baru.
3. Memberikan konteks yang konsisten kepada AI coding agent.
4. Menghindari perubahan arsitektur atau kontrak API berdasarkan asumsi.
5. Menetapkan kondisi yang harus dipenuhi sebelum sebuah task dinyatakan selesai.

## 3. Gambaran Alur Produk

Alur utama yang sudah dikembangkan menurut konteks proyek:

1. Pengguna login.
2. Pengguna membuka dashboard dan daftar permintaan prior authorization.
3. Pengguna membuka halaman detail permintaan.
4. Pengguna mengunggah dokumen.
5. Sistem memproses dokumen dan mengekstrak field.
6. Pengguna meninjau atau memperbarui nilai field dan status verifikasinya.
7. Sistem mengevaluasi checklist berdasarkan field yang tersedia.
8. Pengguna meninjau hasil checklist dan menentukan tindak lanjut sesuai kewenangannya.

**Instruksi untuk agent:** verifikasi setiap langkah melalui kode dan pengujian aktual. Keberadaan halaman atau tombol saja tidak membuktikan alur berfungsi end-to-end.

## 4. Teknologi dan Lokasi Repository yang Diketahui

### Frontend
- Next.js App Router
- React
- TypeScript
- Axios untuk request HTTP
- Token Bearer dibaca dari local storage dengan key `paa_token`

Lokasi:
`C:\src\go\prior-authorization-assistant\frontend`

File yang diketahui:
- `src/app/(auth)/login/page.tsx`
- `src/app/(dashboard)/dashboard/page.tsx`
- `src/app/(dashboard)/requests/page.tsx`
- `src/app/(dashboard)/requests/[id]/page.tsx`
- `src/lib/api.ts`
- `src/lib/auth.ts`

### Backend
- Go
- Gin
- PostgreSQL

Lokasi:
`C:\src\go\prior-authorization-assistant\backend`

Struktur handler, route, service, model, konfigurasi database, dan skema aktual wajib diperiksa langsung dari repository. Jangan mengasumsikan nama file atau struktur tabel yang belum diverifikasi.

## 5. Status Saat Ini dan Masalah Diketahui

Fitur yang telah dikembangkan menurut konteks sebelumnya:
- Login dan halaman dashboard.
- Daftar permintaan serta detail permintaan.
- Upload dokumen.
- Pemrosesan/parse dokumen.
- Tampilan extracted fields dan perubahan nilai/status verifikasi.
- Evaluasi checklist.
- Pengambilan ulang data permintaan, fields, dan checklist dari backend.

Status tiap fitur perlu diuji ulang sebelum dianggap stabil.

### Bug terbaru: upload dokumen menghasilkan HTTP 400

Request yang dilaporkan:
`POST http://localhost:3000/api/pa-requests/1/documents`

Respons:
`400 Bad Request`

Frontend sebelumnya mengirim file sebagai multipart form dengan key `document`. Belum dipastikan apakah key tersebut cocok dengan handler backend atau apakah validasi lain yang menolak request.

Agent harus:
1. Memeriksa route backend yang sebenarnya.
2. Membaca handler upload dan validasi multipart.
3. Membaca response body dari request yang gagal dan log backend yang relevan.
4. Memeriksa nama field multipart, batas ukuran, tipe file, dan format response.
5. Menentukan akar masalah sebelum mengubah kode.
6. Memperbaiki bagian yang terbukti menjadi penyebab.
7. Menguji skenario sukses dan gagal.
8. Memastikan dokumen yang berhasil disimpan bisa ditampilkan kembali setelah refresh, menggunakan endpoint yang benar-benar tersedia.

Jangan mengubah nama field, route, atau payload berdasarkan dugaan.

## 6. Prioritas Pengembangan

### P0 — Stabilitas alur inti
**Tujuan:** memastikan alur dasar dapat digunakan secara konsisten.

Task:
- Memperbaiki upload dokumen HTTP 400.
- Memastikan dokumen yang tersimpan dapat dimuat kembali setelah refresh.
- Memastikan hasil parsing dan extracted fields konsisten dengan data backend.
- Memastikan perubahan field tersimpan dan tetap ada setelah refresh.
- Memastikan checklist dapat dievaluasi ulang setelah perubahan field.
- Menyediakan pesan error dan status loading yang jelas.

**Kriteria selesai:**
- Penyebab error upload telah dibuktikan dan diperbaiki.
- File valid dapat diunggah dan digunakan dalam alur parsing.
- File tidak valid ditolak dengan pesan yang jelas.
- Data dokumen, field, dan checklist tidak hanya bergantung pada state lokal browser.
- Pengujian relevan dijalankan dan hasilnya dicatat.

### P1 — Penyelesaian alur dokumen
**Tujuan:** membuat status dan riwayat pemrosesan dokumen mudah dipahami.

Ide:
- Daftar dokumen terkait suatu permintaan.
- Nama file, ukuran, waktu unggah, dan status parsing.
- Status gagal beserta pesan yang aman untuk ditampilkan.
- Kemampuan memproses ulang jika didukung backend.
- Penanganan file kosong, format tidak didukung, file rusak, dan duplikasi.
- Indikator field yang belum diverifikasi.

**Catatan:** kemampuan seperti retry, penghapusan, atau beberapa dokumen per permintaan hanya dikerjakan setelah dukungan backend dan aturan produknya dikonfirmasi.

### P2 — Kualitas ekstraksi dan review AI
**Tujuan:** meningkatkan keandalan informasi yang diekstrak dan kemudahan verifikasinya.

Ide:
- Menetapkan skema field yang jelas.
- Memvalidasi tipe dan format hasil ekstraksi.
- Menampilkan bukti sumber, halaman, dan confidence jika tersedia.
- Menandai informasi yang hilang atau bertentangan antar dokumen.
- Memisahkan hasil ekstraksi otomatis dari hasil verifikasi pengguna.
- Menyusun dataset uji representatif menggunakan data sintetis, publik, atau data yang telah mendapat izin.
- Mengukur kualitas ekstraksi berdasarkan field yang benar, bukan hanya apakah proses berhasil dijalankan.

**Batasan:** jangan menyatakan informasi medis atau administratif sebagai benar hanya karena dihasilkan model. Hasil harus dapat ditinjau dan diverifikasi.

### P3 — Kualitas evaluasi checklist
**Tujuan:** membuat hasil evaluasi dapat dijelaskan dan ditelusuri.

Ide:
- Menampilkan aturan yang dievaluasi.
- Menjelaskan field atau bukti yang mendasari hasil.
- Membedakan `fulfilled`, `missing`, dan `needs_review` sesuai kontrak backend.
- Menampilkan waktu evaluasi terakhir jika tersedia.
- Menghindari hasil lama terlihat seolah-olah merupakan hasil evaluasi terbaru setelah field diubah.
- Menambahkan pengujian untuk field kosong, nilai tidak valid, dan field yang belum diverifikasi.

### P4 — Keamanan, audit, dan kesiapan penggunaan
**Tujuan:** mengurangi risiko akses tidak sah, perubahan yang tidak terlacak, dan kebocoran data.

Ide:
- Verifikasi autentikasi dan otorisasi di setiap endpoint.
- Kontrol akses dokumen pada tingkat pengguna/permintaan.
- Validasi file di backend; jangan mengandalkan validasi frontend.
- Batas ukuran, tipe file, dan penanganan nama file yang aman.
- Audit log untuk perubahan field dan tindakan penting jika dibutuhkan.
- Hindari menyimpan token, secret, atau data sensitif dalam log dan dokumentasi.
- Gunakan data sintetis atau yang telah diizinkan untuk pengujian.
- Dokumentasikan kebijakan retensi dan penghapusan dokumen jika aplikasi akan dipakai dengan data nyata.

## 7. Ide Fitur untuk Dipertimbangkan, Bukan Langsung Dikerjakan

Ide berikut perlu diprioritaskan berdasarkan kebutuhan pengguna dan kemampuan backend:
- Riwayat dokumen per permintaan.
- Ringkasan kelengkapan permintaan.
- Penjelasan alasan checklist membutuhkan review.
- Deteksi konflik antar dokumen.
- Riwayat perubahan field.
- Filter dan pencarian permintaan.
- Ekspor ringkasan hasil review.
- Notifikasi status parsing.
- Dashboard metrik proses.

Jangan membangun seluruh ide ini sekaligus. Pilih berdasarkan nilai pengguna, dependensi teknis, waktu, dan risiko.

## 8. Prinsip Teknis untuk AI Agent

1. Repository aktual adalah sumber kebenaran teknis utama.
2. Baca implementasi yang relevan sebelum mengubah kode.
3. Jangan menebak endpoint, nama field request, skema response, tabel, atau kolom.
4. Gunakan dependensi dan pola yang sudah ada jika memadai.
5. Hindari refactor luas ketika perubahan kecil sudah menyelesaikan masalah.
6. Jangan menghapus atau mengubah fitur yang telah berjalan tanpa alasan yang jelas.
7. Jangan menambah integrasi eksternal sebelum kebutuhan dan batasannya disepakati.
8. Jangan melakukan perubahan database destruktif tanpa persetujuan dan rencana migrasi.
9. Jangan menaruh password, token, API key, kredensial, atau data sensitif di source code maupun dokumen.
10. Pisahkan fakta yang telah diverifikasi dari asumsi dan rekomendasi.
11. Catat perubahan kontrak API dan skema data.
12. Jalankan pengujian yang relevan dan laporkan hasil aktual.
13. Jika pengujian tidak dapat dijalankan, jelaskan penyebab dan langkah verifikasi yang belum dilakukan.

## 9. Alur Kerja yang Wajib Diikuti Agent

Untuk setiap task:

1. **Inspect:** periksa status Git, file terkait, route, handler, service, model, dan konfigurasi.
2. **Reproduce:** reproduksi bug atau jalankan test yang relevan.
3. **Diagnose:** jelaskan akar masalah berdasarkan bukti.
4. **Plan:** tulis rencana singkat dan file yang akan diubah.
5. **Implement:** lakukan perubahan minimal yang diperlukan.
6. **Test:** jalankan test, lint, type-check, build, atau pengujian manual yang relevan.
7. **Review:** periksa regresi, keamanan, dan konsistensi data.
8. **Document:** perbarui dokumen bila arsitektur atau kontrak berubah.
9. **Report:** rangkum hasil dan rekomendasikan satu task berikutnya.

## 10. Definition of Done

Task dinyatakan selesai jika:
- Acceptance criteria terpenuhi.
- Implementasi sesuai kontrak aktual.
- Alur utama yang terkait telah diuji.
- Penanganan error relevan tersedia.
- Tidak ada regresi yang diketahui dari pengujian yang dijalankan.
- Hasil pengujian dilaporkan secara akurat.
- Dokumentasi diperbarui bila diperlukan.
- Tidak ada secret atau data sensitif yang tidak diperlukan dalam perubahan.

## 11. Task Pertama yang Disarankan

**Judul:** Stabilize document upload and persistence.

Urutan kerja:
1. Audit request upload frontend.
2. Audit route dan handler upload backend.
3. Periksa response body error dan log server.
4. Temukan akar masalah HTTP 400.
5. Perbaiki kontrak atau validasi yang salah.
6. Tambahkan pengujian untuk upload valid dan tidak valid.
7. Verifikasi dokumen tersimpan dengan benar.
8. Identifikasi endpoint daftar/detail dokumen yang benar-benar tersedia.
9. Pastikan dokumen yang tersimpan dapat dimuat kembali setelah refresh.
10. Uji kembali alur upload → parsing → extracted fields → verifikasi → evaluasi checklist.

**Output yang harus diserahkan agent:**
- Akar masalah beserta bukti.
- File yang diubah dan alasan.
- Ringkasan perubahan.
- Test yang dijalankan beserta hasil aktual.
- Risiko atau batasan yang tersisa.
- Satu rekomendasi task berikutnya.

Jangan memulai pengembangan fitur AI baru sebelum task pertama selesai dan diverifikasi.
