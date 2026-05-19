/** Full-page wiki articles (sidebar Wiki). Keep in sync with docs/guide when behavior changes. */

export type WikiPageId = 'wiki-guide' | 'wiki-flag-gen';

export interface WikiPageLink {
  label: string;
  route: string[];
}

export interface WikiPageSection {
  /** Anchor id for in-page TOC (flag-gen 5W1H). */
  id?: string;
  title: string;
  paragraphs?: string[];
  bullets?: string[];
}

export interface WikiMenuCard {
  icon: string;
  title: string;
  summary: string;
  route: string[];
  wikiHint: string;
}

export interface WikiPageContent {
  id: WikiPageId;
  eyebrow: string;
  title: string;
  summary: string;
  sections?: WikiPageSection[];
  tutorialSteps?: string[];
  menuCards?: WikiMenuCard[];
  links?: WikiPageLink[];
}

export const WIKI_PAGES: Record<WikiPageId, WikiPageContent> = {
  'wiki-guide': {
    id: 'wiki-guide',
    eyebrow: 'Wiki · Panduan',
    title: 'Panduan dashboard flag-gen',
    summary:
      'Admin dashboard untuk mengelola project, environment, flag, dan akses. Halaman ini menjelaskan alur kerja singkat, tiap menu di sidebar, dan cara memakai owl (bantuan per halaman).',
    tutorialSteps: [
      'Buka /lifecycle — periksa status database. Jika db_empty, jalankan Install; jika migration_required, jalankan Migrate sampai db_ready.',
      'Login di /login dengan akun admin.',
      'Projects — buat project (key + name). Salin Project ID (UUID) dari baris list; ini dipakai SDK dan API.',
      'Environments — buat environment (mis. development, production) di dalam project.',
      'Flags — pilih environment, buat flag (key, tipe value, enabled, rules/rollout). Uji evaluate dari panel bila tersedia.',
      'Opsional: Audit untuk melacak perubahan; Users/Roles untuk mengatur siapa yang boleh mengedit.',
    ],
    menuCards: [
      {
        icon: 'fa-solid fa-folder-tree',
        title: 'Projects',
        summary: 'Scope teratas: satu produk atau domain flag.',
        route: ['/dashboard/projects'],
        wikiHint: 'Tekan ? di header halaman Projects untuk field key vs Project ID.',
      },
      {
        icon: 'fa-solid fa-layer-group',
        title: 'Environments',
        summary: 'Pemisahan dev/staging/prod dalam satu project.',
        route: ['/dashboard/projects'],
        wikiHint: 'Buka project dulu, lalu Environments di sidebar. Environment key dipakai di path API.',
      },
      {
        icon: 'fa-solid fa-users-viewfinder',
        title: 'Segments',
        summary: 'Definisi audiens tingkat project (MVP: CRUD; evaluator belum memakai segment di rules).',
        route: ['/dashboard/projects'],
        wikiHint: 'Tekan ? di halaman Segments untuk status implementasi.',
      },
      {
        icon: 'fa-solid fa-flag',
        title: 'Flags',
        summary: 'Definisi flag per environment: enabled, value, rules, rollout.',
        route: ['/dashboard/projects'],
        wikiHint: 'Tekan ? di halaman Flags untuk evaluate vs definisi flag.',
      },
      {
        icon: 'fa-solid fa-clock-rotate-left',
        title: 'Audit',
        summary: 'Log perubahan per project (siapa, kapan, entity apa).',
        route: ['/dashboard/audit'],
        wikiHint: 'Pilih project di filter sebelum search.',
      },
      {
        icon: 'fa-solid fa-user-gear',
        title: 'Users',
        summary: 'Akun login dashboard (bukan API key aplikasi).',
        route: ['/dashboard/users'],
        wikiHint: 'Assign role dari halaman Roles.',
      },
      {
        icon: 'fa-solid fa-user-shield',
        title: 'Roles',
        summary: 'Bundle permission untuk user dashboard.',
        route: ['/dashboard/roles'],
        wikiHint: 'Gunakan chip permission atau edit JSON array.',
      },
    ],
    sections: [
      {
        title: 'Apa itu flag-gen?',
        paragraphs: [
          'Flag-gen adalah layanan feature flag: tim produk dan engineering mengontrol kapan fitur aktif, untuk siapa, dan dengan nilai apa — tanpa redeploy untuk setiap toggle.',
          'Dashboard ini untuk operator; aplikasi Anda mengintegrasikan lewat HTTP/WebSocket (lihat Wiki › Developer).',
        ],
      },
      {
        title: 'Owl & tombol ? — bantuan per halaman',
        bullets: [
          'FAB owl (kanan bawah) membuka drawer singkat untuk halaman yang sedang Anda buka — konten mengikuti route (projects, flags, audit, dll.).',
          'Tombol ? di header beberapa halaman membuka drawer yang sama.',
          'Drawer = jawaban cepat saat mengisi form. Wiki sidebar (Panduan, Konsep, Developer) = dokumentasi lengkap dan tutorial.',
          'Setelah membaca Panduan, buka halaman target dan tekan ? untuk detail field dan langkah di layar itu.',
        ],
      },
      {
        title: 'Dokumentasi lain di Wiki',
        bullets: [
          'Konsep Flag Gen — apa/mengapa feature flag, dua alur definitions vs evaluate.',
          'Developer — SDK Go, WebSocket broadcast, env vars, contoh kode Angular/Go.',
        ],
      },
    ],
    links: [
      { label: 'Konsep Flag Gen', route: ['/dashboard/wiki/flag-gen'] },
      { label: 'Developer', route: ['/dashboard/wiki/developer'] },
    ],
  },
  'wiki-flag-gen': {
    id: 'wiki-flag-gen',
    eyebrow: 'Wiki · Konsep',
    title: 'Konsep Flag Gen',
    summary:
      'Ringkasan produk dan arsitektur runtime: apa feature flag di flag-gen, di mana data hidup, siapa penggunanya, kapan dipakai, mengapa, dan bagaimana alurnya bekerja.',
    sections: [
      {
        id: 'apa',
        title: 'Apa',
        paragraphs: [
          'Feature flag adalah switch atau konfigurasi yang mengontrol ketersediaan fitur tanpa deploy ulang kode. Di flag-gen setiap flag punya key, tipe value (boolean, string, number, JSON), status enabled, rules, dan rollout.',
        ],
        bullets: [
          'Definisi flag — enabled, value, rules, version untuk satu project + environment. Disinkronkan ke service lewat WebSocket (flags.snapshot + delta) atau GET .../flags.',
          'Evaluate — hasil per request untuk context (userId + attributes): value akhir, reason (disabled, rule_match, rollout, default), source (cache_l1, database, …).',
          'Variation() / cache lokal hanya memakai definisi + default saat enabled — bukan pengganti evaluate jika ada targeting per user.',
        ],
      },
      {
        id: 'dimana',
        title: 'Di mana',
        bullets: [
          'Source of truth: MySQL (dashboard menulis; outbox memicu broadcast).',
          'Cache server: L1 in-process → L2 Redis → MySQL → snapshot file (fallback degraded).',
          'Integrasi backend: WebSocket GET /v1/broadcast/flags?projectId=&environmentKey=&token=.',
          'Integrasi evaluate: POST /v1/projects/:projectId/environments/:environmentKey/flags/evaluate.',
          'Browser: GET/POST /v1/client/... — hanya flag bertanda clientSide; jangan simpan secret di payload flag.',
          'Project ID (UUID) di path API; project key hanya label manusia di dashboard.',
        ],
      },
      {
        id: 'siapa',
        title: 'Siapa',
        bullets: [
          'Operator / tim produk — mengatur flag lewat dashboard, rollout, rollback.',
          'Engineer backend — subscribe WebSocket + panggil evaluate dari service Go (sdk/flagclient).',
          'Engineer frontend — client API + public key untuk flag yang aman di browser.',
          'Admin — Users/Roles untuk akses dashboard; API key terpisah untuk runtime aplikasi.',
          'flag-gen yang broadcast perubahan; service Anda hanya subscribe, tidak publish flag ke peer lain.',
        ],
      },
      {
        id: 'kapan',
        title: 'Kapan',
        bullets: [
          'Cocok: rilis bertahap, canary, emergency kill switch, fitur per segmen user, koordinasi multi-service.',
          'Tidak cocok: credential/secret, config infrastruktur permanen (timeout DB, DSN), data yang harus selalu konsisten tanpa toggle.',
          'Segments di dashboard: definisi audiens project-level; evaluator MVP masih memakai rules di flag langsung — cek status di halaman Segments (?).',
          'Bersihkan flag setelah rollout stabil — hindari technical debt toggle permanen.',
        ],
      },
      {
        id: 'mengapa',
        title: 'Mengapa',
        bullets: [
          'Rilis bertahap dan rollback cepat tanpa redeploy.',
          'Memisahkan deploy kode dari aktivasi fitur (continuous delivery).',
          'Eksperimen dan segmentasi dengan risiko lebih terkontrol.',
          'Feature flag vs config: flag untuk ketersediaan fitur (lifecycle sementara); config untuk perilaku sistem yang lebih permanen (timeout, URL service).',
        ],
      },
      {
        id: 'bagaimana',
        title: 'Bagaimana',
        paragraphs: [
          'Alur perubahan dari dashboard: PATCH flag → outbox → RabbitMQ → WebSocket hub → setiap subscriber terima flag.updated. Local dev butuh OUTBOX_WORKER_ENABLED, BROADCAST_CONSUMER_ENABLED, RabbitMQ, dan BROADCAST_DEV_TOKEN yang sama di server dan SDK.',
          'Alur evaluate: request + userId/attributes → baca definisi dari cache/DB → evaluator (disabled → rules by priority → rollout hash → default).',
        ],
        bullets: [
          'Reconnect WebSocket = flags.snapshot penuh lagi, lalu delta.',
          'source: cache_l1 pada hasil evaluate berarti definisi flag dibaca dari cache — bukan cache hasil per user.',
          'Langkah integrasi detail: Wiki › Developer dan docs/guide/sdk/sdk-flagclient_id.md di repo.',
        ],
      },
    ],
    links: [
      { label: 'Panduan dashboard', route: ['/dashboard/wiki/guide'] },
      { label: 'Developer (SDK)', route: ['/dashboard/wiki/developer'] },
    ],
  },
};
