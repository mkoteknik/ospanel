# OpenSpeed Panel — Senior Seviye Kusursuzlaştırma Planı

**Tarih:** 2026-08-08 | **Versiyon:** v0.1.0-dev | **Stack:** Go 1.26.5 + Vue3/TS + SQLite (WAL) + OpenLiteSpeed
**Hedef:** Stabilite, güvenlik (en üst seviye), hız ve OLS %100 entegrasyon ile üretime hazır, cPanel/Plesk ayarında panel

---

## Goal

`mkoteknik/ospanel`’i tek sunucuda 10k+ vhost, 100+ eşzamanlı panel kullanıcısı altında:
- **Güvenlik:** OWASP Top-10’a dayanıklı, tenant izolasyonlu, audit’li
- **Stabilite:** Parçalı domain oluşturma yok, config bozulması yok, SQLite race’i yok, graceful degrade
- **Hız:** API p95 < 80ms (panel), dosya listeleme 10k dosyada < 300ms, monitoring overhead <%1 CPU
- **OLS:** vhost/SSL/PHP/listener/rewrite/log %100 çift yönlü senkron

Bu plan **kod yazmadan** karar-complete yol haritasıdır. Her faz kendi içinde test edilebilir, geri alınabilir.

---

## Success Criteria

- [ ] `gosec` + `golangci-lint` 0 high, `npm audit` 0 critical, OWASP ZAP baseline 0 high (SC1)
- [ ] Dosya Terminal ve FileManager üzerinden tenant dışına çıkılamıyor, fuzz testleri PASS (SC2)
- [ ] Domain create: OLS vhost + DNS(PowerDNS) + Mail(DKIM/SPF/DMARC) + docRoot ya hep ya hiç (compensating tx), yarım kalan domain yok (SC3)
- [ ] `httpd_config.xml` ve `vhconf.xml` atomik yazılıyor, corrupt config durumunda otomatik rollback + OLS config test (`lshttpd -t`) geçmeden reload yok (SC4)
- [ ] Auth: brute-force lockout çalışıyor, JWT refresh rotation + revoke, 2FA recovery codes, password policy + Argon2id parametreleri OWASP önerisi (SC5)
- [ ] 500 concurrent `ListDomains` + `GetSystemStats` altında p95 < 100ms, `getCPUUsage` request thread’ini bloke etmiyor (SC6)
- [ ] Ön yüzde XSS’e karşı token httpOnly/SameSite, RBAC guard hem FE hem BE’de eş, canlı yenileme (WS) auth’lı (SC7)
- [ ] Tek komut installer idempotent, 3 kez üst üste çalıştırıldığında aynı sonuç, hata durumunda önceki sürüme rollback (SC8)

---

## Context And Current Facts

**Kanıtlı mevcut durum (dosya: satır):**

