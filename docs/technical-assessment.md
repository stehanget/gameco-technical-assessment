# Technical Test — Senior Backend Engineer

## Public Marketplace API (Hardened Edition)

**Durasi:** 2 hari kalender (48 jam sejak email soal diterima) **Estimasi effort inti:** 10–14 jam. Sisanya waktu berpikir, menulis dokumen, dan istirahat. Kami **tidak** mengharapkan Anda begadang 2x24 jam. **Level:** Senior (5+ tahun) **Bahasa/teknologi:** BEBAS. Go, Java, Kotlin, Node/TypeScript, PHP, Python, Rust, C#, apapun. Framework bebas. Database bebas.

---

## 0. Baca Ini Dulu

Tes ini **bukan** tes kecepatan mengetik fitur. Kami tidak menghitung berapa banyak endpoint yang Anda selesaikan.

Yang kami nilai adalah **keputusan teknis Anda dan alasan di baliknya**, khususnya di tiga sumbu:

1. **Keamanan** — API ini publik, dikonsumsi langsung oleh client yang tidak dipercaya (browser & mobile app). Tidak ada secret yang bisa disembunyikan di sisi client.
2. **Arsitektur** — batas modul, arah dependensi, kemudahan diubah 6 bulan lagi oleh orang lain.
3. **Flow & correctness** — race condition, konsistensi data uang & stok, idempotency, kegagalan parsial.

**Sepuluh endpoint yang dangkal akan kalah dari empat endpoint yang benar-benar matang.** Kalau waktu Anda habis, potong scope fitur, jangan potong kualitas. Tulis di README apa yang Anda potong dan kenapa.

---

## 1. Konteks Bisnis

Anda bergabung sebagai backend engineer di sebuah marketplace. Tim frontend (web SPA + mobile app) akan mengonsumsi API yang Anda bangun. Ada juga rencana membuka sebagian API untuk **partner/seller tools pihak ketiga** (server-to-server).

Artinya ada **dua kelas consumer dengan model kepercayaan yang berbeda**:

| Consumer                           | Sifat                                        | Bisa simpan secret? |
| ---------------------------------- | -------------------------------------------- | ------------------- |
| Web SPA & Mobile App (first-party) | Public client, kode bisa dibaca siapa saja   | ❌ Tidak            |
| Partner Server (third-party)       | Confidential client, jalan di server partner | ✅ Ya               |

Skema autentikasi & otorisasi Anda harus memperlakukan keduanya secara berbeda. Ini adalah salah satu titik penilaian utama.

**Aktor sistem:**

- `guest` — belum login, boleh browsing katalog
- `buyer` — user terdaftar, bisa checkout
- `seller` — punya toko, mengelola produk sendiri
- `admin` — internal, akses terbatas & teraudit

---

## 2. Scope Fungsional

### 2.1 WAJIB (Core) — kerjakan ini dulu sampai benar-benar solid

#### A. Identity & Session

- `POST /v1/auth/register` — registrasi buyer
- `POST /v1/auth/login` — login, mengembalikan kredensial sesi
- `POST /v1/auth/refresh` — perpanjang sesi
- `POST /v1/auth/logout` — akhiri sesi (harus benar-benar mati, bukan sekadar hapus di client)
- `GET  /v1/me` — profil user saat ini

#### B. Katalog (publik, tanpa auth)

- `GET /v1/products` — list dengan filter (`q`, `category`, `min_price`, `max_price`), sort, dan **pagination**
- `GET /v1/products/{id}` — detail produk

#### C. Seller

- `POST  /v1/seller/products` — buat produk
- `PATCH /v1/seller/products/{id}` — update produk
- `GET   /v1/seller/orders` — daftar order yang masuk ke tokonya

#### D. Transaksi (bagian paling penting)

- `POST /v1/cart/items` — tambah item ke keranjang
- `POST /v1/checkout` — buat order dari keranjang, **potong stok**, hasilkan `payment_reference`
- `GET  /v1/orders/{id}` — detail order
- `POST /v1/webhooks/payment` — callback dari payment gateway (simulasikan; tidak perlu integrasi PG asli)

### 2.2 OPSIONAL (Bonus) — hanya kalau core sudah benar-benar selesai

- Partner API server-to-server: `GET /v1/partner/orders` dengan mekanisme auth yang berbeda dari first-party
- Admin: soft-delete produk bermasalah + jejak audit
- Pembatalan order + pengembalian stok
- Search yang lebih serius (full-text / trigram index)

