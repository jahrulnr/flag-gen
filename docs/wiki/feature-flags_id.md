# Wiki Feature Flags

## Pengantar

Feature flag, atau feature toggle, adalah teknik pengembangan perangkat lunak untuk mengaktifkan atau menonaktifkan fitur tanpa deployment kode baru. Pendekatan ini membantu tim merilis fitur secara bertahap, melakukan rollback cepat, dan menguji perilaku fitur dengan risiko lebih rendah.

## Apa Itu Feature Flag?

Feature flag biasanya terdiri dari:

- Nama atau key flag
- Nilai flag, seperti boolean atau konfigurasi JSON
- Logika evaluasi untuk menentukan nilai yang akan dipakai aplikasi

### Manfaat

- **Rilis bertahap**: fitur bisa dirilis ke sebagian pengguna terlebih dahulu.
- **Rollback cepat**: fitur bermasalah bisa dimatikan tanpa redeployment.
- **Continuous deployment**: deployment kode bisa dipisahkan dari aktivasi fitur.
- **Koordinasi tim**: beberapa tim bisa mengembangkan fitur paralel dengan risiko konflik lebih kecil.
- **Eksperimen**: variasi fitur bisa diuji jika platform sudah mendukung analitik eksperimen.

## Kapan Feature Flag Diperlukan?

Feature flag cocok digunakan untuk:

- Rilis fitur baru yang ingin dilakukan bertahap
- Canary release atau beta testing
- Emergency rollback
- Fitur yang hanya aktif untuk segmen pengguna tertentu
- Koordinasi perubahan di beberapa service
- Multi-tenant application
- Fitur opsional yang belum tentu aktif untuk semua produk atau pelanggan

Feature flag tidak cocok digunakan sebagai tempat menyimpan credential, secret, atau konfigurasi sensitif. Semua payload flag, terutama yang dikirim ke client-side, harus dianggap bisa dibaca oleh pengguna.

## Cakupan Feature Flag

Feature flag dapat dikategorikan berdasarkan tujuan dan lifecycle:

- **Release toggle**: mengontrol kapan fitur dirilis.
- **Experiment toggle**: mendukung A/B test atau eksperimen.
- **Ops toggle**: membantu tim operasional mematikan atau mengaktifkan perilaku tertentu.
- **Permission toggle**: mengontrol akses fitur berdasarkan role, paket, atau entitlement.
- **Business toggle**: mengaktifkan fitur berdasarkan aturan bisnis.

## Feature Flag vs Config

| Aspek | Feature Flag | Config |
| --- | --- | --- |
| Tujuan | Mengontrol ketersediaan fitur | Mengatur perilaku aplikasi |
| Lifecycle | Biasanya sementara sampai rollout selesai | Biasanya lebih permanen |
| Contoh | `enable_checkout_v2` | `api_timeout`, `database_url` |
| Pengelola | Produk, engineering, operasional | Engineering atau DevOps |
| Frekuensi perubahan | Cenderung sering selama rollout | Lebih jarang |
| Risiko | Salah nilai bisa memengaruhi fitur | Salah nilai bisa merusak sistem inti |

## Contoh Kasus

### Scheduled vs Non-Scheduled Product Ordering

Misalnya aplikasi e-commerce menjual produk seperti pulsa atau paket internet. Produk bisa dipesan sebagai:

- **Non-scheduled order**: produk aktif langsung setelah pembelian.
- **Scheduled order**: produk aktif di waktu tertentu, misalnya paket bulanan.

Ini bisa menjadi kasus feature flag jika scheduled order adalah fitur baru yang ingin dirilis bertahap. Flag seperti `enable_scheduled_orders` bisa digunakan untuk mengaktifkan fitur ke segmen tertentu, melakukan beta testing, atau mematikan fitur jika terjadi masalah.

Jika kedua mode order sudah menjadi bagian permanen dari model bisnis, fitur ini lebih tepat ditangani sebagai business logic atau konfigurasi produk, bukan feature flag.

### Optional Voucher Feature

Voucher pada alur checkout adalah contoh feature flag yang kuat jika fitur tersebut:

- Menambahkan langkah baru pada user journey
- Belum tersedia untuk semua produk atau semua pengguna
- Perlu diuji secara bertahap
- Perlu bisa dimatikan cepat jika terjadi masalah

Flag seperti `enable_voucher_feature` dapat mengontrol apakah opsi voucher muncul selama checkout. Setelah fitur stabil dan menjadi perilaku permanen, flag sebaiknya dibersihkan dari kode.

## Alat Manajemen Feature Flag

### GrowthBook

GrowthBook adalah platform open-source untuk feature flagging dan experimentation.

#### Fitur Utama

- Open-source dan bisa self-hosted
- Feature flag dan eksperimen
- SDK untuk berbagai bahasa
- Integrasi dengan data warehouse
- Analitik eksperimen

#### Kelebihan

- Biaya rendah karena open-source
- Cocok untuk tim yang butuh experimentation
- Bisa self-hosted
- Developer-friendly
- Data tetap berada di infrastruktur sendiri

#### Kekurangan

- Setup awal lebih kompleks dibanding layanan managed
- Fitur enterprise tertentu mungkin membutuhkan konfigurasi tambahan
- Perlu pemahaman statistik untuk menggunakan experimentation dengan benar
- Kurang sederhana jika hanya membutuhkan core feature flag

### Unleash

Unleash adalah platform feature flag open-source dengan opsi enterprise.

#### Fitur Utama

- Feature flag management
- Targeting dan segmentation
- Real-time update
- API yang lengkap
- Integrasi dengan CI/CD
- Fitur enterprise seperti SSO dan audit log pada paket tertentu

#### Kelebihan

- Cocok untuk skala organisasi yang lebih besar
- Targeting cukup kuat
- Arsitektur fleksibel
- Komunitas open-source aktif
- Mendukung kebutuhan enterprise

#### Kekurangan

- Bisa terlalu kompleks untuk tim kecil
- Membutuhkan resource operasional lebih besar untuk deployment skala besar
- Beberapa fitur lanjutan berada di paket berbayar
- Setup dan konfigurasi penuh membutuhkan pembelajaran

## Praktik Terbaik

- Gunakan nama flag yang jelas dan konsisten.
- Jangan menyimpan secret atau credential di flag.
- Tandai flag yang aman diekspos ke client-side.
- Bersihkan flag yang sudah tidak dibutuhkan.
- Dokumentasikan tujuan, owner, dan rencana penghapusan flag.
- Test kedua kondisi: flag aktif dan flag nonaktif.
- Tambahkan monitoring untuk evaluasi, fallback, dan error.

## Kesimpulan

Feature flag membantu tim merilis fitur dengan lebih aman, terutama untuk gradual rollout, rollback cepat, dan koordinasi antar service. Untuk sistem internal seperti flag-gen, prioritas awal sebaiknya fokus pada core flag management, evaluasi deterministik, audit trail, WebSocket synchronization, dan fallback availability sebelum masuk ke experimentation atau fitur enterprise.