| Alan | Dosya | Durum |
|------|-------|-------|
| Entry | `cmd/ospanel/main.go:45` | Config yoksa `Default()` + random JWT secret her restart’ta değişir → tüm tokenlar geçersiz |
| JWT | `internal/api/middleware/auth.go:27` + `handler/auth.go:270` | HS256, MapClaims float64 cast, `type` claim hiç kontrol edilmiyor, refresh de aynı secret, logout stateless (revoke yok) |
| Login | `handler/auth.go:45` | `LoginAttempts` artırılıyor ama `LockUser` hiç çağrılmıyor, `MaxLoginAttempts/LockoutDuration` config ölü kod |
| 2FA | `handler/auth.go:112` + `twofa.go:14` | İki ayrı TOTP implementasyonu, `Setup2FA` içinde `TODO-generate-secret`, production placeholder |
| RBAC | `middleware/rbac.go:12` + `api/router.go:110` | `/admin/*` korumalı ama diğer tüm kaynaklar (domain, db, file) `userID` kontrolü yapmadan `List` → IDOR riski |
| RateLimit | `middleware/ratelimit.go:12` | Global 100r/s burst 200, memory map + `X-Forwarded-For` spoof’lanabilir, per-endpoint (login 5/dk) yok |
| CORS/Sec | `middleware/global.go:14` | `Allow-Origin: *` + `Allow-Headers: Authorization` → credential’lı istekte wildcard tehlikeli, HSTS yok, CSP yok |
| Audit | `middleware/audit.go:18` | Sadece POST/PUT/DELETE, `details=RawQuery`, body loglanmıyor, senkron DB yazımı request’i yavaşlatıyor |
| File | `handler/file.go:34` | `path = Clean(path)` + `Contains("..")` → `//etc/passwd`, `/etc/passwd`, symlink bypass açık; `absPath` kullanıcı home’u ile sınırlandırılmıyor, 50MB upload `io.Copy` disk fill |
| Terminal | `handler/terminal.go:22` | WS `CheckOrigin:true`, auth header yok, `bash --login` root olarak, stdin pipe doğrudan WS → RCE, PTY yok |
| Domain | `handler/domain.go:34` | `isValidDomain` sadece uzunluk+dot, `CreateVHost` string replace ile `httpd_config.xml`’e ekleme, dosya lock yok, reload senkron |
| OLS | `adapter/ols/client.go:54` | `IsAvailable = Stat(binPath)` → process çalışmıyor olsa da true, `generateVHostConfig` XML injection açık, `addVHostToMainConfig` `</virtualHostList>` replace atomik değil |
| DB pass | `handler/database.go:42` | `PasswordEnc = req.Password` plaintext, `adapter/database/mysql.go` içinde de şifreleme yok |
| System | `adapter/system/filesystem.go:120` | `getCPUUsage` içinde `time.Sleep(100ms)` → handler goroutine bloke |
| SQLite | `store/sqlite/sqlite.go:20` | `MaxOpenConns=1` doğru ama DSN `busy_timeout=5000` tek, migration `ALTER TABLE` idempotency zayıf (`migrate.go:11`) |
| Service | `internal/service/` | Boş → tüm iş mantığı handler’da, transaction yok |
| FE auth | `web/src/stores/auth.ts:7` + `api/client.ts:14` | Token `localStorage`, XSS çalınabilir, interceptor concurrent refresh race, router guard sadece `isAuthenticated` bool |
| RBAC FE | `web/src/router/index.ts:82` | `admin/*` için `role` kontrolü yok, BE 403’e güveniyor → UX sızıntısı |
| Build | `web/node_modules/` | Repoda commitli, `.gitignore:356` kontrol edilmeli, `go.mod` 1.26.5 çok yeni (CI uyumu) |

**Mimari:** `handler → store + adapter` direkt, servis yok. Toplam 8889 satır, en büyük dosya `domain.go:672` satır (tanrı handler).
**Installer:** `installer/install.sh` 3+ fallback stratejili, ancak idempotency/rollback zayıf, OLS 1.9.1 deb fallback hardcoded.
**Test:** `make test` tanımlı ama test dosyası sayısı ~0 (grep ile *_test.go yok).

---

## Constraints And Non-goals

**Constraints:**
- Tek binary + embed SPA (`cmd/ospanel:web-dist`) korunmalı, kurulum tek komut kalmalı
- SQLite dışı DB’ye geçiş yok (WAL kalacak), ancak Postgres adaptörü korunacak
- OLS’in dosya tabanlı `httpd_config.xml` mimarisi değiştirilemez, üzerine güvenli katman eklenecek
- Panel `root` ile çalışabilir ama tenant izolasyonu için OLS/docRoot seviyesinde drop-privilege şart

**Non-goals (bu planda yapılmayacak):**
- Multi-node cluster / horizontal scale
- Billing / ödeme entegrasyonu
- WAF kural editörü (ModSecurity sadece aç/kapa)
- Mail server’ı sıfırdan yazmak (Postfix/Dovecot adaptörü iyileştirme)

---

## Key Decisions

