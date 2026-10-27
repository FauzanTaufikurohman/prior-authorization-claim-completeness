# Panduan Field Pengguna dan Demo Aplikasi

Panduan ini menjelaskan isian yang benar-benar dilihat pengguna di aplikasi dan skenario demo memakai data sintetis. PDF sample AI terpisah: `output/pdf/pa-ai-synthetic-test.pdf`.

## Perbedaan role

- **Admin** menyiapkan akun dan checklist pasangan payer + layanan, lalu memantau seluruh request.
- **Staff** membuat request, mengunggah dokumen, memeriksa hasil parsing, mengirim ke Reviewer, lalu mencatat pengajuan eksternal dan keputusan payer.
- **Reviewer** memeriksa antrian, mengoreksi/menandai field, mengevaluasi checklist, lalu meminta informasi atau menyelesaikan review.

## Field login dan registrasi

| Form | Field | Fungsi |
|---|---|---|
| Login | Email | Identitas akun yang sudah terdaftar. |
| Login | Password | Membuka sesi akun; sesi berlaku 12 jam. |
| Register | Nama lengkap | Nama pengguna yang akan ditampilkan pada aktivitas. |
| Register | Email | Alamat akun; harus valid dan belum terdaftar. |
| Register | Password | Kata sandi awal; minimal 12 dan maksimal 128 karakter. |
| Register | Ulangi password | Memastikan password pertama tidak salah ketik. |

Register publik selalu menghasilkan akun Staff. Role Reviewer/Admin diatur Admin.

## Field Staff: membuat permohonan

| Field | Wajib? | Fungsi/panduan isi |
|---|---|---|
| Payer dan layanan | Ya | Pilih pasangan dari checklist yang diaktifkan Admin. Pasangan ini menentukan aturan checklist yang berlaku. Opsi manual hanya untuk menyimpan draft; tidak bisa dikirim sampai checklist cocok disiapkan Admin. |
| Nama payer | Ya jika opsi manual | Nama payer/health plan yang akan memproses permohonan. Ejaan perlu konsisten dengan checklist. |
| Jenis layanan | Ya jika opsi manual | Prosedur/layanan/obat yang dimintakan otorisasi; harus cocok dengan checklist. |
| Patient reference / Referensi pasien | Ya | ID internal atau identifier untuk menemukan record pasien. Untuk demo pakai ID sintetis; hindari data identitas nyata. |
| Provider NPI | Tidak | Nomor identitas penyedia 10 digit. Tombol Cari mencocokkan ke CMS Registry dan membantu menampilkan nama/alamat/taksonomi. Lookup bukan validasi lisensi, coverage, atau kelayakan PA; input manual dapat digunakan. |
| Diagnosis / kode dari dokumen | Ya | Salin diagnosis/kode dari dokumen pendukung. Jangan menebak. Untuk demo gunakan label sintetis. |
| Isi data uji otomatis | Tombol | Mengisi contoh sintetis pada form agar pengguna dapat mencoba workflow tanpa pengetahuan klinis. Tetap periksa isi sebelum menyimpan. |

## Field Staff: dokumen dan hasil AI

| Field/tindakan | Fungsi |
|---|---|
| File dokumen | Unggah PDF berbasis teks atau CSV maksimal 10 MiB. PDF hasil scan belum didukung OCR. |
| Konfirmasi data sintetis/de-identifikasi | Untuk PDF, menyatakan teks yang dikirim ke OpenAI tidak memuat data pasien yang dapat dikenali. Wajib sebelum parsing PDF. |
| Jalankan parsing | PDF: parser membaca pasangan label-nilai (`:` atau `=`) dan tabel sederhana, lalu AI dapat melengkapi field semantik checklist. CSV diproses lokal; dua kolom dianggap label/nilai, sedangkan tabel lebih lebar memakai header kolom sebagai nama field. |
| Field value | Nilai hasil ekstraksi. Staff dapat membetulkan/melengkapi saat Draft atau Perlu informasi. |
| Evidence text | Kutipan sumber yang dipakai untuk menunjukkan asal nilai; periksa kecocokan dengan dokumen. |
| Confidence | Perkiraan keyakinan model (0 sampai 1), bukan jaminan kebenaran. |
| Simpan perubahan | Menyimpan koreksi Staff. Hasil AI tetap harus diverifikasi Reviewer. |

Contoh key yang tersedia untuk aturan checklist: `patient_name`, `patient_reference`, `payer_name`, `service_type`, `provider_npi`, `diagnosis`, `clinical_notes`, `prior_treatments`.

## Field Admin: kelola user

| Field | Fungsi |
|---|---|
| Nama lengkap | Nama tampilan akun. |
| Email | Username/login dan identitas unik. |
| Password awal | Password untuk akun baru, minimal 12 karakter. Berikan melalui kanal aman kepada pemilik akun. |
| Role | `Staff`, `Reviewer`, atau `Admin`; menentukan menu dan kewenangan. |
| Role di daftar akun | Mengubah role user yang sudah ada; perlu klik Simpan role. |
| Aktif/Nonaktif | Mengunci atau membuka akses akun. Admin tidak dapat menonaktifkan atau menurunkan role akunnya sendiri. |

## Field Admin: kelola checklist

