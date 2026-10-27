# Analisis Adaptasi Prior Authorization Assistant untuk Indonesia

Tanggal analisis: 27 Oktober 2026  
Tujuan: memetakan perubahan produk yang diperlukan sebelum mengklaim aplikasi sesuai untuk alur Indonesia. Dokumen ini adalah analisis awal, bukan checklist klinis, legal opinion, atau spesifikasi integrasi BPJS/SATUSEHAT yang telah disetujui.

## Kesimpulan utama

Proposal versi 2.0 secara eksplisit memilih klinik/praktik Amerika Serikat sebagai pasar awal. Proyek yang ada mengikuti asumsi tersebut: identitas provider memakai NPI dan CMS Registry, konteks permohonan memakai payer + service type, contoh field dan checklist berbahasa Inggris, serta tahap submission/keputusan dicatat setelah proses eksternal. CSV hanya diterima dalam format `field,value`; PDF harus berisi teks yang bisa diekstrak. Karena itu, hipotesis pengguna bahwa perlu menerima PDF/CSV Indonesia masuk akal sebagai arah investigasi, tetapi belum terbukti menjadi kebutuhan tunggal atau cukup untuk adaptasi.

Indonesia perlu diperlakukan sebagai beberapa jalur kerja yang berbeda. JKN/BPJS Kesehatan memiliki proses peserta, rujukan berjenjang, FKTP/FKRTL, dan SEP. Asuransi komersial dapat memiliki formulir, manfaat, kanal, dan aturan masing-masing. Kebutuhan klinik juga bisa berupa menyiapkan lampiran atau data klaim/rujukan, bukan “prior authorization” model AS. Jangan memakai label “PA Indonesia” sebagai satu proses seragam sebelum diwawancarai.

Rekomendasi produk: **conditional go untuk riset dan prototype lokal**, dengan satu alur sempit (misalnya administrasi rujukan/pra-verifikasi manfaat untuk satu tipe fasilitas dan satu pembayar), bukan langsung porting semua checklist payer AS. Minta mentor domain kesehatan atau calon pengguna memeriksa workflow, contoh dokumen yang boleh digunakan, serta daftar field sebelum mengunci skema.

## 1. Temuan proposal dan proyek saat ini

| Area | Yang dinyatakan/dilakukan sekarang | Dampak untuk Indonesia |
|---|---|---|
| Pasar dan persona | Proposal menarget klinik/praktik medis AS, staf PA/billing, practice manager, dan payer AS. | Persona, proses, volume, pembeli, dan nilai bisnis perlu divalidasi ulang di fasyankes Indonesia. |
| Entitas permohonan | `payer_name`, `service_type`, `provider_npi`, `patient_reference`, `diagnosis`, checklist version. | Belum ada model eksplisit untuk skema penjamin/JKN, FKTP/FKRTL, rujukan, SEP, kunjungan, poli, atau manfaat/polis komersial. |
| Provider | NPI 10 digit, lookup ke CMS NPPES. | NPI/CMS Registry adalah ketergantungan AS; identitas Indonesia perlu berangkat dari master data fasilitas dan tenaga kesehatan yang berlaku (contohnya IHS/SATUSEHAT bila use case dan aksesnya memerlukan). Jangan mengganti NPI dengan satu nomor tanpa verifikasi domain. |
| Field ekstraksi bawaan | `patient_name`, `patient_reference`, `payer_name`, `service_type`, `provider_npi`, `diagnosis`, `clinical_notes`, `prior_treatments`. | Terlalu umum dan berbahasa Inggris; belum merepresentasikan konteks rujukan, penjamin, kunjungan, dokumen pendukung, dan kode/terminologi yang dipakai lokal. |
| Checklist | Berversi per pasangan payer + service; sumber URL/tanggal berlaku/reviewer opsional. Kelengkapan berbasis field key. | Perlu ruang lingkup per program/produk, jenjang fasilitas, tipe pelayanan, formulir/kanal, versi, masa berlaku, wilayah bila relevan, dan bukti sumber. Beberapa aturan mungkin perlu memeriksa dokumen atau kombinasi field, tidak hanya ada/tidaknya satu field. |
| Upload | PDF teks atau CSV, maks 10 MiB; PDF scan/OCR belum didukung. Parser kini menerima CSV dua kolom label/nilai dengan nama header beragam, tabel CSV dengan header kolom, serta pasangan `:`/`=` dan tabel sederhana dari PDF teks. | Dokumen operasional Indonesia mungkin scan/foto/formulir hasil ekspor; bentuk dan kualitasnya harus diuji. Ekstraksi fleksibel belum memvalidasi arti field terhadap aturan lokal. |
| AI dan data | Teks PDF dikirim ke OpenAI setelah konfirmasi data sintetis/de-identifikasi. CSV diparse lokal. AI hanya ekstraktor; wajib review manusia. | Untuk data kesehatan teridentifikasi, perlu keputusan organisasi mengenai dasar pemrosesan, peran pengendali/prosesor, vendor/subprosesor, lokasi/transfer, retensi, akses, dan mitigasi. Checkbox de-identifikasi bukan pengganti tata kelola. |
| Alur hasil | Staff → reviewer internal → Staff mengirim ke payer di luar aplikasi → catat nomor dan keputusan payer. | Status “siap diajukan” tidak boleh mengesankan integrasi/otorisasi BPJS. Bedakan draft administrasi, rujukan, SEP, verifikasi manfaat, pengajuan komersial, klaim, dan keputusan eksternal sesuai alur tervalidasi. |
| Data dan audit | Dokumen, nilai ekstraksi, confidence/evidence, status verifikasi, status history, audit log tersedia. | Perlu audit siapa melihat/mengubah/mengekspor/mengirim data, sumber dokumen per nilai, waktu dan koreksi, serta aturan akses dan retensi untuk data kesehatan nyata. |