| # | Karar | Seçilen | Reddedilen Alternatif | Neden |
|---|-------|---------|----------------------|-------|
| D1 | Auth token saklama | BE: httpOnly Secure cookie + SameSite=Lax (header fallback), FE: memory only | localStorage kalması | XSS ile token çalınmasını kapatır; 15+ yıl sektör standardı |
| D2 | JWT revoke | Refresh rotation + `jti` + SQLite `refresh_tokens` tablosu (hashlenmiş) | Stateless kalıp logout’un no-op kalması | Çalınan refresh’in tek kullanımlık olması, logout/şifre değişiminde global revoke |
| D3 | File/Terminal izolasyonu | Her kullanıcı → `chroot`-benzeri `homeDir` jail + `filepath.Rel` + symlink `EvalSymlinks` + allowlist | Sadece `Contains("..")` | Path traversal ve symlink RCE’yi kapatmanın tek gerçek yolu |
| D4 | Servis katmanı | `internal/service` doldurulacak, handler sadece HTTP, servis transactional | Handler’da iş mantığı kalması | Test edilebilirlik, rollback, OLS/DNS/Mail compensating transaction için şart |
| D5 | OLS config yazımı | `WriteFileAtomic` (tmp+fsync+rename) + file lock (`flock`) + `lshttpd -t` öncesi commit + backup `.bak` | Direkt `os.WriteFile` + string replace | Corrupt `httpd_config.xml` tüm sunucuyu düşürüyor, en kritik stabilite riski |
| D6 | Rate limit | `golang.org/x/time/rate` per-IP + per-endpoint (login 5/dk, file/upload 20/dk) + trusted proxy list | Mevcut global map | Login brute-force ve upload DoS’u gerçekçi kapatır, XFF spoof’u engeller |
| D7 | DB şifreleri | `age`/`libsodium` sealed box ile `password_enc` gerçekten şifreli + `OSPANEL_MASTER_KEY` env/file | Plaintext kalması | Hosting panelinde DB şifresi sızıntısı tam felaket |
| D8 | Performans - CPU | `getCPUUsage` 100ms sleep’i kaldır, `/proc/stat` 2 örnek arası delta’yı background ticker’da hesapla (1s) | Request içinde sleep | p95’i doğrudan vuran blokaj |
| D9 | Frontend paket | `node_modules` repodan çıkar + `package-lock.json` kilit + Vite `manualChunks` (naive-ui, vendor) | Commitli kalması | CI hızını ve supply-chain güvenliğini düzeltir |

---

## Recommended Approach

**Sıra kritik → nitelikli:** Önce **güvenlik açığını kapat** (Terminal, File, Auth), sonra **veri bütünlüğünü** (servis+tx), sonra **OLS atomikliği**, en son **performans/UX/operasyon**.

Teknik omurga:
1. **Güvenlik duvarı:** `internal/pkg/safePath` (jail resolver) + `middleware.Auth`’u WS için query-cookie-token’a genişlet + `internal/auth` paketi (JWT issue/verify/rotate/revoke, Argon2id cost 3 iter 64MB, zxcvbn policy).
2. **Servis:** `service/domain.Service.Create(ctx, req)` içinde `sql.Tx` (BEGIN → CreateDomain → OLS CreateVHost → PowerDNS → Mail → COMMIT; hata → compensating delete + rollback). Aynı pattern DB, Email, SSL için.
3. **OLS:** `ols.Client`’a `atomicWrite`, `withLock`, `validateDomain` (RFC1035 regex + punycode + reserved), `renderVHost` (xml-escape + template), `ensureListener` (80/443 mapping), `testAndReload` adımları eklenecek. Her yazım öncesi `cp httpd_config.xml httpd_config.xml.bak.<ts>`.
4. **Hız:** `system.StatsCollector` background goroutine (1s tick), `store`’a indexler (`idx_domains_user_id`, `idx_audit_created_at`, `idx_dns_domain_id`), file list için `readdir` + pagination (limit/offset, 500 max).
5. **FE:** `api/client` → refresh mutex (`p-queue`), token memory, `router` RBAC guard (`meta.roles`), `Content-Security-Policy` meta, `Naive UI` treeshake.

Mevcut kodu **yeniden yazma değil**, katman ekleyerek sertleştirme (handler imzası korunacak, içerde service’e delegate).

---

## Work Plan

### Faz 0 — Hazırlık (0.5 gün) — Bağımlılık yok
- **0.1** `git stash` + `build` baseline al: `make test`, `golangci-lint run ./...`, `npm audit`, `gosec ./...`
- **0.2** `.gitignore` düzelt: `web/node_modules/`, `build/`, `*.db`, `coverage.*` ekle; commitli `node_modules`’i history’den çıkarmadan `.gitignore` ile yok say
- **0.3** Feature flag: `config.Security.StrictFileJail` (default false → faz 1 sonunda true) ekle ki kademeli açılabilsin
- **DoD:** `git status` temiz, lint baseline raporu kaydedildi

