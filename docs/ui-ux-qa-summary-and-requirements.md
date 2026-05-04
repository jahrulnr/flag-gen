# Ringkasan QA UX — Admin dashboard & persyaratan

Dokumen ini merangkum hasil uji eksploratif/E2E pada **Flag Gen Admin** (`web/`, base `/app/`), mencatat apa yang **tidak lolos** dari sudut UX, dan memberi **rekomendasi tegas** untuk menekan human error. Isi ini selaras dengan bar QA di skill **`web-admin-dashboard-qa`**.

---

## Ringkasan singkat

| Area | Status UX |
|------|-----------|
| Alur utama (projects → environments → flags → history, segments, audit, wiki) | **Lulus** setelah perbaikan kontras tabel, breadcrumb, empty state audit |
| **Update** flag & segment (form atas, reset setelah simpan) | **Lulus** |
| **Delete** segment (tanpa konfirmasi, satu klik) | **Tidak lolos** — risiko salah klik tinggi |
| **Delete** flag (tombol baris terakhir) | **Tidak lolos** — klik tertangkap **overlay maskot owl (Lottie canvas)**; aksi destruktif tidak dapat diandalkan |
| **Delete** project/environment (ada `confirm()` native) | **Sebagian** — ada gate, tapi UX native tipis; belum setara standar “sengaja menghapus” |
| Pesan error bertipe **`[VALIDATION_ERROR]`** / kode mentah API | **Tidak lolos** — terlalu teknikal untuk operator |

---

## Yang tidak lolos (detail)

### 1. Maskot owl menutupi aksi destruktif

**Gejala:** `canvas.owl-lottie__canvas` menangkap pointer di area kanan bawah; tombol **hapus flag** pada baris terakhir **tidak terpanggil** — pengguna mengira UI rusak atau tombol mati.

**Dampak:** human error berlipat: gagal hapus *dan* frustrasi; sebaliknya jika suatu saat klik “nyasar” ke area yang salah, risiko tap tidak terduga.

### 2. Hapus flag & segment tanpa dialog yang serius

**Gejala:** `remove()` memanggil API DELETE langsung; tidak ada modal / tidak ada langkah konfirmasi bertingkat.

**Dampak:** satu klik salah = data hilang; tidak sejajar dengan praktik admin console yang aman.

### 3. Pesan error terlalu teknikal

**Gejala:** respons API memakai kode seperti **`VALIDATION_ERROR`** (lihat konstanta di `pkg/error/error.go`); jika string itu (atau serupa) muncul apa adanya di UI, pengguna non-engineer tidak tahu **apa** yang harus diperbaiki.

**Dampak:** support noise, retry membabi buta, dan kepercayaan pada produk turun.

### 4. Minor yang tetap mengganggu

- **Segments — label aksesibilitas Rules:** teks seperti “Rules JSON JSON array…” (duplikasi) membingungkan pembaca layar.
- **Segments vs flags — pola edit:** Flags punya banner *editing*; Segments tidak — inkonsistensi orientasi “mode edit”.
- **Project ID di Audit:** UUID di field teks memicu **spellcheck** browser (garis merah) — terlihat seperti error padahal valid.

---

## Rekomendasi (tegas, prioritas tinggi)

Rekomendasi di bawah sengaja **keras**: tujuannya memperkecil human error dan memenuhi ekspektasi admin tool level produksi.

### A. Posisi & perilaku maskot owl (wajib)

1. **Pindahkan FAB / maskot ke kiri bawah** (kesepakatan produk: kiri bawah), *atau* ke rail yang tidak overlap baris aksi list.
2. Pastikan **z-index** dan **`pointer-events`**: area animasi tidak boleh menjadi target klik di atas **row actions** (Edit / History / Delete). Prefer: hanya **tombol kecil** maskot yang `pointer-events: auto`, sisanya `none` pada canvas/container besar.
3. Uji regresi di **viewport pendek** dan **list 1–3 baris** — kasus paling rentan overlap.

### B. Hapus destruktif — modal wajib + ketik ulang **key** (wajib)

Berlaku konsisten untuk: **flag**, **segment**, **environment**, **project** (setara atau lebih ketat dari yang sudah ada untuk project).

**Pol yang disarankan:**

1. Pengguna memilih **Delete** → buka **modal** (bukan `confirm()` satu baris saja).
2. Teks jelas: apa yang dihapus, dampak (mis. “semua flag di environment ini”), dan bahwa aksi **tidak dapat dibatalkan** di UI.
3. Field wajib: **ketik ulang persis `key` sumber daya** (mis. `qa_e2e_tmp_flag`, `production`, `my_service`) untuk mengaktifkan tombol **Hapus permanen**.
4. Tombol utama destruktif: warna danger, label eksplisit (**Hapus flag**), bukan hanya ikon.
5. Opsional tambahan: centang “Saya mengerti data akan hilang” *setelah* field key cocok — untuk double gate tanpa menjengkelkan orang yang sudah mengetik key.

Ini sengaja “repot” sedikit: **itu fitur**, bukan bug — mengubah biaya kognitif salah klik dari nol menjadi sengaja.

### C. Error surface — jangan pernah tunjukkan kode enum mentah (wajib)

1. **Peta** `error` / `code` dari API ke **kalimat bahasa manusia** (Indonesia atau Inggris konsisten dengan produk), plus **field** yang bermasalah jika ada `details`.
2. **`VALIDATION_ERROR`** → contoh salin user: *“Beberapa isian tidak valid. Periksa Key (hanya huruf kecil, angka, underscore).”* — sesuaikan dengan aturan validasi sebenarnya.
3. Simpan `correlationId` di **footer** error / “Lapor ke admin” — jangan jadikan headline.
4. Untuk 403 CSRF / session: pesan seperti *“Sesi keamanan habis. Muat ulang halaman lalu coba lagi.”*

### D. Segments — samakan standar dengan Flags

1. Banner **Editing segment** dengan menampilkan **`key`** yang sedang diedit saat mode patch.
2. Perbaiki **accessible name** textarea Rules (satu frasa jelas, tanpa duplikasi “JSON JSON”).

### E. Audit — field Project ID

- `spellcheck="false"` `autocorrect="off"` `autocapitalize="off"` pada input UUID agar tidak terlihat “salah eja”.

---

## Prinsip UX non-negotiable (untuk QA & implementasi)

1. **Tidak ada elemen dekoratif/animasi yang memblokir aksi primer atau destruktif.**
2. **Tidak ada penghapusan sumber daya tanpa modal + konfirmasi berbasis key** (setara untuk semua entitas yang bisa dihapus dari UI).
3. **Tidak ada pesan error yang hanya menampilkan kode internal** tanpa terjemahan yang bisa ditindak pengguna akhir.
4. **Mode edit vs buat** harus terbaca sekilas (judul, banner, atau layout yang konsisten antar halaman serupa).

QA yang menemukan pelanggaran di atas mencatat sebagai **fail** sampai diperbaiki; bukan “nice to have”.

---

## Referensi terkait

- Alur rute & checklist teknis: **`.agents/skills/web-admin-dashboard-qa/SKILL.md`**
- Temuan regresi visual awal (tabel, breadcrumb): **`docs/ui-ux-regression-report-2026-05-04.md`** (jika masih ada di repo)

---

*Terakhir diperbarui mengikuti sesi QA manual (flags, segments, delete, update, overlay owl, pesan error).*