Rujukan kode yang telah diperiksa: `backend/internal/models/pa_request.go`, `backend/internal/services/{csv_parser.go,pdf_parser.go,document_validation.go,openai_extractor.go}`, `backend/internal/handlers/pa_workflow_handler.go`, migrasi `001_init.sql`, `004_pa_workflow.sql`, `005_checklist_audit.sql`, `006_pa_workflow_repair.sql`, dan panduan `docs/PANDUAN_FIELD_DAN_DEMO.md` serta `docs/RANGKUMAN_APLIKASI.md`.

## 2. Bedakan workflow sebelum mengubah aplikasi

### A. JKN/BPJS Kesehatan

Dokumentasi BPJS yang tersedia untuk peserta menjelaskan rawat jalan tingkat lanjutan sebagai kedatangan peserta ke FKRTL dengan identitas peserta aktif, identitas tambahan bila diperlukan, serta surat rujukan (kecuali gawat darurat); rumah sakit menerbitkan SEP untuk memperoleh pelayanan. Ada juga konsul internal/eksternal, kontrol ulang, dan surat rujuk balik. Ini cukup untuk menunjukkan bahwa asumsi “staf mengajukan satu PA ke payer dan menunggu approval” tidak otomatis cocok dengan setiap kunjungan JKN. Rincian administrasi dan sistem operasional yang berlaku harus dikonfirmasi ke fasyankes/BPJS untuk skenario terpilih; leaflet peserta bukan spesifikasi API atau checklist klaim.

Potensi alat bantu yang perlu diuji: memeriksa keberadaan/konsistensi dokumen dan informasi sebelum rujukan/kunjungan/pengajuan administrasi, menampilkan field yang hilang dengan rujukan ke halaman/bukti, atau menyiapkan ringkasan kerja untuk petugas. Jangan mengotomatisasi eligibility, indikasi klinis, penerbitan SEP, keputusan penjamin, maupun pengiriman klaim tanpa proses dan otorisasi resmi.

### B. Asuransi kesehatan komersial

Jangan mengasumsikan satu standar lintas perusahaan. Perlu wawancara dengan provider dan/atau perusahaan asuransi/TPA untuk mengetahui apakah yang dimaksud pra-otorisasi, penjaminan cashless, surat jaminan, verifikasi manfaat, klaim reimbursement, atau alur lain. Formulir, data anggota/polis, kriteria manfaat, dokumen pendukung, kanal pengiriman, nomor kasus, status, SLA, dan alasan penolakan berpotensi berbeda per produk dan layanan. Checklist harus memiliki sumber resmi dan pemilik pembaruan.

### C. Pertanyaan penentu scope

1. Siapa pengguna pertama: klinik, rumah sakit, laboratorium, pasien, penjamin, atau TPA?
2. Apakah kasus pertama JKN/BPJS atau asuransi komersial tertentu? Rawat jalan, rawat inap, obat, tindakan, atau rujukan?
3. Langkah mana yang benar-benar berulang dan memakan waktu? Apa keluaran kerja yang diterima sistem berikutnya?
4. Dokumen dan data mana yang boleh diakses, dan apakah data contoh bisa disediakan secara sintetis/de-identifikasi?
5. Checklist siapa yang menyusun, menyetujui, memberi tanggal berlaku, dan memberitahu perubahan?
6. Apakah ada API/integrasi yang disetujui? Jika tidak, apakah produk cukup sebagai workspace dan ekspor manual?