### Faz 1 — Kritik Güvenlik (P0) — 3-4 gün — **En yüksek risk**
- **1.1 Terminal RCE kapat:** `handler/terminal.go`
  - `CheckOrigin: false` → allowlist (panel origin)
  - WS upgrade öncesi `AuthMiddleware`’i query `?token=` + cookie’den doğrula (`middleware.AuthWS`)
  - `exec.Command("bash","--login")` → `user.Lookup` ile hedef kullanıcıya `syscall.Credential` + `PTY` (`creack/pty`) + `Allow *` kaldır, `TMOUT=600` + max 1 shell/user + audit log
  - `getClientIP` trusted proxy olmadan XFF’den alma → `config.Server.TrustedProxies []string`
- **1.2 FileManager jail:** `handler/file.go` + yeni `internal/pkg/safepath`
  - Tüm `Read/Write/Delete/Chmod/Rename/Upload/Download/Archive/Extract` → `safepath.Resolve(homeDir, userInput)` (Clean+Join+EvalSymlinks+HasPrefix)
  - Upload: `http.DetectContentType` + extension allowlist + max 50MB per-user quota + `io.LimitReader` + temp file + `os.Rename`
  - ZipSlip zaten `ExtractTarGz`’de var, `ExtractZip`’e de aynı `HasPrefix` ekle + symlink skip
  - `List` için `homeDir` dışına çıkma → 403
- **1.3 Auth/JWT sertleştirme:** `handler/auth.go`, `middleware/auth.go`, `pkg/config`
  - `generateRandomSecret` fallback kaldır, `Load` yoksa `/etc/ospanel/jwt.key`’den oku/yaz (0600), restart’ta değişmesin
  - `hashPassword` → `argon2.IDKey(3, 64*1024, 4, 32)` + `zxcvbn` min score 2, `ChangePassword` policy uygula
  - `Login`: `UpdateLoginAttempts` + `LockUser` gerçekten çağır (5 deneme → 30dk kilit), `LastLogin` güncelleme hatası yutulmasın
  - JWT: `RegisteredClaims` (iss=`ospanel`, aud=`ospanel`, jti=uuid, exp/iat), `type` kontrolü, `ParseWithClaims` + `WithIssuer/Audience`
  - Refresh: `refresh_tokens` tablosu (id, user_id, jti_hash, expires_at, revoked), rotation (eski revoke), `Logout` revoke, `RefreshToken` eskiyi tek kullanımlık yap
  - `TOTP`: `twofa.go` tek kaynak, `auth.go:Setup2FA` TODO kaldır, recovery codes (10×8 hex, bcrypt hash) ekle
- **1.4 Global sec headers:** `middleware/global.go`
  - CORS: `*` → `config.Server.AllowedOrigins` allowlist, `AllowCredentials` only if origin match
  - Ekle: `Strict-Transport-Security` (TLS ise), `Content-Security-Policy`, `Permissions-Policy`, `Cross-Origin-*`
- **1.5 Database password:** `handler/database.go` + `store` + `adapter/database`
  - `PasswordEnc` → `crypto/aead` (XChaCha20-Poly1305) + master key (`/etc/ospanel/master.key` 0600 veya `OSPANEL_MASTER_KEY` env)
  - Migration: mevcut plaintext’leri re-encrypt job’u (offline)
- **DoD/Validasyon:** `gosec -fmt json` 0 high, ZAP baseline, `go test ./internal/api/middleware -run TestAuth` + manuel: `curl --path-as-is /files?path=/etc/passwd` → 403, WS `wss://.../terminal/ws` tokensiz → 401

### Faz 2 — Veri & İşlem Bütünlüğü (P1) — 3 gün — Faz 1 sonrası
- **2.1 Servis katmanı iskeleti:** `internal/service/domain.go`, `email.go`, `database.go`, `ssl.go`
  - Interface → `DomainService{ Create(ctx, userID, req) (*Domain, *AutoSetupResult, error) }` (handler sadece `service.Create` çağırır)
  - Handler’dan taşınacak: validasyon, quota (`settings.max_domains_per_user`), `CreateDocumentRoot`, OLS/PowerDNS/Mail orkestrasyonu
- **2.2 Transaction + compensating:** `store/sqlite`’a `WithTx(ctx, func(tx *sql.Tx) error)` helper
  - Domain create: `BEGIN` → `INSERT domains` → `CreateVHost` (hata → `DELETE` + `ROLLBACK`) → `CreateZone` (hata → `DeleteVHost` + rollback) → `COMMIT`
  - İdempotency key: `domain` UNIQUE zaten var, race için `INSERT OR FAIL` + 409