| Field | Fungsi |
|---|---|
| Nama checklist | Nama yang mudah dikenali Admin/Reviewer. |
| Payer | Nama payer yang dicakup oleh aturan ini. Harus sama dengan pilihan Staff. |
| Jenis layanan | Layanan yang dicakup. Dikombinasikan dengan payer untuk memasangkan versi. |
| Nomor versi | Nomor versi positif; payer + layanan + versi harus unik. |
| Tanggal berlaku | Tanggal mulai berlaku (opsional). |
| Sumber aturan | URL sumber kebijakan payer (opsional). Gunakan dokumen resmi untuk penerapan nyata. |
| Data yang harus diperiksa | Pilih field dari dropdown yang harus ditemukan pada dokumen; pilihan ini menjadi key ekstraksi AI/CSV. |
| Deskripsi | Penjelasan yang dibaca manusia tentang aturan pemeriksaan. |
| Tambah aturan | Menambah satu rule field lagi. Field wajib setidaknya satu untuk mengaktifkan checklist. |
| Simpan sebagai draft / Aktifkan | Simpan belum membuatnya dipakai Staff. Klik Aktifkan setelah memeriksa payer, layanan, versi, dan rules. Aktivasi menggantikan versi aktif lain untuk pasangan yang sama. |

## Field Reviewer: verifikasi dan keputusan review

| Field/tindakan | Fungsi |
|---|---|
| Mulai review | Mengambil item `Siap direview`; status menjadi `Dalam review`. |
| Field value | Nilai yang terbaca dari dokumen. Reviewer dapat memperbaikinya jika AI salah/kurang. |
| Verification status | `pending`: belum diperiksa; `verified`: nilai sesuai sumber; `corrected`: dikoreksi dan dikonfirmasi; `uncertain`: masih tidak yakin/perlu klarifikasi. Hanya verified/corrected dengan nilai terisi membuat rule wajib fulfilled. |
| Evaluasi checklist | Membandingkan field hasil ekstraksi dengan required field checklist. Hasil: fulfilled, missing, atau needs review. Ini pemeriksaan kelengkapan, bukan keputusan medis/payer. |
| Catatan Minta informasi | Jelaskan dengan spesifik data/dokumen yang perlu Staff lengkapi. Permohonan kembali ke Staff. |
| Selesaikan review | Tersedia jika semua field wajib terisi dan diverifikasi serta hasil checklist fulfilled. Mengubah status ke Siap diajukan. |

## Field Staff: setelah review

| Field/tindakan | Fungsi |
|---|---|
| Catat pengajuan ke payer | Pilih kanal dan catat setelah Staff benar-benar mengirim permohonan. Aplikasi tidak mengirimkannya otomatis. |
| Nomor referensi payer | Nomor yang diberikan payer setelah menerima pengajuan; opsional. |
| Outcome keputusan payer | Pilih `approved`, `denied`, `partially_approved`, atau `withdrawn` sesuai jawaban payer. |
| Catatan keputusan payer | Ringkasan hasil/alasan/tanggal penting; wajib saat mencatat keputusan. Setelah tersimpan status menjadi Selesai. |

## Skenario presentasi demo end-to-end

Gunakan hanya data sintetis. Untuk menguji parsing AI PDF, OpenAI key harus tersedia pada environment backend. Jika tidak ingin memanggil OpenAI, gunakan CSV sample; CSV memeriksa alur parsing namun tidak mendemonstrasikan AI.

### A. Admin menyiapkan checklist demo

- Nama: `Demo PA Checklist`
- Payer: `SYNTHETIC_TEST_PAYER`
- Jenis layanan: `SYNTHETIC_TEST_SERVICE`
- Nomor versi: `1`
- Rules wajib: `diagnosis`, `clinical_notes`, `prior_treatments`
- Deskripsi: `Field tercantum di dokumen uji sintetis`
- Simpan sebagai draft, lalu aktifkan.

Nilai payer dan layanan ini sengaja sama dengan PDF sample. Jika memakai opsi lain, sesuaikan pasangan di checklist dan request.

### B. Staff membuat request dan menjalankan AI

- Payer/layanan: pilih `SYNTHETIC_TEST_PAYER · SYNTHETIC_TEST_SERVICE`.
- Referensi pasien: `SYN-PA-2026-001`.
- Provider NPI: kosongkan (field opsional).
- Diagnosis: `SAMPLE-DX-001 - synthetic extraction label only`.
- Unggah `output/pdf/pa-ai-synthetic-test.pdf`.
- Centang konfirmasi data sintetis, lalu jalankan parsing.
- Tunjukkan hasil AI, evidence, confidence, serta status pending; jelaskan bahwa ini ekstraksi, bukan keputusan.
- Kirim untuk review.

### C. Reviewer memeriksa

- Masuk sebagai Reviewer, buka request yang siap direview, klik Mulai review.
- Pada Verifikasi field, periksa nilai/evidence untuk diagnosis, clinical_notes, prior_treatments. Untuk fixture ini tandai verified (atau corrected bila memang diedit), bukan uncertain.
- Jalankan Evaluasi checklist. Pastikan tiga rules fulfilled, lalu Selesaikan review.

### D. Staff mencatat proses eksternal

- Masuk kembali sebagai Staff. Klik Catat pengajuan ke payer. Untuk demo gunakan referensi `DEMO-REF-001` sebagai label simulasi.
- Klik Catat keputusan payer, pilih outcome demo, misalnya `approved`, dan catatan: `SIMULASI DEMO - bukan keputusan payer nyata`.
- Jelaskan status selesai hanya berarti outcome simulasi dicatat di aplikasi. Dalam penggunaan nyata, Staff harus mengisi hasil yang benar-benar diterima dari payer.

## Arti status yang tampil

`Draft` → `Siap direview` → `Dalam review` → (opsional `Perlu informasi` lalu dikirim ulang) → `Siap diajukan` → `Diajukan` → `Selesai`. Status `Dibatalkan` dapat dipilih Staff dari draft/permohonan yang dikembalikan.

Reviewer menyelesaikan pemeriksaan kelengkapan. Keputusan klinis dan keputusan payer tidak dibuat oleh AI.
