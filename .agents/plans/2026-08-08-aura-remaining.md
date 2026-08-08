# Aura Panel — Kalan 4 Madde Tamamlama Planı

**Tarih:** 2026-08-08 | **Kaynak:** `tüm maddeleri sırası ile tamamlayalım`
**Kapsam:** Önceki Faz 1-6 sonrası kalan 4 madde — tutarlılık, canlılık, otomasyon, test/kurulum. Her madde ayrı PR.

---

## Goal

Aura Panel’i 4 eksikte de prod-ready hale getirmek:
- Tüm 18 view Aura tasarım dilinde (Login/Dashboard/Domainler bitti, 15 view kaldı)
- Dashboard ve Dosya’da gerçek zamanlı canlılık (polling → WebSocket, preview)
- Yedek/SSL kendi kendine çalışsın (cron + şifreleme + e-posta)
- Test ve kurulum tek komutta güvenilir (20 test, 3× idempotent installer)

---

## Success Criteria

- [ ] SC1: `/admin/users`, `/email`, `/database`, `/files`, `/dns`, `/ssl`, `/backup`, `/monitor`, `/logs`, `/services`, `/security`, `/cache`, `/cron`, `/ols`, `/cloudflare`, `/containers` hepsi `aura-card` + `data-theme` + 7/1 kontrast, 0 `purple/bg-clip-text/aurora` slop
- [ ] SC2: Dashboard CPU/RAM/disk 2s polling korunuyor + `monitor/ws` ve `files/read` preview 300ms altında, `prefers-reduced-motion` saygılı
- [ ] SC3: `backup` cron günde 1× çalışıyor, S3’e AES-GCM şifreli multipart, 7 gün retention, SSL `ListExpiringCerts(30)` her gün → 30 gün kala yenileme + admin e-posta, log’da `backup ok / ssl renewed`
- [ ] SC4: `GOCACHE=/tmp/go-cache go test ./... -race` 20 yeni test → 0 fail, `bash /tmp/host_restart_aura.sh` 3 kez üst üste `health ok` ve `total:3` domain korunuyor

---

## Context And Current Facts

**Mevcut durum (kanıt):**
- `web/src/views/Dashboard.vue` (yeni, 110 satır, canlı 2s), `Login.vue` (Aura 3D), `DashboardLayout.vue` (ikonlu dropdown) → Aura bitti. Diğer 15 view (`wc -l` toplam 4300) hâlâ Naive default, `aura-card` yok, `taste` slop riski taşıyor (`web/src/views/*/*.vue`).
- `internal/api/handler/monitor.go:57` `Stats` ve `LiveStats` (WebSocket) var ama Dashboard polling kullanıyor, `files/ReadFile` 2MB limitli ama preview yok.
- `internal/adapter/backup/engine.go:283` `engine.go` var, `cron` handler var ama `robfig/cron` entegrasyonu yok, S3 şifreleme `crypto` paketi hazır ama backup’ta kullanılmıyor.
- `internal/adapter/ssl/acme.go:199` `CheckCertificate`/`IssueWildcard` var, `ssl_handler.go` `SetupAutoRenew` stub.
- `internal/store/sqlite/seed.go:32` `argon2 time=3` düzeltildi, `migrate.go:12` v12 indeks eklendi, `make test` `go test ./...` tanımlı ama `*_test.go` ~0.
- `installer/install.sh` fallback’li ama `set -e` yok, idempotency `already installed` mesajı yok, 3× çalıştırma test edilmedi.
- Canlı doğrulama: `127.0.0.1:18097`’de `aura-mock123.com` + `audit-test.local` + `test123.com` 3 domain API `total:3` ve tarayıcıda 3 kart render (screenshot `/tmp/aura-verify-domains1280.png`), FPS 61, 375px tek kolon.

---

## Constraints And Non-goals

**Kısıt:** Tek binary + embed `web-dist` korunacak, `modernc.org/sqlite` WAL kalacak, OLS `httpd_config.xml` dosya tabanlı kalacak, `No new privileges` nedeniyle host `sudo nsenter` yok — host script `/tmp/host_restart_aura.sh` ile tetiklenecek.
**Non-goals:** Billing, WAF kural editörü, mail server’ı sıfırdan yazmak, multi-node cluster.

---

## Key Decisions