- **2.3 Input validasyon:** `internal/pkg/validate` (go-playground/validator)
  - Domain: `^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$` + punycode + reserved (`localhost`, `test`)
  - PHP: sadece `8.2/8.3/8.4` (config’den), Email, DB name `^[a-zA-Z0-9_]{3,64}$`
  - Tüm handler’larda `validate.Struct(req)` → 422 detaylı hata
- **2.4 RBAC + IDOR:** `handler/domain.go:Get/Update/Delete` → `if domain.UserID != callerID && callerRole != admin → 403`, `List` zaten `userID` filtreli ama `GetDomainByName` bypass’ı kapat
- **2.5 Audit async:** `middleware/audit.go` → buffered channel + worker goroutine (1000 cap, drop+log), body’den `password` redaction
- **DoD:** `go test ./internal/service -race` tüm happy+rollback path’leri, `make test-race` yeşil

### Faz 3 — OLS %100 Entegrasyon (P1) — 4 gün — Faz 2 ile paralel başlanabilir (dosya lock bağımsız)
- **3.1 Atomic FS:** `internal/pkg/atomicfile.WriteFileAtomic(path, data, perm)` (tmp+fsync dir+rename) + `internal/pkg/flock`
- **3.2 OLS client sertleştirme:** `adapter/ols/client.go`
  - `IsAvailable` → `Stat` + `pgrep lshttpd` + `ss -tlnp :7080` üçlü
  - `generateVHostConfig` → `html.EscapeString` + `template` strict, `docRoot` absolute check, `phpVersion` allowlist
  - `add/removeVHostToMainConfig` → `xml.Decoder` ile parse, `virtualHostList` append, lock + atomic write + backup
  - `SetPHPVersion` → `lsphp` string replace değil, XML node update
  - Yeni: `EnsureListener(domain, ssl bool)` (80/443 listener mapping), `SetSSL(domain, certPath, keyPath)`, `SetRewrite(domain, htaccess bool)`, `SetAccessLog`
- **3.3 Config test gate:** `Reload` → `lshttpd -t` fail → rollback + error dön, success → `lshttpd -r` + `systemctl is-active lsws` poll 5s
- **3.4 OLS test harness:** `adapter/ols/client_test.go` (temp `confDir/vhostsDir` ile `Create/Delete/SetPHP` ve corrupt `httpd_config.xml` rollback testi)
- **3.5 PHP handler keşfi:** `GetPHPHandlers` → `ls -d /usr/local/lsws/lsphp*` dinamik, kurulu değilse handler oluşturma 4xx
- **DoD:** Linux VM’de (Ubuntu 22.04) 50 ardışık domain create/delete döngüsünde `httpd_config.xml` xmllint valid, OLS reload 0 fail

### Faz 4 — Stabilite & Performans (P2) — 2-3 gün
- **4.1 HTTP server tuning:** `cmd/ospanel/main.go`
  - `ReadHeaderTimeout` 5s ekle (Slowloris), `WriteTimeout` 30s → 60s (backup), `MaxHeaderBytes` 8KB (1MB çok yüksek)
  - TLS: `CurvePreferences` + `CipherSuites` zaten iyi, ek: `ClientAuth: NoClientCert`, `SessionTicketsDisabled: false` + HSTS header
- **4.2 Stats collector:** `adapter/system/filesystem.go`
  - `GetSystemStats` içindeki `Sleep(100ms)` → `StatsCollector` singleton (ticker 1s, cache `atomic.Value`), handler anında cache döner
  - `DiskUsage` → `du` yerine `Statfs` + per-domain quota cache (5m)
- **4.3 SQLite índices:** `store/sqlite/migrate.go` v12
  - `idx_domains_user_domain(user_id, domain)`, `idx_audit_user_created(user_id, created_at)`, `idx_dns_domain_id`, `idx_backup_user`, `idx_ssl_expires(expires_at)`
  - `VACUUM` + `PRAGMA journal_size_limit=64MB` + `auto_vacuum=INCREMENTAL`
- **4.4 RateLimit v2:** `middleware/ratelimit.go` → `x/time/rate` + `sync.Map` + `TrustedProxies` CIDR check, per-route: `login 5/m`, `refresh 10/m`, `upload 20/m/10MB`, `terminal/ws 20/m`
- **4.5 Context timeout:** Tüm `adapter/*` çağrılarına `context.WithTimeout(8s)`, `http.Client{Timeout:5s}`, DNS/Mail OLS çağrıları paralel değil sıralı + retry 1 (idempotent olanlar)
- **DoD:** `go test -bench` (stats 10k req/s, p95 < 50ms), `wrk -t4 -c100 /api/v1/monitor/stats` 30s error 0

