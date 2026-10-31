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

## Intake WhatsApp

Admin atau Staff dapat membuka **WhatsApp Staff**, memasukkan nomor Staff, lalu membuat QR pairing. Pindai QR dari menu **Perangkat tertaut** di WhatsApp. Setiap akun aplikasi dapat memasangkan satu device. Device yang sudah ada di akun Fonnte akan dipakai kembali jika nomornya cocok. Batas device dan biaya mengikuti paket Fonnte. Jika device dihapus langsung melalui Fonnte, hapus juga catatan pairing di aplikasi melalui tombol **Hapus catatan pairing**, lalu buat pairing baru. Tindakan tersebut hanya menghapus catatan di aplikasi, bukan device Fonnte.

Atur `FONNTE_WEBHOOK_URL`, `FRONTEND_PUBLIC_URL`, `VITE_APP_NAME`, dan `FONNTE_ACCOUNT_TOKEN` di `backend/.env`. `FONNTE_ACCOUNT_TOKEN` adalah account token dari Settings Fonnte untuk mengelola device, bukan token device. `FONNTE_WEBHOOK_URL` harus berisi URL publik backend yang dapat menerima webhook, misalnya `https://domain-backend-anda/api/webhooks/whatsapp?secret=<secret-acak-minimal-32-karakter>`. Secret berada di query URL ini, bukan di variabel tambahan. Gunakan URL yang sama pada konfigurasi webhook Fonnte. Jangan mengganti secret setelah pairing karena backend memakainya untuk memvalidasi webhook dan membuka token device terenkripsi. Jaga agar URL lengkap tidak tercatat di log proxy.

`FRONTEND_PUBLIC_URL` harus berisi URL publik aplikasi Next.js, misalnya `https://domain-frontend-anda`. URL ini berbeda dari `FONNTE_WEBHOOK_URL` dan `BACKEND_API_URL`. Setelah Staff menyetujui ringkasan dengan **YA**, backend menyimpan draft beserta riwayat status dalam transaksi database, lalu mengirim link `/upload-user` dari domain frontend hanya setelah penyimpanan berhasil. Pastikan frontend dapat diakses oleh pengguna WhatsApp. Jika URL frontend kosong atau tidak valid, backend tidak membuat draft.

Untuk pengujian lokal, jalankan backend pada port `8081` dan biarkan tunnel publik tetap hidup dengan `ngrok http 8081`. Ganti host pada `FONNTE_WEBHOOK_URL` dengan URL HTTPS forwarding yang diberikan ngrok, restart backend, lalu tekan **Perbarui webhook Fonnte** pada halaman WhatsApp. Jika URL forwarding berubah, ulangi langkah ini. Buka `<host-forwarding>/api/health/ready` untuk memastikan tunnel mencapai backend sebelum mengirim chat. Status device **Terhubung** hanya menunjukkan WhatsApp tertaut ke Fonnte; status itu tidak memastikan tunnel atau webhook aktif.

Jika catatan pairing tidak dapat dihapus dari halaman, restart backend agar route `DELETE /api/whatsapp/device-pairing` aktif, lalu coba lagi. Penghapusan hanya membuang catatan pairing aplikasi, tidak menghapus device Fonnte. Jika masih gagal, periksa log backend untuk detail kegagalan database.

Setelah pairing terhubung dan webhook aktif, balas **BANTUAN** untuk melihat petunjuk. Balas **SETUJU** untuk mulai membuat PA. Chatbot menanyakan payer, lalu pengguna dapat menceritakan keluhan/gejala dan diagnosis yang diketahui. AI boleh mengusulkan pemeriksaan atau prosedur sebagai bahan draft, tetapi usulan itu bukan keputusan medis dan Staff harus memeriksanya sebelum melanjutkan PA. AI tidak mengusulkan obat berdasarkan gejala; pengguna hanya dapat memasukkan obat yang memang diminta dokter. Jika AI tidak dapat mengusulkan layanan, pengguna dapat menyebut layanan yang dokter minta atau membalas **LANJUTKAN** agar Staff menentukan layanan setelah memeriksa draft. Sistem membuat ID pasien sintetis, jadi pengguna tidak perlu mengirim nomor rekam medis. Jangan kirim nama lengkap, NIK, alamat, atau tanggal lahir. Jika checklist aktif mewajibkan NPI provider, chatbot meminta NPI 10 digit sebelum ringkasan. Balas **YA** untuk membuat draft agar diperiksa Staff atau **TIDAK** untuk membatalkan; kirim koreksi bila ada data yang perlu diperbaiki. Setelah draft dibuat, nomor WhatsApp yang sama dapat memulai permohonan lain dengan membalas **MULAI** atau **BUAT PA BARU**, lalu menyetujui penggunaan data lagi dengan **SETUJU**. Draft sebelumnya tetap tersimpan sebagai permohonan tersendiri. OpenAI memproses jawaban sesuai `OPENAI_API_KEY` dan `OPENAI_MODEL` yang sudah digunakan aplikasi. Nomor pengirim dipakai untuk melanjutkan percakapan WhatsApp, tetapi tidak dikirim ke OpenAI dan tidak disimpan sebagai nomor kontak notifikasi pada permohonan PA. Draft masuk ke akun Staff yang memasangkan device, dan chatbot mengirim tautan upload frontend setelah database berhasil menyimpan draft. Form upload tidak meminta nomor telepon. Staff dapat melihat status proses dan penyebab umum kegagalan OpenAI pada panel Aktivitas chatbot. Webhook mencatat status HTTP di log backend tanpa isi pesan atau secret URL. Jika aktivitas tetap kosong, pastikan tunnel dan URL webhook Fonnte mengarah ke backend. Staff tetap dapat membuat atau mengubah permohonan secara manual dan harus memeriksa draft sebelum melanjutkan proses. Migration `013_remove_whatsapp_notifications.sql` menghapus nomor dan persetujuan notifikasi yang tersimpan dari skema PA.

Jika AI gagal memproses pesan, pengguna dapat mencoba lagi atau melanjutkan pengisian dengan Staff. Jangan kirim pengenal pribadi seperti nama lengkap, NIK, alamat, tanggal lahir, atau nomor rekam medis. Tautan upload memakai token akses khusus, jadi kirim hanya kepada pihak yang perlu mengunggah dokumen.

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