## 3. Kandidat data dan field untuk diverifikasi

Ini adalah **daftar discovery**, bukan seluruh field wajib untuk semua pasien. Kumpulkan hanya data yang dibutuhkan untuk use case dan sumbernya; hindari mengumpulkan NIK, alamat, nama, atau diagnosis pasien secara default bila ID lokal/de-identifikasi cukup.

| Kelompok | Kandidat field | Catatan validasi |
|---|---|---|
| Organisasi pemohon | nama/kode fasilitas, jenis fasilitas, alamat kerja bila dibutuhkan, ID Organization/IHS SATUSEHAT bila relevan | Tentukan sistem sumber (SIMRS/SIMPUS/RME), kode resmi yang dipakai, serta apakah fasilitas sudah onboarding SATUSEHAT. |
| Petugas | nama/peran pengisi, tenaga kesehatan pengirim/penanggung jawab, identifier profesi/IHS bila relevan | Jangan gunakan NPI. Verifikasi jenis identifier dan hak akses dari workflow dan master data Indonesia. |
| Pasien/peserta | ID internal/MRN, nomor kepesertaan JKN atau ID anggota/polis komersial, status aktif/masa berlaku bila tersedia; nama/tanggal lahir hanya jika diperlukan | Identitas dan data kesehatan adalah data berisiko tinggi. NIK hanya bila kebutuhan hukum dan operasional jelas, dengan akses/minimisasi terkontrol. Kepesertaan/eligibility bukan kesimpulan AI. |
| Penjamin/skema | JKN/BPJS Kesehatan atau nama perusahaan asuransi/TPA, program/produk, nomor polis/anggota, kelas/manfaat bila relevan | Pisahkan payer, program, policy/member, dan pihak pengelola. Validasi apakah data boleh diverifikasi manual/API. |
| Rute pelayanan | FKTP/FKRTL, tingkat/jenis layanan, rawat jalan/rawat inap/IGD, tujuan rujukan, poli/spesialis, tanggal kunjungan/rencana layanan | Aturan emergensi, konsul, kontrol, rujuk balik dan wilayah/jejaring dapat mengubah dokumen yang dibutuhkan. |
| Referensi transaksi | nomor surat rujukan dan tanggal/masa berlaku, nomor SEP dan tanggal (jika sudah diterbitkan), nomor kasus/otorisasi/penjaminan, nomor klaim bila use case terkait | Bedakan dokumen prasyarat dari hasil proses. Jangan menandai SEP/approval tersedia bila belum diterbitkan. |
| Klinis | diagnosis utama/sekunder beserta kode dan sistem terminologi, ringkasan klinis, indikasi/tujuan layanan, temuan pemeriksaan, riwayat terapi dan respons, alergi/obat bila relevan | Petugas klinis bertanggung jawab atas isi. AI tidak menentukan diagnosis, kebutuhan medis, coverage, atau kelayakan. Konfirmasi terminologi dan kode yang diwajibkan tiap use case. |
| Layanan diminta | nama tindakan/obat/pemeriksaan, kode lokal/standar yang digunakan, lokasi/setting layanan, tanggal/kuantitas/durasi, urgensi | Tidak ada asumsi satu kode “procedure_code” yang universal. Validasi kode, format, dan kamus dari pemilik proses. |
| Lampiran | surat rujukan/konsul, resume/ringkasan medis, hasil laboratorium/radiologi, resep/rencana tindakan, formulir penjamin, hasil terapi sebelumnya, bukti lain | Checklist harus menyatakan alasan, dokumen yang diterima, periode dokumen, apakah wajib/bersyarat, dan sumber resmi. OCR dan kualitas scan perlu dinilai dengan berkas yang diizinkan. |
| Tata kelola | asal sistem/dokumen, uploader, waktu, versi dokumen, persetujuan/otorisasi yang relevan, label identifikasi/de-identifikasi, retensi, akses, audit | Catat provenance per extracted field dan per hasil checklist. Persyaratan persetujuan/dasar pemrosesan perlu penelaahan privacy/legal institusi. |
| Hasil proses eksternal | status yang benar-benar diberi sistem/penjamin, timestamp, referensi, permintaan tambahan, hasil dan alasan seperti yang diterima | Simpan sumber dan siapa yang mencatatnya. Taksonomi status/hasil harus berasal dari sistem target, bukan asumsi `approved/denied` model AS. |