> Sekali lagi: **bonus tidak menutupi core yang lemah.** Kami lebih memilih kandidat tanpa bonus tapi core-nya bersih.

---

## 3. Persyaratan Keamanan (Bobot Terbesar)

Ini fokus utama tes. Kerjakan dengan sadar, bukan dengan menempel library.

### 3.1 Autentikasi & Manajemen Sesi

Anda harus **memilih dan mempertahankan** sebuah strategi sesi. Pertanyaan yang harus terjawab jelas di kode dan di ADR:

- Di mana kredensial sesi disimpan di sisi client? Cookie `HttpOnly` + `Secure` + `SameSite`? Bearer token di memory? Pola BFF? **Jelaskan trade-off XSS vs CSRF dari pilihan Anda.**
- Berapa TTL access token dan refresh token? Kenapa angka itu?
- Apakah refresh token dirotasi? Bagaimana Anda mendeteksi **refresh token reuse** (indikasi token dicuri) dan apa aksinya?
- Bagaimana logout mematikan sesi secara nyata? (denylist / versi sesi / server-side session — bebas, tapi harus bekerja)
- Bagaimana revokasi massal saat user ganti password?
- Kalau Anda pakai JWT: algoritma apa, kenapa, dan bagaimana Anda mencegah `alg: none` serta algorithm confusion? Bagaimana rencana rotasi key (`kid`)?

**Password:** wajib hashing dengan algoritma modern ber-salt (argon2id / bcrypt / scrypt). Cost parameter disebutkan alasannya. Plaintext atau MD5/SHA-1 = gugur otomatis.

**Anti-enumeration:** respons login/register/forgot-password tidak boleh membocorkan apakah sebuah email terdaftar. Perhatikan juga **timing attack**.

### 3.2 Otorisasi

- Terapkan otorisasi **per-objek**, bukan hanya per-role. Seller A tidak boleh menyentuh produk seller B; buyer X tidak boleh membaca order buyer Y. Ini adalah kerentanan nomor satu OWASP API Top 10 (BOLA/IDOR) dan kami **akan mengujinya**.
- Default deny. Endpoint baru harus aman secara default kalau developer lupa memasang guard — tunjukkan bagaimana desain Anda mencegah "lupa".
- **Mass assignment**: pastikan pembeli tidak bisa mengirim `{"role":"admin"}` atau `{"price":1}` dan mengubah field yang bukan haknya. Gunakan DTO/allow-list eksplisit, bukan bind langsung ke entity.

### 3.3 Integritas Transaksi

- **Race condition stok**: dua request checkout bersamaan pada stok terakhir tidak boleh menghasilkan oversell. Pilih optimistic locking, pessimistic locking, atau conditional update — dan **jelaskan pilihannya**. Sertakan bukti (test konkuren).
- **Idempotency**: `POST /v1/checkout` wajib menerima header `Idempotency-Key`. Request ulang dengan key sama harus mengembalikan hasil yang sama, bukan order kedua. Jelaskan bagaimana Anda menangani dua request identik yang datang **bersamaan**.
- **Webhook payment**: verifikasi signature (HMAC) dengan **constant-time comparison**, tolak timestamp kedaluwarsa (replay), dan pastikan pemrosesan idempoten karena PG akan mengirim ulang.
- Uang & jumlah: jangan pakai floating point. Jelaskan representasi yang Anda pilih.

### 3.4 Input, Output, dan Permukaan Serangan

- Validasi semua input di boundary. Allow-list, bukan deny-list.
- SQL injection: parameterized query / prepared statement. Perhatikan juga sorting & filtering dinamis — ini titik bocor yang paling sering terlewat.
- Batasi ukuran body request dan kedalaman/panjang array.
- **SSRF**: kalau Anda menerima URL gambar produk, jangan langsung fetch. Jelaskan mitigasinya.
- Response tidak boleh membocorkan stack trace, versi framework, query SQL, atau PII milik user lain. Error harus terformat konsisten.
- Security headers & CORS: origin **allow-list eksplisit**, bukan `*` apalagi refleksi origin. Kalau pakai cookie, jelaskan implikasi `credentials`.

### 3.5 Rate Limiting & Abuse

- Rate limit berlapis: per IP, per user/sesi, dan per endpoint sensitif (login, register, checkout, refresh). Angka boleh Anda tentukan, tapi harus beralasan.
- Login: throttling progresif atau lockout — dan jelaskan bagaimana Anda menghindari lockout jadi senjata DoS terhadap korban.
- Jelaskan apa yang terjadi kalau instance API di-scale jadi 5 pod: apakah rate limiter Anda masih benar?

