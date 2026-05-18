# SDK Go (`flagclient`) — Panduan Lengkap

Panduan ini menjelaskan **semua fitur** package Go resmi flag-gen: [`sdk/flagclient`](../../../sdk/flagclient/). 

Dokumen terkait:

- [Indeks guide](../README.md) · [Cara kerja runtime](../how-it-works.md)
- [Feature flags (ID)](../concepts/feature-flags_id.md) — konsep feature flag umum
- [Evaluation semantics](../../assessment/evaluation-semantics.md) — aturan hitung `value` / `reason`
- [API contract](../../assessment/api-contract.md) — endpoint HTTP/WebSocket
- [Contoh integrasi](../../../examples/) — `golang-service` & `angular-client`

---

## Daftar isi

1. [SDK itu apa?](#1-sdk-itu-apa)
2. [Siapa yang broadcast?](#2-siapa-yang-broadcast)
3. [Dua bagian SDK: Client vs ServerSDK](#3-dua-bagian-sdk-client-vs-serversdk)
4. [Konsep penting sebelum coding](#4-konsep-penting-sebelum-coding)
5. [Instalasi](#5-instalasi)
6. [Struct `Client` — konfigurasi](#6-struct-client--konfigurasi)
7. [HTTP: baca daftar flag](#7-http-baca-daftar-flag)
8. [HTTP: evaluate (per user / per request)](#8-http-evaluate-per-user--per-request)
9. [ServerSDK: cache lokal + WebSocket](#9-serversdk-cache-lokal--websocket)
10. [Pesan WebSocket yang didukung](#10-pesan-websocket-yang-didukung)
11. [Kapan pakai fitur yang mana?](#11-kapan-pakai-fitur-yang-mana)
12. [Multi user di satu service](#12-multi-user-di-satu-service)
13. [Autentikasi](#13-autentikasi)
14. [Contoh program lengkap](#14-contoh-program-lengkap)
15. [Anti-pattern (hindari)](#15-anti-pattern-hindari)
16. [Limitasi MVP](#16-limitasi-mvp)
17. [Referensi cepat API](#17-referensi-cepat-api)

---

## 1. SDK itu apa?

**SDK** (Software Development Kit) adalah library Go yang membantu service kamu berbicara dengan flag-gen tanpa menulis HTTP/WebSocket mentah setiap kali.


| Tanpa SDK                       | Dengan SDK                                             |
| ------------------------------- | ------------------------------------------------------ |
| Susun URL, header, JSON sendiri | Panggil fungsi `EvaluateFlags`, `StartWebSocket`, dll. |
| Parsing response manual         | Struct `EvaluationResult` sudah jelas                  |
| Reconnect WebSocket sendiri     | `StartWebSocket` + backoff sudah ada                   |


**Tujuan SDK:**

- Integrasi **backend Go** ke flag-gen dengan cepat
- Pola **service-to-service**: sync definisi flag lewat WebSocket, evaluate lewat HTTP
- Pola **public read** (opsional): baca flag client-side lewat HTTP

Saat ini SDK Go ada di repo ini (`sdk/flagclient`). SDK Angular/browser belum ada sebagai package terpisah — gunakan HTTP langsung (lihat [examples/angular-client](../../../examples/angular-client/)).

---

## 2. Siapa yang broadcast?

**flag-gen** yang broadcast. Service kamu **hanya subscribe**, tidak mengumumkan perubahan flag ke service lain.

```
Dashboard ubah flag
       ↓
flag-gen (outbox → RabbitMQ → WebSocket hub)
       ↓
Service kamu (WebSocket client) — terima flags.snapshot + flag.updated
```

| Peran | Siapa |
|-------|--------|
| Publish perubahan flag | **flag-gen** (outbox + RabbitMQ + WebSocket) |
| Simpan definisi di memory process | **Service kamu** (ServerSDK cache) |
| Keputusan per user (rules/rollout) | **flag-gen evaluate API** (per request) |

**Jangan** jadikan HTTP polling sebagai cara utama sync — itu hanya **fallback** kalau WebSocket/RabbitMQ down.

## 3. Dua bagian SDK: Client vs ServerSDK

```
┌─────────────────────────────────────────────────────────┐
│  Client (HTTP)                                          │
│  • ListFlags / ListPublicFlags                          │
│  • EvaluateFlags / EvaluateFlag / EvaluatePublicFlags   │
└─────────────────────────────────────────────────────────┘
                          │
                          │ dipakai oleh
                          ▼
┌─────────────────────────────────────────────────────────┐
│  ServerSDK (subscriber — cache lokal)                   │
│  • StartWebSocket → flags.snapshot + flag.updated (utama)│
│  • StartPolling → HTTP poll (fallback opsional saja)    │
│  • Snapshot / Variation → baca cache di memory           │
└─────────────────────────────────────────────────────────┘
```


| Komponen        | Transport                   | Isi data                                 |
| --------------- | --------------------------- | ---------------------------------------- |
| `**Client**`    | HTTP                        | Request/response ke API flag-gen         |
| `**ServerSDK**` | WebSocket + HTTP (opsional) | **Definisi flag** di memory process kamu |


`ServerSDK` **bukan** pengganti evaluate untuk targeting per user. Untuk rules/rollout per `country`, `plan`, dll., tetap pakai `**Client.EvaluateFlags`**.

---

## 4. Konsep penting sebelum coding

### 4.1 Dua hal berbeda: **flags** vs **evaluate**


|                            | **Flags (definisi)**                                             | **Evaluate (keputusan)**                      |
| -------------------------- | ---------------------------------------------------------------- | --------------------------------------------- |
| **Apa**                    | Konfigurasi flag: enabled, default, rules, rollout               | Hasil untuk **satu context user** saat ini    |
| **Contoh**                 | `fitur_hutang` default `false`, rule `country=ID` → serve `true` | User ID → `value: true`, `reason: rule_match` |
| **Di-cache global?**       | **Ya** (L1/L2 server + cache lokal SDK)                          | **Tidak** (dihitung tiap request)             |
| **Sama untuk semua user?** | Definisinya sama                                                 | Hasilnya bisa beda per user                   |


Analogi sederhana:

- **Flags** = buku aturan (satu buku di rak, dibaca semua orang)
- **Evaluate** = “kalau user ini, aturan mana yang menang?” (jawaban per orang)

### 4.2 Dari mana datang **context** (userId + attributes)?

Flag-gen **tidak** otomatis baca database user aplikasi kamu.

```json
{
  "context": {
    "userId": "u-123",
    "attributes": {
      "country": "ID",
      "locale": "id",
      "plan": "premium"
    }
  },
  "flags": ["fitur_hutang"]
}
```

**Siapa yang mengisi `attributes`?** Service kamu — biasanya setelah baca profil user dari DB/session/JWT sendiri, lalu kirim ke flag-gen.

### 4.3 Arti field `source` di response evaluate

`source: cache_l1` artinya **definisi flag** dibaca dari cache server — **bukan** “hasil evaluate user di-cache”.

Alur singkat:

1. Ambil definisi flag dari cache global
2. Jalankan algoritma evaluate dengan `context` request ini
3. Kembalikan `value`, `reason`, `version`, …

Detail algoritma: [evaluation-semantics.md](../../assessment/evaluation-semantics.md).

### 4.4 `Variation()` vs `EvaluateFlags()`


| Fungsi                               | Apa yang dikembalikan                   | Targeting rules? | Rollout %? |
| ------------------------------------ | --------------------------------------- | ---------------- | ---------- |
| `ServerSDK.Variation(key, fallback)` | Nilai **default** flag dari cache lokal | **Tidak**        | **Tidak**  |
| `Client.EvaluateFlags(...)`          | Hasil penuh dari server                 | **Ya**           | **Ya**     |


**Kesalahan umum:** pakai `Variation()` untuk fitur “hanya user ID”, lalu heran user EN juga dapat nilai yang sama. Solusinya: **evaluate per request**.

---

## 5. Instalasi

Di `go.mod` service kamu (monorepo flag-gen):

```go
require flag-gen v0.0.0

replace flag-gen => ../path/to/flag-gen  // sesuaikan path
```

Import:

```go
import "flag-gen/sdk/flagclient"
```

Contoh siap jalan: [examples/golang-service](../../../examples/golang-service/).

---

## 6. Struct `Client` — konfigurasi

### `New(baseURL string) *Client`

Membuat client HTTP. `baseURL` biasanya `http://localhost:6123/v1` atau URL production.

### Field konfigurasi


| Field               | Tujuan                                                    | Kapan diisi                                                     |
| ------------------- | --------------------------------------------------------- | --------------------------------------------------------------- |
| `BaseURL`           | Root API `/v1`                                            | Selalu                                                          |
| `HTTPClient`        | Timeout, transport custom                                 | Opsional (default 15 detik)                                     |
| `RuntimeKey`        | API key server (`X-API-Key` atau Bearer)                  | Production; scope `flags:evaluate`, `broadcast:subscribe`, dll. |
| `PublicKey`         | `X-Public-Api-Key` untuk endpoint `/public` dan `/client` | Browser-facing service atau BFF                                 |
| `BroadcastDevToken` | Query `token=` pada WebSocket dev                         | Lokal saja; harus sama dengan `BROADCAST_DEV_TOKEN` di flag-gen |


```go
client := flagclient.New("http://localhost:6123/v1")
client.RuntimeKey = os.Getenv("FLAG_GEN_API_KEY")
client.BroadcastDevToken = os.Getenv("BROADCAST_DEV_TOKEN")
client.PublicKey = os.Getenv("PUBLIC_READ_API_KEY")
```

### Project key vs project ID

Setiap project punya **dua** identifier:

| | Project **key** | Project **ID** |
|---|-----------------|----------------|
| Contoh | `development` | `2c9be7ab-1d09-4caa-bd54-91fdf1d34ee1` |
| Di dashboard | Label yang kamu buat saat create project | Ditampilkan di halaman Projects / Environments / Flags |
| Di API path | Tidak dipakai | **Wajib** di `/v1/projects/:projectId/...` dan `projectId=` WebSocket |

**Environment** memakai **environment key** (mis. `development`) di path — sama seperti yang terlihat di UI.

Cara dapat project ID:

1. Dashboard → **Projects** → baris **Project ID** di list, atau panel **API & SDK identifiers** di Environments / Flags.
2. `GET /v1/projects` → field `id` (bukan `key`).
3. Example Go: set `FLAG_GEN_PROJECT_KEY=development` dan biarkan example resolve ke UUID, atau set `FLAG_GEN_PROJECT_ID` langsung.

---

## 7. HTTP: baca daftar flag

### `ListFlags(ctx, projectID, environmentKey) ([]Flag, error)`


|                   |                                                           |
| ----------------- | --------------------------------------------------------- |
| **Endpoint**      | `GET /v1/projects/{id}/environments/{env}/flags`          |
| **Tujuan**        | Ambil **semua** definisi flag di scope (dashboard/server) |
| **Kapan dipakai** | Refresh manual ke ServerSDK; debugging; tool internal     |
| **Auth**          | Dev: sering tanpa key; production: API key                |


**Struct `Flag`:** `Key`, `Enabled`, `ClientSide`, `Value.Default`, `Version`.

### `ListPublicFlags(ctx, projectID, environmentKey) (map[string]any, error)`


|                   |                                                                      |
| ----------------- | -------------------------------------------------------------------- |
| **Endpoint**      | `GET /v1/public/projects/.../flags`                                  |
| **Tujuan**        | Hanya flag dengan `clientSide: true` (aman diekspos ke browser)      |
| **Kapan dipakai** | BFF Go yang melayani frontend; **bukan** untuk secret                |
| **Auth**          | Header `X-Public-Api-Key` jika `PUBLIC_READ_API_KEY` diset di server |


---

## 8. HTTP: evaluate (per user / per request)

### Types

```go
type EvalContext struct {
    UserID     string
    Attributes map[string]any  // country, plan, locale, ...
}

type EvaluationResult struct {
    Value         any     // bool atau JSON
    Version       int
    Reason        string  // default, rule_match, rollout_match, ...
    MatchedRuleID *string
    Source        string  // cache_l1, database, ...
}

type EvaluateResponse struct {
    Results       map[string]EvaluationResult
    Timestamp     string
    CorrelationID string
}
```

### `EvaluateFlags(ctx, projectID, environmentKey, evalCtx, flagKeys) (*EvaluateResponse, error)`


|                   |                                                                         |
| ----------------- | ----------------------------------------------------------------------- |
| **Endpoint**      | `POST /v1/projects/.../flags/evaluate`                                  |
| **Tujuan**        | Keputusan flag untuk **context ini** (rules + rollout)                  |
| **Kapan dipakai** | **Hampir selalu** di path request user (checkout, hutang, banner, dll.) |
| **Auth**          | `RuntimeKey` dengan scope `flags:evaluate` (production)                 |


**Contoh:**

```go
res, err := client.EvaluateFlags(ctx, projectID, "production", flagclient.EvalContext{
    UserID: "user-1",
    Attributes: map[string]any{
        "country": "ID",
        "plan":    "premium",
    },
}, []string{"fitur_hutang", "checkout_v2"})
if err != nil { /* handle */ }

if res.Results["fitur_hutang"].Value == true {
    // aktifkan fitur hutang
}
```

**Membaca hasil:**


| Field           | Arti praktis                                                        |
| --------------- | ------------------------------------------------------------------- |
| `Value`         | Nilai akhir (boolean atau object JSON)                              |
| `Reason`        | Kenapa dapat nilai itu (`rule_match`, `default`, `rollout_miss`, …) |
| `MatchedRuleID` | Rule mana yang menang (jika ada)                                    |
| `Version`       | Versi definisi flag saat evaluate                                   |
| `Source`        | Dari mana definisi flag dibaca (bukan cache hasil user)             |


### `EvaluateFlag(...)`

Sama seperti `EvaluateFlags` tetapi untuk **satu** flag key — convenience wrapper.

### `EvaluatePublicFlags(...)`


|                   |                                                                               |
| ----------------- | ----------------------------------------------------------------------------- |
| **Endpoint**      | `POST /v1/client/.../flags/evaluate`                                          |
| **Tujuan**        | Evaluate flag **clientSide** dari konteks browser                             |
| **Kapan dipakai** | BFF/backend ringan yang meneruskan ke frontend; flag harus `clientSide: true` |
| **Auth**          | `PublicKey`                                                                   |


---

## 9. ServerSDK: cache lokal + WebSocket

### `NewServerSDK(client, projectID, environmentKey, pollInterval) *ServerSDK`

Membuat cache in-memory per scope. `pollInterval` dipakai oleh `StartPolling` (default efektif 30 detik).

### `Refresh(ctx) error`


|                   |                                                            |
| ----------------- | ---------------------------------------------------------- |
| **Tujuan**        | Isi/refresh cache dari `Client.ListFlags` (HTTP)           |
| **Kapan dipakai** | Sekali saat startup; cadangan jika WebSocket belum connect |


### `StartWebSocket(ctx)`


|                   |                                                                                    |
| ----------------- | ---------------------------------------------------------------------------------- |
| **Tujuan**        | Connect ke `ws://.../v1/broadcast/flags`; terima `flags.snapshot` lalu update live |
| **Kapan dipakai** | **Backend service** yang ingin definisi flag selalu fresh tanpa polling terus      |
| **Auth**          | `BroadcastDevToken` (query) dan/atau `RuntimeKey` (header)                         |
| **Perilaku**      | Reconnect otomatis dengan exponential backoff (1s → max 30s)                       |


**Yang disimpan:** definisi flag (key, enabled, default, version, …) — **bukan** hasil evaluate per user.

### `StartPolling(ctx)`


|                   |                                                                             |
| ----------------- | --------------------------------------------------------------------------- |
| **Tujuan**        | Panggil `Refresh` berkala sebagai **fallback** jika WebSocket/RabbitMQ down |
| **Kapan dipakai** | Bersamaan dengan `StartWebSocket` (disarankan)                              |


### `Snapshot() []Flag`


|                   |                                                                                |
| ----------------- | ------------------------------------------------------------------------------ |
| **Tujuan**        | Salin semua flag di cache lokal (untuk debug, health check, `/flags` endpoint) |
| **Kapan dipakai** | Observability; **jangan** dipakai sebagai pengganti evaluate per user          |


### `Variation(flagKey, fallback any) any`


|                   |                                                                      |
| ----------------- | -------------------------------------------------------------------- |
| **Tujuan**        | Baca `Value.Default` dari cache jika flag ada & enabled              |
| **Kapan dipakai** | Kill switch sederhana **tanpa** segmentasi user; hot path tanpa HTTP |
| **Batasan**       | **Tidak** menjalankan rules/rollout — hanya default                  |


```go
// OK: flag global on/off, sama untuk semua user
if sdk.Variation("maintenance_mode", false) == true { ... }

// SALAH untuk: hanya country ID
// Gunakan EvaluateFlags dengan attributes["country"]
```

---

## 10. Pesan WebSocket yang didukung

ServerSDK memproses pesan berikut (lihat juga [api-contract](../../assessment/api-contract.md#websocket-broadcast)):


| `type`                          | Efek di cache lokal                                  |
| ------------------------------- | ---------------------------------------------------- |
| `flags.snapshot`                | Ganti seluruh map flag (initial connect)             |
| `flag.created` / `flag.updated` | Upsert satu flag (abaikan jika `version` lebih lama) |
| `flag.deleted`                  | Hapus flag by key (perhatikan `version`)             |


**Prasyarat server:** `BROADCAST_CONSUMER_ENABLED=true` + RabbitMQ agar perubahan dashboard sampai ke WebSocket.

---

## 11. Kapan pakai fitur yang mana?

### Peta keputusan

```
Apakah keputusan bergantung pada user / country / plan / rollout %?
│
├─ YA → Client.EvaluateFlags (tiap request, dengan EvalContext)
│
└─ TIDAK (semua user sama, cukup default on/off)
        │
        ├─ Perlu latency sangat rendah, tanpa HTTP?
        │     └─ ServerSDK.Variation (setelah StartWebSocket)
        │
        └─ Boleh HTTP sekali saat startup
              └─ ServerSDK.Refresh + Variation
```

### Tabel ringkas


| Kebutuhan                            | Fitur SDK                                 |
| ------------------------------------ | ----------------------------------------- |
| Fitur beda per negara/user           | `EvaluateFlags` + `attributes`            |
| Rollout 10% user                     | `EvaluateFlags` + `userId`                |
| Sync definisi flag ke memory service | `StartWebSocket` (utama); `StartPolling` hanya fallback |
| Lihat semua flag di memory           | `Snapshot()`                              |
| Kill switch global                   | `Variation()` atau evaluate               |
| Flag untuk Angular/browser           | `ListPublicFlags` / `EvaluatePublicFlags` |
| Debug: definisi terbaru sekali       | `Refresh()` atau `ListFlags()`            |


### Pola arsitektur yang disarankan (backend)

```
Startup:
  sdk := NewServerSDK(client, projectID, env, 30*time.Second)
  sdk.Refresh(ctx)
  sdk.StartWebSocket(ctx)
  // StartPolling(ctx) — hanya jika broker/WS down

Per HTTP request user:
  evalCtx := flagclient.EvalContext{
      UserID: req.UserID,
      Attributes: map[string]any{
          "country": req.User.Country,
          "plan":    req.User.Plan,
      },
  }
  res, _ := client.EvaluateFlags(ctx, projectID, env, evalCtx, []string{"fitur_hutang"})
  use(res.Results["fitur_hutang"].Value)
```

---

## 12. Multi user di satu service

**Tidak ada conflict** jika:

1. **Definisi flag** di-cache global (WebSocket) — **benar dan diharapkan**
2. **Evaluate** dipanggil **per request** dengan context user yang benar


| User   | Attributes    | Hasil `fitur_hutang` |
| ------ | ------------- | -------------------- |
| User A | `country: ID` | `true` (rule match)  |
| User B | `country: EN` | `false` (default)    |


Keduanya di **satu process Go** — aman, karena hasil evaluate **tidak** disimpan global di flag-gen.

---

## 13. Autentikasi


| Lingkungan     | WebSocket                           | Evaluate server                | Public/client                     |
| -------------- | ----------------------------------- | ------------------------------ | --------------------------------- |
| **Dev lokal**  | `BroadcastDevToken`                 | Sering tanpa key               | `PUBLIC_READ_API_KEY` atau kosong |
| **Production** | API key scope `broadcast:subscribe` | API key scope `flags:evaluate` | `X-Public-Api-Key`                |


Generate API key lewat dashboard: Projects → Environment → API Keys.

---

## 14. Contoh program lengkap

Lihat [examples/golang-service](../../../examples/golang-service/):


| Endpoint contoh  | Menunjukkan                                      |
| ---------------- | ------------------------------------------------ |
| `GET /flags`     | `ServerSDK.Snapshot()` — definisi di cache       |
| `POST /evaluate` | `Client.EvaluateFlags` — response penuh flag-gen |


Cuplikan minimal:

```go
client := flagclient.New(os.Getenv("FLAG_GEN_BASE_URL"))
client.RuntimeKey = os.Getenv("FLAG_GEN_API_KEY")
client.BroadcastDevToken = os.Getenv("BROADCAST_DEV_TOKEN")

projectID := os.Getenv("FLAG_GEN_PROJECT_ID")
env := "development"

sdk := flagclient.NewServerSDK(client, projectID, env, 30*time.Second)
sdk.Refresh(context.Background())
sdk.StartWebSocket(ctx)
sdk.StartPolling(ctx)

// Per request:
out, err := client.EvaluateFlags(ctx, projectID, env, flagclient.EvalContext{
    UserID:     r.Header.Get("X-User-Id"),
    Attributes: map[string]any{"country": "ID"},
}, []string{"fitur_hutang"})
```

---

## 15. Anti-pattern (hindari)


| Anti-pattern                                              | Kenapa salah              | Ganti dengan                                                                     |
| --------------------------------------------------------- | ------------------------- | -------------------------------------------------------------------------------- |
| Cache hasil evaluate per flag di `map[string]bool` global | User saling timpa         | Evaluate tiap request, atau cache per `(userId, flagKey)` di app kamu dengan TTL |
| `Variation()` untuk segmentasi negara                     | Rules tidak dijalankan    | `EvaluateFlags` + attributes                                                     |
| Attributes kosong padahal rule pakai `country`            | Rule tidak pernah match   | Isi dari DB user                                                                 |
| Menyimpan secret di flag value                            | Payload bisa ke client    | Config/secret manager                                                            |
| Hanya WebSocket tanpa evaluate untuk fitur per-user       | Cache hanya punya default | Evaluate per request                                                             |
| HTTP polling tiap N detik sebagai sync utama              | Mengabaikan broadcast flag-gen; telat & boros | `StartWebSocket`; polling hanya fallback (`FLAG_GEN_POLL_FALLBACK`) |


---

## 16. Limitasi MVP

- **Tidak ada evaluasi rules lokal di SDK** — rules/rollout hanya di server (`EvaluateFlags`).
- `**Variation()`** = default value saja.
- **Satu bahasa** — Go saja; Angular pakai HTTP (contoh di `examples/angular-client`).
- **Tidak ada SDK CLI** — hanya library.
- WebSocket butuh consumer RabbitMQ aktif untuk update live setelah edit di dashboard.

Rencana perluasan: lihat [implementation-status.md](../implementation-status.md) dan [features.md](../../assessment/features.md).

---

## 17. Referensi cepat API

### `Client` (HTTP)


| Fungsi                                                    | Ringkas                       |
| --------------------------------------------------------- | ----------------------------- |
| `New(baseURL)`                                            | Buat client                   |
| `ListFlags(ctx, projectID, env)`                          | Daftar definisi flag (server) |
| `ListPublicFlags(ctx, projectID, env)`                    | Daftar flag client-side       |
| `EvaluateFlags(ctx, projectID, env, evalCtx, keys)`       | Evaluate bulk                 |
| `EvaluateFlag(ctx, projectID, env, evalCtx, key)`         | Evaluate satu flag            |
| `EvaluatePublicFlags(ctx, projectID, env, evalCtx, keys)` | Evaluate client-side          |


### `ServerSDK` (cache + WebSocket)


| Fungsi                                               | Ringkas                    |
| ---------------------------------------------------- | -------------------------- |
| `NewServerSDK(client, projectID, env, pollInterval)` | Buat cache                 |
| `Refresh(ctx)`                                       | Isi cache via HTTP         |
| `StartWebSocket(ctx)`                                | Sync via WebSocket         |
| `StartPolling(ctx)`                                  | Refresh berkala (fallback) |
| `Snapshot()`                                         | Salin cache                |
| `Variation(key, fallback)`                           | Baca default (tanpa rules) |


---

*Terakhir diperbarui sesuai `sdk/flagclient` di repo flag-gen.*