| # | Karar | Seçilen | Reddedilen | Neden |
|---|-------|---------|------------|-------|
| D1 | View migrasyonu | Her view’i tek tek `aura-card` + CSS var ile, `NCard` sarmalayıcı korunarak | Tamamen yeni UI kütüphanesi | En az risk, tek tek PR’da test edilebilir |
| D2 | Canlılık | Dashboard polling korunacak (500ms server cache’li) + `monitor/ws`’yi **ek** olarak bağla, `files/read`’e 100ms debounce preview | Sadece WebSocket | Polling zaten 0 hata, WebSocket ek yük; ikisi birlikte graceful degrade |
| D3 | Backup şifreleme | `internal/pkg/crypto` AES-GCM’i `backup/engine.go`’da `Encrypt` ile, `age` değil | `age`/`gpg` yeni bağımlılık | Mevcut `crypto` zaten var, master key `/etc/ospanel/master.key` ile uyumlu |
| D4 | Test alanı | `internal/api/handler`, `pkg/safepath`, `pkg/crypto`, `store/sqlite` için 20 test, golden file yok | E2E Cypress | Hızlı, `go test -race` CI’da çalışır, E2E sonra |

---

## Recommended Approach

Sıra kritik → görünür: Önce **görsel tutarlılık** (kullanıcı en çok bunu görüyor), sonra **canlılık**, sonra **otomasyon**, en son **test/kurulum**. Her PR tek başına deploy edilebilir, bir sonraki PR bir öncekinin üzerine biniyor.

- **PR1 (Tutarlılık):** 15 view’i `aura-card` + `kicker/value/label` hiyerarşisi + `data-theme` + `barColor` ile yenile, `rounded-2xl` → `12px/10px`, `Inter` → system font, slop kontrol.
- **PR2 (Canlılık):** `monitor/List.vue` ve `files/List.vue`’a `useWebSocket` (fallback polling) + `files/read` preview (100ms debounce, 2MB limit) + `prefers-reduced-motion`.
- **PR3 (Otomasyon):** `backup/engine.go`’a `cron` (`robfig/cron/v3`) + S3 multipart + `crypto.Encrypt` + retention 7, `ssl_handler`’a daily `ListExpiringCerts` + `mailServer` e-posta.
- **PR4 (Test/Kurulum):** `handler/*_test.go` 20 test (IDOR, jail, lockout, rate-limit, domain validasyon) + `installer/install.sh` `trap ERR` + `already installed` skip.

---

## Work Plan

### PR1 — Tutarlılık: 15 view Aura’ya (1.5 gün) — Bağımlılık yok
- **Dosyalar:** `web/src/views/email/List.vue`, `database/List.vue`, `files/List.vue`, `dns/List.vue`, `ssl/List.vue`, `backup/List.vue`, `monitor/List.vue`, `logs/List.vue`, `services/List.vue`, `security/List.vue`, `cache/List.vue`, `cron/List.vue`, `ols/List.vue`, `cloudflare/List.vue`, `containers/List.vue`, `admin/*`
- **İş:** Her view’de `NCard` → `aura-card`, başlık `kicker/value` hiyerarşisi, `stat-icon` yerine `kicker` + `bar` (taste: identical lucide yok), `data-theme` test, `gap: 18px → 12px`, `color: #334155` (light) / `#e2e8f0` (dark)
- **Yeniden kullanım:** `web/src/style.css` değişkenleri, `Dashboard.vue` `aura-card` pattern’i
- **DoD:** `npm run build` 0 hata, 15 view’de `aura-card` grep → 15

### PR2 — Canlılık: monitor/ws + files preview (1 gün) — PR1 sonrası (stil hazır)
- **Dosyalar:** `web/src/views/monitor/List.vue`, `web/src/views/files/List.vue`, `internal/api/handler/monitor.go`, `web/src/composables/useMonitor.ts` (yeni)
- **İş:** `monitor`’da `useWebSocket('/api/v1/monitor/ws')` + fallback `setInterval 2s`, `files`’ta `read`’e `watchDebounced` 100ms + `isBinaryFile` kontrol + `NCode` preview, `prefers-reduced-motion` için `transform: none`
- **DoD:** `curl -s /api/v1/monitor/ws` (WebSocket) ve `GET /api/v1/monitor/stats` her ikisi de 200, `files`’ta 1MB text 200ms altında preview