Terminologi awal untuk investigasi, bukan keputusan skema: identitas pasien IHS/MRN SATUSEHAT bila diperlukan; identitas organisasi/lokasi dan tenaga kesehatan sesuai onboarding; ICD-10/ICD-9-CM/LOINC/SNOMED CT hanya sesuai resource dan playbook yang relevan. SATUSEHAT mendokumentasikan resource FHIR dan validasi terminologi; tiap use case punya pemetaan sendiri.

## 4. Alur pembuatan dan pemrosesan yang perlu dipetakan

### Workflow yang ada

`draft → ready_for_review → in_review → (needs_information) → ready_for_submission → submitted → completed`

Saat ini pengiriman terjadi di luar sistem; aplikasi hanya mencatat referensi dan outcome yang dimasukkan Staff. Staff membuat permohonan dengan payer/layanan/pasien/diagnosis, mengunggah PDF teks atau CSV, memeriksa ekstraksi, kemudian Reviewer menilai field dan checklist. Satu dokumen yang selesai diparse dan setidaknya satu dokumen diperlukan untuk mengirim ke review.

### Alur target Indonesia: sketsa untuk sesi discovery

1. **Pilih konteks kasus**: fasyankes, skema penjamin, tipe layanan, level pelayanan, serta workflow (rujukan, pra-verifikasi manfaat, penjaminan/otorisasi komersial, atau klaim). Hindari satu formulir yang mencampur semua jalur.
2. **Buat kasus minimal**: nomor internal, pemilik/staf, sumber RME/dokumen, tanggal rencana, identitas pasien minimum yang diperlukan, fasilitas pengirim dan penerima.
3. **Ambil/isi data dari sumber yang berwenang**: identitas pasien/anggota, episode/kunjungan, diagnosis dan layanan. Tampilkan sumber serta tanggal, dan sediakan koreksi/manual fallback.
4. **Tentukan prasyarat berdasarkan aturan berversi**: jenis rujukan/kunjungan, status yang perlu dikonfirmasi, surat/dokumen wajib/bersyarat, versi aturan, sumber resmi, masa berlaku, dan siapa yang menyetujui.
5. **Unggah/tautkan dokumen**: beberapa berkas per kasus; klasifikasi jenis, penerbit, tanggal, masa berlaku; deduplikasi dan keterbacaan; status OCR/parse, error dan retry. Gunakan PDF/CSV sebagai format dokumen yang mungkin, bukan model alur itu sendiri.
6. **Ekstrak dengan provenance**: setiap nilai menyimpan berkas/halaman/posisi atau baris sumber, metode ekstraksi, confidence dan status review. Tandai data konflik lintas dokumen; jangan menimpa nilai sumber diam-diam.
7. **Review manusia**: staf administrasi memeriksa kelengkapan; klinisi memvalidasi informasi klinis sesuai kewenangan; lakukan klarifikasi dan versi revisi dengan audit trail.
8. **Langkah luar aplikasi/integrasi resmi**: petugas melanjutkan ke sistem RME, BPJS, SATUSEHAT, asuransi, atau TPA sesuai kanal yang sah. Catat hanya kejadian aktual, referensi dan waktu; jangan menampilkan status “terkirim” sebelum tindakan terjadi.
9. **Catat respons/tindak lanjut**: diterima, perlu dilengkapi, dirujuk, SEP diterbitkan, penjaminan/otorisasi disetujui/ditolak, atau klaim diproses—hanya jika taksonomi itu sesuai sistem target. Simpan alasan/dokumen respons.
10. **Tutup sesuai kebijakan**: retensi, pembatasan akses, ekspor, koreksi, penghapusan, dan audit mengikuti kebijakan institusi dan aturan yang berlaku.

## 5. Requirements perubahan aplikasi