### Faz 5 — Frontend & UX Sertleştirme (P2) — 2 gün
- **5.1 Token & XSS:** `web/src/api/client.ts`
  - Token memory + `httpOnly` cookie tercih (BE `Set-Cookie`), `localStorage` kaldır, `refresh` mutex (`isRefreshing` + queue)
  - `axios` → `withCredentials: true`, CSRF: `X-CSRF-Token` (double submit cookie) login sonrası
- **5.2 RBAC guard:** `web/src/router/index.ts`
  - `meta.roles: ['admin']` + `beforeEach` → `if requiresAuth && !hasRole → /` + `useAuthStore.fetchMe()` ile role tazele
  - `DashboardLayout.vue` menüyü role’e göre filtrele (admin link’lerini gizle)
- **5.3 Paket & hız:** `web/package.json`
  - `node_modules` repodan çıkar, `vite.config.ts` → `build.rollupOptions.output.manualChunks: { vendor, naive: ['naive-ui'] }`, `chunkSizeWarningLimit 800`
  - `views/files/List.vue` → pagination + virtual scroll (naive `n-data-table` remote), `monitor` WS → `useWebSocket` + exponential backoff
- **5.4 Hata UX:** `api/client` error mapping (422 → field errors, 429 → Retry-After toast), `Login.vue` → lockout geri sayım + 2FA recovery code input
- **DoD:** Lighthouse perf >90, `npm run build` chunk < 300KB (gz), XSS: `"><script>` domain adı → escape, WS `monitor/ws` tokensiz → 401

### Faz 6 — Gözlemlenebilirlik & Operasyon (P3) — 2 gün
- **6.1 Logging:** `internal/pkg/logger`
  - JSON + `trace_id` (X-Request-ID), `zap` sampling (100/s), audit için ayrı `audit.log` (rotating `lumberjack`)
  - Tüm `h.log.Errorw` → `Errorw` + `request_id` + `user_id`
- **6.2 Health & metrics:** `api/router.go`
  - `/health` → DB ping + OLS `lshttpd -t` + disk free >10% → 200 else 503
  - `/metrics` (Prometheus, sadece `127.0.0.1` + admin) → `go_*`, `http_requests_total`, `ols_reload_total`, `sqlite_busy_total`
- **6.3 Backup/SSL cron:** `handler/cron.go` + `adapter/backup/engine.go`
  - Cron `robfig/cron/v3` ile `backup_jobs.schedule` → worker pool (2), S3 multipart + AES-GCM encrypt, retention sweep
  - SSL auto-renew: daily `ListExpiringCerts(30)` → `acme.Renew` → OLS reload, failure → audit + email admin
- **6.4 Installer idempotency:** `installer/install.sh`
  - Her adım `if already_installed → skip`, `set -e` kaldır → `trap ERR` + `rollback()` (önceki binary `.bak`), `ospanel --version` sonrası `systemctl is-active`
  - OLS `.deb` checksum verify (`sha256sum`), `packages/` fallback önceliği korunacak
- **DoD:** `systemctl kill ospanel; systemctl start` sonrası 5s içinde `/health` 200, log rotate çalışıyor, backup dry-run PASS

---

## Validation Plan