### 3.6 Data & Secret

- Tidak ada secret ter-commit ke repo. Sediakan `.env.example`. Jelaskan bagaimana secret dikelola di produksi.
- PII (email, nomor telepon, alamat) — jelaskan perlakuan Anda: enkripsi at-rest, masking di log, atau keduanya.
- **Logging**: wajib ada `request_id`/`correlation_id` di setiap log dan di response header. Password, token, dan nomor kartu **tidak boleh** pernah masuk log.
- **Audit trail** untuk aksi sensitif: login gagal, perubahan harga, perubahan status order.

### 3.7 Threat Model (dokumen wajib)

Buat file `docs/THREAT-MODEL.md` berisi:

- Pemetaan **OWASP API Security Top 10 (2023)** → di mana Anda memitigasinya di kode (sebut file/fungsi).
- Minimal **5 skenario serangan konkret** terhadap sistem Anda, beserta mitigasi dan **residual risk** yang Anda sadari tapi sengaja tidak tangani (dengan alasan).
- Bagian jujur berjudul "Yang belum aman" — kandidat senior tahu batas sistemnya. Kejujuran di sini bernilai plus, bukan minus.

---

## 4. Persyaratan Arsitektur

- **Layering yang jelas.** Layered / hexagonal / clean — bebas, tapi konsisten. Logic domain tidak boleh bergantung pada framework HTTP atau ORM. Kami akan melihat arah dependensi Anda.
- **Skema database**: normalisasi wajar, constraint di level DB (FK, unique, check), dan **index yang disengaja**. Jangan andalkan aplikasi saja untuk menjaga integritas.
- **Migration**: skema dibuat lewat migration tool, bukan SQL manual di README.
- **N+1 query**: hindari, terutama di list produk & list order.
- **Pagination**: jelaskan pilihan offset vs cursor untuk katalog publik yang datanya banyak.
- **Caching** (kalau dipakai): sebutkan strategi invalidasi. Cache tanpa strategi invalidasi adalah bug yang tertunda. Perhatikan agar data privat tidak pernah masuk cache publik.
- **Dependensi eksternal** (payment gateway): timeout wajib, retry dengan backoff, dan perilaku saat PG down. Circuit breaker adalah nilai plus.
- **Konsistensi event**: kalau Anda mengirim event/notifikasi setelah checkout, jelaskan bagaimana Anda menghindari "order tersimpan tapi event hilang" (mis. transactional outbox).
- **Graceful shutdown**: request yang sedang jalan tidak terpotong saat deploy.
- **Health check**: pisahkan liveness dan readiness, dan jelaskan bedanya.

---

## 5. Kualitas & Testing

- **Test wajib ada** untuk jalur kritis. Kami tidak menuntut coverage tinggi; kami menuntut test yang **tepat sasaran**:
  - satu test konkurensi yang membuktikan tidak ada oversell,
  - satu test idempotency checkout,
  - satu test otorisasi lintas-user (seller A gagal mengubah produk seller B),
  - satu test verifikasi signature webhook (termasuk kasus signature palsu & replay).
- Struktur commit rapi dan bermakna. Satu commit raksasa "init" mengurangi nilai.
- Linter/formatter terpasang.

---

## 6. Deliverables

Kirimkan **link repository Git** (GitHub/GitLab, private, undang reviewer — email di bawah) berisi:

```
/
├── README.md               # wajib, lihat §6.1
├── docs/
│   ├── THREAT-MODEL.md     # wajib
│   ├── ADR/                # wajib, minimal 4 keputusan
│   └── erd.(png|md|puml)   # wajib
├── openapi.yaml            # atau Postman collection
├── docker-compose.yml      # wajib, lihat §6.2
├── .env.example
├── migrations/
└── src/ (atau internal/, app/, dst.)
```

### 6.1 README wajib memuat

1. Cara menjalankan (harus benar-benar jalan di mesin bersih)
2. Ringkasan arsitektur + alasan pemilihannya
3. **Strategi keamanan dalam 1 halaman** — ini bagian yang paling kami baca
4. Trade-off yang Anda ambil karena keterbatasan waktu
5. "Kalau punya 2 minggu lagi, saya akan…"
6. Estimasi waktu yang benar-benar Anda habiskan (jujur; ini tidak dinilai)
7. **Penggunaan AI assistant**: boleh dipakai, tapi tulis bagian mana yang dibantu AI dan bagaimana Anda memverifikasinya. Ketidakjujuran di sini jauh lebih fatal daripada memakai AI-nya.