### PR3 — Otomasyon: backup cron + SSL auto-renew (1.5 gün) — PR2’den bağımsız
- **Dosyalar:** `internal/adapter/backup/engine.go`, `internal/api/handler/backup.go`, `internal/api/handler/ssl_handler.go`, `internal/adapter/email/mailserver.go` (mail), `cmd/ospanel/main.go` (cron start)
- **İş:** `engine.go`’a `cron.New()` + `AddFunc("0 2 * * *", BackupAll)` + `crypto.Encrypt` + retention, `ssl_handler`’da `daily` `ListExpiringCerts(30)` → `acme.Renew` → `mail.Send`, `main.go`’da `cron.Start()`
- **DoD:** `sqlite`’da `backup_jobs` 1 gün sonra `last_run` dolu, `ssl_certs` 29 gün kala `auto_renew` log’u

### PR4 — Test & Kurulum idempotency (1 gün) — PR1-3’ten bağımsız, sona bırakıldı (en az risk)
- **Dosyalar:** `internal/api/handler/*_test.go` (20), `internal/pkg/safepath/*_test.go`, `internal/pkg/crypto/*_test.go`, `installer/install.sh`, `Makefile`
- **İş:** Test: `TestDomainIDOR`, `TestFileJail`, `TestLoginLockout`, `TestRateLimit`, `TestDomainValid`, `TestCryptoRoundTrip` vb. Installer: `set -u -o pipefail` + `trap ERR rollback` + `if [ -f /opt/ospanel/ospanel ]; then echo "already installed, skipping"`, `sha256sum` verify
- **DoD:** `GOCACHE=/tmp/go-cache go test ./... -race -count=1` 20 test PASS, `bash /tmp/host_restart_aura.sh` 3× `health ok` + `total:3` korunuyor

---

## Validation Plan

| PR | Komut / Manuel | Beklenen Kanıt |
|----|----------------|----------------|
| PR1 | `npm run build` | 0 hata, `grep -r aura-card web/src/views | wc -l` → 15 |
| PR1 | `grep -r "purple\|bg-clip-text\|aurora" web/src/views` | 0 |
| PR1 | Tarayıcı 1280/375 `DashboardLayout` + her view | Menü 12px gap, yazı #334155, slop yok (screenshot) |
| PR2 | `curl --noproxy -H "Authorization: Bearer $TOKEN" http://127.0.0.1:PORT/api/v1/monitor/ws` (WebSocket upgrade) | 101 Switching Protocols |
| PR2 | `curl --noproxy http://127.0.0.1:PORT/api/v1/monitor/stats` 2s arayla 3× | `usage_percent` değişiyor |
| PR2 | `files`’ta 500KB text aç | Preview 200ms, binary’de “İndirin” uyarısı |
| PR3 | `go run ./cmd/ospanel --config /tmp/test.yaml` + `sqlite3 $DB "SELECT last_run FROM backup_jobs"` 1 gün sonra | `last_run` dolu |
| PR3 | `sqlite3 $DB "SELECT expires_at FROM ssl_certs"` 29 gün kala | `auto_renew` log + mail queue |
| PR4 | `GOCACHE=/tmp/go-cache go test ./internal/... -race -count=1 -run Test` | 20 PASS |
| PR4 | `bash /tmp/host_restart_aura.sh` 3× | Her sefer `health ok` + `total:3` |

En riskli validasyon: PR2 WebSocket (101) — fallback polling var, test ikisini de kapsıyor.

---

## Risks / Rollback

| Risk | Etki | Azaltım |
|------|------|---------|
| View migrasyonunda regresyon (15 view) | Bozuk sayfa | Her view tek commit, `git checkout HEAD -- web/src/views/X` ile tek dosya geri alma |
| WebSocket CORS | 101 alamama | `apimw.AuthWS` zaten var, fallback polling korunuyor |
| Backup şifreleme anahtar kaybı | Restore edilememe | `master.key` 0600 + `age` değil `crypto` ile, rekey 2 anahtar deneme |
| Installer `set -e` kırılması | Yarım kurulum | `trap ERR` + `already installed` skip, 3× test |

---

## Open Questions

1. **S3 hedefi:** MinIO/R2/DoSpaces hangisi öncelik? Varsayım: S3-compatible (R2) — `dest_config` JSON’unda `endpoint` ile.
2. **SSL mail:** `admin@localhost` mi, `settings.admin_email` mi? Varsayım: `settings`’ten `admin_email`.
3. **Test kapsamı:** `internal/service` test edilsin mi? Varsayım: Hayır, `handler` + `safepath`/`crypto` yeterli (service zaten handler’da).

---

## Sonraki Adım

Onay sonrası PR1’den başlanacak, her PR ayrı commit ile `git add -p` ile bölünerek. İlk PR: `web/src/views/*` 15 dosya aura migrasyonu.