| Faz | Komut / Manuel | Beklenen Kanıt |
|-----|----------------|----------------|
| 0 | `go vet ./... && golangci-lint run ./...` | 0 error, baseline dosyası `.agents/lint-baseline.txt` |
| 1 | `gosec ./... -fmt json` | 0 high (file/terminal/auth) |
| 1 | `go test ./internal/api/handler -run TestSafePath -count=20` | `../`, `/etc/passwd`, `symlink /tmp->/etc` → 403 |
| 1 | `curl -H "X-Forwarded-For: 1.1.1.1" /api/v1/auth/login` brute 6 kez | 6. istek 429 + `Retry-After`, 7. istek 30dk sonra 200 |
| 1 | `wscat -c ws://localhost:8090/api/v1/terminal/ws` tokensiz | 401, tokenli → PTY prompt |
| 2 | `go test ./internal/service -race -run TestDomainCreateRollback` | DB rollback + vhost silinmesi assert |
| 2 | `go test ./internal/pkg/validate -run TestDomain` | `"-bad.com"`, `"a..b.com"` → 422 |
| 3 | `go test ./internal/adapter/ols -run TestAtomic -count=50` | 50 concurrent Create/Delete → xmllint valid + backup sayısı 50 |
| 3 | Linux VM: `for i in {1..20}; do curl -X POST /api/v1/domains -d '{"domain":"t'$i'.com"}' ; done` | 20 vhost `vhconf.xml` valid, `lshttpd -t` ok, `ss -tlnp | grep 80` mapping var |
| 4 | `wrk -t4 -c50 -d30s http://localhost:8090/api/v1/monitor/stats -H "Authorization: Bearer $TOKEN"` | p95 < 100ms, 0 timeout, CPU `top` % <5 |
| 4 | `go test -run TestStatsCollector -count=1` | 2 çağrı arası 1ms (cache hit) |
| 5 | `npm run build && du -sh web/dist` | total < 2MB gz, chunk naive < 300KB |
| 5 | Browser: `Login` → LS `localStorage.getItem('access_token')` | `null` (memory only) |
| 6 | `curl localhost:8090/health` + `curl localhost:8090/metrics` | 200 + Prometheus exposition |
| 6 | `bash installer/install.sh` 2. kez | `already installed, skipping` + exit 0, `systemctl status ospanel` active |

**En riskli validasyon:** Faz 1 WS+File jail (manuel fuzz olmadan otomatik test yetersiz) → `internal/pkg/safepath` için 100+ case’lik table-driven + symlink tempFS testi + `gofuzz` ekle.

---

## Risks / Rollback

| Risk | Etki | Azaltım / Rollback |
|------|------|-------------------|
| JWT cookie’e geçiş FE’yi kırar | Login loop | Dual-mode 1 sürüm: hem header hem cookie kabul, FE feature flag `VITE_AUTH_MODE` |
| File jail mevcut kullanıcı dosyalarına erişimi engeller | 500 | `StrictFileJail=false` config ile kademeli rollout, per-user allowlist `/var/lib/ospanel/jail-exceptions.json` |
| OLS atomic write bug’ı config’i bozar | Tüm vhost down | Her write öncesi `httpd_config.xml.bak.<ts>` + `lshttpd -t` fail → otomatik `cp .bak` + alert |
| Master key rotasyonu DB şifrelerini okunamaz yapar | DB decrypt fail | Rekey job: eski key ile decrypt → yeni key ile encrypt, 2 key aynı anda dene (key version header) |
| SQLite index migration lock | Kısa downtime | `CREATE INDEX IF NOT EXISTS` + `PRAGMA busy_timeout` + maintenance window (gece 03:00) |

---

## Open Questions

1. **Panel çalıştırma kullanıcısı:** `root` mu kalacak, `ospanel` system user + `sudo` whitelist mi? Karar → Faz 1 terminal jail için kritik. *Öneri:* `ospanel` user + `cap_dac_override` + OLS `lshttpd` root kalır, panel `ospanel:ospanel` 0750 docRoot.
2. **Master key yönetimi:** `/etc/ospanel/master.key` file mı, Vault/KMS mi? Tek sunucu için file + 0600 yeterli, ama backup’ı nasıl? *Varsayım:* file, installer’da `openssl rand 32` ile oluştur + `chmod 600`.
3. **PowerDNS/Mail zorunlu mu?** Yoksa domain create DNS/Mail adımlarını skip mi etmeli? *Mevcut:* `IsAvailable()` ile skip zaten var, Faz 2’de skip durumu explicit `auto_setup.skipped_reason` dönecek.
4. **HSTS preload:** `ForceHTTPS` true iken HSTS header’ı panel mi OLS vhost mu ekleyecek? *Öneri:* OLS vhost’a `Header set Strict-Transport-Security` ekle, panel sadece 308 redirect.
5. **Backup hedefleri:** S3, FTP, local dışında Google Drive isteniyor mu? *Varsayım:* Faz 6’da S3-compatible (MinIO, R2) öncelik.

---

## Sonraki Adım

Bu plan onaylanırsa **Faz 1’den başlayarak** her faz **ayrı commit/PR** olarak ilerleyecek (plan skill kuralı gereği split korunacak). İlk PR: `safepath + terminal auth + JWT master key` (en kritik 3 dosya, ~400 satır).

Onay bekliyor — değişiklik istersen faz sırası/kapsamı güncellenir.