### 6.2 Docker

`docker compose up` harus menjalankan seluruh sistem (API + DB + dependensi) tanpa langkah manual selain menyalin `.env.example`. Seed data minimal wajib ada agar reviewer bisa langsung mencoba.

Kalau reviewer tidak bisa menjalankan aplikasi Anda dalam 10 menit, penilaian berhenti di situ.

### 6.3 ADR

Minimal 4 Architecture Decision Record singkat (1 halaman masing-masing: konteks → opsi → keputusan → konsekuensi). Wajib termasuk:

- ADR strategi sesi & penyimpanan token di client
- ADR penanganan konkurensi stok
- ADR pilihan database & pemodelan data
- ADR bebas

---

## 7. Rubrik Penilaian

| Aspek                                                                    |   Bobot |
| ------------------------------------------------------------------------ | ------: |
| Keamanan (auth, otorisasi objek, input, secret, threat model)            | **35%** |
| Arsitektur & desain data (batas modul, skema, index, dependensi)         | **25%** |
| Correctness transaksi (konkurensi, idempotency, webhook, error handling) | **20%** |
| Kualitas kode & testing                                                  | **10%** |
| Dokumentasi, ADR, kejelasan reasoning                                    | **10%** |

### Yang membuat kami sangat terkesan

- Anda menemukan lubang di soal ini dan menyebutkannya
- Anda menolak menambah fitur demi menyelesaikan satu hal dengan benar
- Threat model yang menyebut residual risk secara jujur
- Test konkurensi yang benar-benar reproducible

### Gugur otomatis

- Password disimpan plaintext atau di-hash dengan MD5/SHA-1 tanpa salt
- Secret asli ter-commit di repo
- IDOR terbuka di endpoint order/produk
- Aplikasi tidak bisa dijalankan sama sekali
- Kode disalin dari repo lain tanpa atribusi

---

## 8. Saran Pembagian Waktu

**Hari 1**

- 1–2 jam: desain. Gambar ERD, tentukan skema auth, tulis draft ADR **sebelum** menulis kode.
- 3–4 jam: fondasi — skema DB + migration, auth lengkap (register/login/refresh/logout + rotasi), middleware otorisasi, error handling terpusat, structured logging, docker compose.
- 1–2 jam: katalog publik + endpoint seller.

**Hari 2**

- 3–4 jam: cart, checkout dengan locking stok + idempotency, webhook payment bertanda tangan.
- 1–2 jam: test jalur kritis (konkurensi, idempotency, otorisasi, webhook).
- 1–2 jam: rate limiting, security headers, CORS, pembersihan.
- 1–2 jam: **README, threat model, ADR.** Jangan sisakan ini di 15 menit terakhir — bobotnya besar.

---

## 9. Aturan Main

- Boleh memakai library, framework, dan referensi apapun. Boleh memakai AI assistant, dengan syarat disebutkan (§6.1 poin 7).
- Tidak boleh menyalin repo utuh milik orang lain.
- Kode ini milik Anda. Kami tidak akan memakainya untuk produksi.
- Ada pertanyaan tentang soal? Kirim email. Ambiguitas yang Anda temukan boleh Anda putuskan sendiri — cukup catat asumsinya di README, dan itu justru bernilai plus.

---

## 10. Sesi Defense (Setelah Submit)

Sesi live 45–60 menit. Kami akan membahas kode Anda, bukan menguji hafalan. Contoh arah pertanyaannya:

- "Tunjukkan alur request dari HTTP masuk sampai stok berkurang."
- "Kalau saya curi refresh token dari perangkat user, apa yang terjadi? Berapa lama saya bisa memakainya?"
- "Saya kirim 100 request checkout paralel untuk stok terakhir. Apa yang terjadi, dan bagaimana Anda tahu?"
- "Payment gateway mengirim webhook yang sama 3 kali, salah satunya dengan body yang diubah. Apa yang terjadi?"
- "Traffic naik 50x. Apa yang pecah duluan?"
- "Bagian mana dari kode ini yang paling Anda tidak sukai, dan kenapa?"

Kami menghargai jawaban "saya tidak tahu, tapi begini cara saya mencari tahunya."

---

**Submit ke:** `[dev@game-consign.com]` **Deadline:** `[11 September 2026, 23:59, WIB]` **Subject:** `Backend Test — [Nama Lengkap]`

Selamat mengerjakan. Kami lebih tertarik pada cara Anda berpikir daripada seberapa banyak yang Anda selesaikan.