| Prioritas | Kebutuhan | Saran implementasi produk |
|---|---|---|
| P0 - discovery | Pilih satu persona, fasilitas, pembayar, jenis layanan, dan proses | Jadikan keputusan scope dan checklist contoh sebagai artefak yang ditinjau mentor/calon pengguna sebelum kontrak field dibuat. |
| P0 - model domain | Pisahkan `coverage_scheme/payer`, `service_context`, `facility`, `patient reference`, `referral/visit`, `documents`, dan `external transaction` | Hindari mengunci seluruh domain ke `payer_name + service_type`; buat adapter/config untuk workflow yang beda. Hindari migration destruktif; pertahankan data lama sebagai legacy/AS profile. |
| P0 - Indonesia | Hilangkan ketergantungan NPI sebagai identifier default | Hentikan asumsi NPI dan lookup CMS pada profil Indonesia. Tambahkan model identifier bertipe/sistem/sumber; dukung entry manual sampai master data/API terverifikasi. |
| P0 - checklist | Dukung aturan bersyarat dan dokumen | Checklist/version perlu konteks (skema, fasilitas/level, jenis encounter, layanan), tanggal berlaku, sumber, reviewer/pemilik, tanggal tinjau berikutnya, jenis bukti, kondisi penerapan, dan versi yang dibekukan pada kasus. |
| P0 - dokumen | Katalog jenis dokumen dan attachment majemuk | Simpan tipe, tanggal terbit/berlaku, asal, halaman, ukuran/hash, uploader, status parse/OCR, kegagalan, dan relasi per field. Rancang ukuran/retensi dan kontrol akses sebelum data nyata. |
| P0 - workflow | Status spesifik konteks, dibedakan dari hasil kelengkapan | Pisahkan status proses kasus internal dari status transaksi eksternal; jangan menggabungkan `ready_for_submission` generik dengan rujukan, SEP, otorisasi atau klaim. Tampilkan kanal dan sumber status. |
| P0 - privacy/security | Tata kelola data kesehatan | Tinjau peran pengendali/prosesor, dasar pemrosesan dan pemberitahuan, minimisasi, akses per kasus, enkripsi, retensi/pemusnahan, backup, audit, incident handling, dan penggunaan vendor AI. Terapkan data sintetis sampai persetujuan institusi. |
| P1 - interoperabilitas | SATUSEHAT/RME atau BPJS bila benar-benar dibutuhkan | Mulai dari sandbox/dokumentasi resmi dan use case spesifik; onboarding organisasi/lokasi/tenaga kesehatan, OAuth/token, IHS, FHIR resource dan terminologi terkait. Validasi apakah integrasi BPJS/provider tersedia dan diizinkan. Jangan mengarang endpoint. |
| P1 - upload/import | Format dokumen lokal dan impor data terstruktur | Tetapkan template CSV/XLSX dengan kolom, kamus, versi, encoding, tanggal, validasi, dan contoh sintetis. PDF scan/OCR baru diprioritaskan setelah pengujian berizin; pertimbangkan impor dari RME/API sebagai alternatif input manual. |
| P1 - bahasa dan UX | Bahasa Indonesia dan istilah yang dipakai staf | Label, instruksi, error, contoh dokumen, ekspor, format tanggal/angka, dan istilah status harus diuji pemahaman oleh calon pengguna. |
| P1 - audit | Jejak perubahan dan keputusan | Log read/export/parse/correction/checklist/transisi; nilai sebelumnya/baru; aktor/waktu; alasan; versi aturan; identitas sumber. Periksa kesesuaian visibilitas log terhadap kebijakan klinik. |
| P2 - evaluasi | Validasi lapangan sebelum klaim manfaat | Ukur exact match field, sensitivitas temuan dokumen hilang, false-complete rate, koreksi, waktu staf, error parsing, dan usability per jenis dokumen/use case; laporkan sampel dan batasnya. |

Perubahan skema yang mungkin dipertimbangkan setelah discovery: `workflow_profile`/`payer_program`, fasilitas/organisasi, typed identifiers, kasus layanan/episode/rujukan, lampiran bertipe, checklist criteria/conditional rules, extraction provenance per page/row, external event log, retention metadata. Ini bukan instruksi untuk langsung membuat semua tabel; desain final harus mengikuti alur terpilih dan aturan sumber.

## 6. Pertanyaan untuk mentor domain/calon pengguna

Gunakan sesi walkthrough tanpa mengumpulkan data pasien nyata. Minta narasumber menggambarkan satu kasus terakhir langkah demi langkah dan menunjukkan blank form/contoh sintetis yang disetujui.

