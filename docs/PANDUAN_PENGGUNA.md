# Panduan penggunaan

## Sebelum permohonan dikirim

Admin menyiapkan akun dan checklist di aplikasi. Buat versi checklist untuk kombinasi **payer + jenis layanan**, masukkan aturan beserta field yang harus dicari, lalu pilih **Aktifkan**. Versi aktif dipasangkan otomatis ke permohonan yang cocok. Jika belum ada checklist yang cocok, Staff tetap dapat menyimpan draft; tombol kirim akan menjelaskan bahwa Admin perlu mengaktifkan checklist.

## Alur Staff

1. Buka **PA Requests** lalu buat permohonan. Isi payer, jenis layanan, referensi pasien, diagnosis, dan NPI penyedia bila tersedia. NPI adalah 10 digit; gunakan tombol pencarian untuk mencocokkan data CMS. NPI tidak menggantikan persyaratan payer.
2. Buka halaman permohonan dan unggah CSV bertabel atau PDF berbasis teks. Parser membaca pasangan label dan nilai dari baris berisi `:` atau `=`, serta nilai dari kolom tabel. PDF hasil scan belum didukung.
3. Jalankan parsing. Parser membaca pasangan label-nilai secara lokal; AI dapat membantu menemukan field semantik pada PDF. Periksa bukti sumber dan koreksi nilai bila perlu. AI tidak menentukan kelayakan atau keputusan payer.
4. Pilih **Kirim untuk review**. Permohonan masuk ke antrian Reviewer setelah dokumen berhasil diproses dan checklist aktif ditemukan.
5. Jika Reviewer meminta informasi, status menjadi **Perlu informasi**. Lengkapi data/dokumen sesuai catatan di riwayat permohonan, lalu kirim ulang.
6. Setelah Reviewer menyelesaikan pemeriksaan, status menjadi **Siap diajukan**. Staff mengajukan ke payer di luar aplikasi, lalu mencatat kanal pengajuan dan nomor referensi bila ada.
7. Jika payer meminta informasi, catat permintaan, lengkapi dokumen, lalu kirim tambahan untuk review internal sebelum mengajukannya kembali.
8. Saat keputusan payer diterima, pilih outcome dan tulis alasannya. Untuk keputusan ditolak atau disetujui sebagian, Staff dapat mencatat pengajuan banding dan responsnya.

Draft dapat diubah atau dibatalkan oleh Staff pemilik. Setelah dikirim, perubahan dokumen dan data hanya dibuka kembali bila Reviewer meminta informasi.

## Alur Reviewer

1. Buka **PA Requests**. Daftar awal menampilkan permohonan berstatus **Siap direview**.
2. Buka kasus dan pilih **Mulai review**. Status berubah menjadi **Dalam review**.
3. Periksa dokumen dan bukti ekstraksi. Koreksi field atau tetapkan status verifikasi pada tahap **Verifikasi field**. Evaluasi checklist dilakukan saat status Dalam review.
4. Jika ada kekurangan, pilih **Minta informasi** dan tulis hal yang harus dilengkapi. Kasus kembali ke Staff dengan status **Perlu informasi**.
5. Jika semua field wajib sudah terverifikasi dan checklist terpenuhi, pilih **Selesaikan review**. Kasus berpindah ke **Siap diajukan** untuk ditindaklanjuti Staff.

Reviewer memeriksa kelengkapan dan bukti; keputusan akhir tetap berasal dari payer.

## Alur Admin

- **Kelola user**: buat akun, tetapkan role Admin/Staff/Reviewer, atau ubah status akun.
- **Kelola checklist**: buat versi aturan, periksa pasangan payer/layanan dan field, kemudian aktifkan. Aktivasi menonaktifkan versi sebelumnya untuk pasangan yang sama.
- Admin dapat melihat seluruh permohonan dan riwayat status untuk pemantauan. Admin tidak mengambil alih pengisian Staff atau verifikasi klinis Reviewer.

## Arti status

| Status | Penanggung jawab berikutnya | Makna |
| --- | --- | --- |
| Draft | Staff | Permohonan sedang disiapkan. |
| Siap direview | Reviewer | Dokumen siap dan permohonan masuk antrian. |
| Dalam review | Reviewer | Field dan checklist sedang diperiksa. |
| Perlu informasi | Staff | Reviewer meminta data atau dokumen tambahan. |
| Siap diajukan | Staff | Review selesai; Staff mengirim permohonan ke payer di luar aplikasi. |
| Diajukan | Staff | Pengajuan ke payer telah dicatat; menunggu keputusan. |
| Selesai | Tidak ada | Keputusan payer dan catatannya telah disimpan. |
| Dibatalkan | Tidak ada | Staff membatalkan draft atau permohonan yang dikembalikan. |

## File dan privasi

- CSV dapat memakai nama header yang berbeda. Dua kolom pertama dibaca sebagai label dan nilai; tabel dengan tiga kolom atau lebih memakai header sebagai nama field. PDF berbasis teks juga dibaca untuk pasangan label dan nilai yang dipisahkan `:` atau `=`. PDF hasil scan belum didukung.
- Ukuran maksimum file adalah 10 MiB.
- Sebelum mengirim teks PDF ke OpenAI, Staff harus mengonfirmasi bahwa data sintetis atau sudah di-de-identifikasi.
- Hasil AI perlu diperiksa manusia dan bukan rekomendasi persetujuan medis. Status checklist menunjukkan kecocokan field dengan aturan yang dikonfigurasi, bukan keputusan klinis atau keputusan payer.
- Pengajuan, jawaban tambahan, dan banding berlangsung di kanal payer di luar aplikasi. Proses dan tenggat commercial plan berbeda menurut produk dan yurisdiksi; validasi aturan payer sebelum memakai checklist dalam operasi nyata.
- Backend menerapkan migrasi database yang belum tercatat saat mulai. Jika migrasi gagal, backend berhenti dengan pesan kesalahan yang menyebutkan versi migrasi.
