# Panduan penggunaan

## Sebelum permohonan dikirim

Admin menyiapkan akun dan checklist di aplikasi. Buat versi checklist untuk kombinasi **payer + jenis layanan**, masukkan aturan beserta field yang harus dicari, lalu pilih **Aktifkan**. Versi aktif dipasangkan otomatis ke permohonan yang cocok. Jika belum ada checklist yang cocok, Staff tetap dapat menyimpan draft; tombol kirim akan menjelaskan bahwa Admin perlu mengaktifkan checklist.

## Alur Staff

1. Buka **PA Requests** lalu buat permohonan. Isi payer, jenis layanan, referensi pasien, diagnosis, dan NPI penyedia bila tersedia. NPI adalah 10 digit; gunakan tombol pencarian untuk mencocokkan data CMS. NPI tidak menggantikan persyaratan payer.
2. Buka halaman permohonan dan unggah dokumen CSV (`field,value`) atau PDF berbasis teks. PDF hasil scan belum didukung.
3. Jalankan parsing. AI membantu mengekstrak field dari PDF; CSV diproses dengan aturan tetap. Periksa bukti sumber dan koreksi nilai bila perlu. AI tidak menentukan kelayakan atau keputusan payer.
4. Pilih **Kirim untuk review**. Permohonan masuk ke antrian Reviewer setelah dokumen berhasil diproses dan checklist aktif ditemukan.
5. Jika Reviewer meminta informasi, status menjadi **Perlu informasi**. Lengkapi data/dokumen sesuai catatan di riwayat permohonan, lalu kirim ulang.
6. Setelah Reviewer menyelesaikan pemeriksaan, status menjadi **Siap diajukan**. Staff mengajukan ke payer di luar aplikasi, lalu pilih **Catat pengajuan ke payer** dan masukkan nomor referensi bila ada.
7. Saat keputusan payer diterima, pilih **Catat keputusan payer**, pilih outcome, dan tulis catatan. Status selesai dan outcome disimpan terpisah.

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

- CSV memerlukan header `field,value`; PDF harus berupa PDF berbasis teks.
- Ukuran maksimum file adalah 10 MiB.
- Sebelum mengirim teks PDF ke OpenAI, Staff harus mengonfirmasi bahwa data sintetis atau sudah di-de-identifikasi.
- Hasil AI perlu diperiksa manusia dan bukan rekomendasi persetujuan medis. Status checklist menunjukkan kecocokan field dengan aturan yang dikonfigurasi, bukan keputusan klinis atau keputusan payer.
- Backend menerapkan migrasi database yang belum tercatat saat mulai. Jika migrasi gagal, backend berhenti dengan pesan kesalahan yang menyebutkan versi migrasi.