- Apa nama proses yang benar dipakai organisasi? Siapa yang memulai dan siapa yang bertanggung jawab pada tiap langkah?
- Apakah ini alur JKN, perusahaan asuransi tertentu, TPA, rujukan, SEP, penjaminan, otorisasi, atau klaim? Apa definisi “lengkap”?
- Field/dokumen mana yang wajib, kondisional, atau hanya membantu? Sumber resmi dan versi masing-masing apa?
- Dokumen datang dari RME mana, file apa, siapa yang boleh mengunduh/mengirimnya, dan bagian mana yang paling sering hilang/salah?
- Apakah PDF scan, foto, formulir cetak, hasil lab atau rekam medis menjadi input? Bahasa, variasi layout, kualitas, volume dan batas ukurannya?
- Identifier mana yang dipakai: MRN, nomor peserta JKN, IHS, nomor rujukan, SEP, polis/anggota, dan identifier fasilitas/nakes? Mana yang tidak perlu disimpan?
- Bagaimana sistem memberi status, nomor transaksi, permintaan perbaikan dan alasan? Kanal manual atau integrasi apa yang resmi tersedia?
- Siapa pemilik checklist; bagaimana pembaruan aturan/produk diketahui; berapa lama versi lama berlaku untuk kasus berjalan?
- Risiko salah ekstraksi apa yang paling kritis? Siapa memverifikasi klinis? Bagaimana ketidaksesuaian antar dokumen diatasi?
- Persetujuan, dasar pemrosesan, penyimpanan, vendor, lokasi, retensi dan akses apa yang diwajibkan institusi sebelum pilot?
- Apa manfaat yang bisa diukur sekarang (waktu, rework, missing documents)? Baseline dan kriteria “layak dilanjutkan” apa?

## 7. Batas dan status validasi

**Didukung bukti yang diperiksa:** proposal menyatakan sasaran awal AS dan NPI/CMS; kode mengimplementasikan NPI 10 digit, payer + service checklist, CSV `field,value`, text-PDF tanpa OCR, ekstraksi OpenAI dan review manusia; regulasi/dokumentasi Indonesia meletakkan persyaratan RME, keamanan/kerahasiaan, variabel metadata, serta interoperabilitas SATUSEHAT/FHIR. Panduan BPJS untuk peserta memperlihatkan elemen FKTP/FKRTL, rujukan, dan SEP dalam layanan rujukan lanjutan.

**Belum divalidasi:** workflow operasional terbaru di provider target; ketentuan dan API integrasi BPJS untuk skenario sasaran; formulir dan checklist asuransi komersial; kewajiban legal spesifik pada model hosting/vendor; definisi field yang benar-benar wajib; akses, retensi, dan willingness-to-pay. Sumber publik yang ditelaah tidak membuktikan bahwa semua layanan Indonesia memerlukan PA atau menggunakan satu format PDF/CSV.

## Referensi primer

- Proposal pengguna, *Proposal Industri Prior Authorization Assistant*, versi 2.0, 21 Oktober 2026 (17 halaman).
- [Permenkes Nomor 24 Tahun 2022 tentang Rekam Medis (JDIH Kemenkes)](https://jdih.kemkes.go.id/documents/peraturan-menteri-kesehatan-nomor-24-tahun-2022) — status tercantum berlaku; memuat kewajiban RME, kegiatan rekam medis, keamanan/kerahasiaan, dan interoperabilitas.
- [Kepmenkes HK.01.07/Menkes/1423/2022 tentang Pedoman Variabel dan Metadata RME (JDIH Kemenkes)](https://jdih.kemkes.go.id/documents/keputusan-menteri-kesehatan-nomor-hk0107menkes14232022).
- [SATUSEHAT: Apa itu platform dan ekosistem Indonesia](https://satusehat.kemkes.go.id/platform/docs/id/playbook/introduction/).
- [SATUSEHAT: prasyarat onboarding, Organization, Location, tenaga kesehatan, dan pasien](https://satusehat.kemkes.go.id/platform/docs/id/fhir/prerequisites/).
- [SATUSEHAT: interoperabilitas FHIR](https://satusehat.kemkes.go.id/platform/docs/id/api-catalogue/integrations/) dan [validasi terminologi](https://satusehat.kemkes.go.id/platform/docs/id/api-catalogue/validasi/).
- [BPJS Kesehatan: prosedur pelayanan rujukan tingkat lanjutan (leaflet peserta PDF)](https://web.bpjs-kesehatan.go.id/uploads/information/4bd28c6ea8f022040f6eb93cfcd6e723.pdf).
- [UU Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi (BPK RI)](https://peraturan.bpk.go.id/Home/Download/224884/UU%20Nomor%2027%20Tahun%202022.pdf) — data/informasi kesehatan termasuk data pribadi spesifik.

Sumber regulasi dan API bersifat dapat berubah; periksa kembali dokumen yang berlaku dan konfirmasi dengan petugas legal/privacy serta pemilik integrasi sebelum pilot.
