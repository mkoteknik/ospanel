# ⚡ OpenSpeed Panel

<div align="center">

**Modern, Hızlı, %100 Ücretsiz Hosting Kontrol Paneli**

[![Build](https://img.shields.io/github/actions/workflow/status/mkoteknik/ospanel/build.yml)](https://github.com/mkoteknik/ospanel/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mkoteknik/ospanel)](https://go.dev)
[![License](https://img.shields.io/github/license/mkoteknik/ospanel)](LICENSE)
[![Release](https://img.shields.io/github/v/release/mkoteknik/ospanel)](https://github.com/mkoteknik/ospanel/releases)

</div>

---

## 🎯 Nedir?

OpenSpeed Panel, **OpenLiteSpeed** web sunucusu ile %100 uyumlu, cPanel/Plesk alternatifi, tamamen ücretsiz ve açık kaynaklı bir hosting kontrol panelidir.

### Neden OpenSpeed Panel?

| | cPanel / Plesk | OpenSpeed Panel |
|---|---|---|
| **Lisans** | $15-45/ay | **Tamamen Ücretsiz** 🎉 |
| **Web Sunucu** | Apache/Nginx | **OpenLiteSpeed** (HTTP/3, QUIC) |
| **Kaynak** | 1GB+ RAM | **256MB RAM** ile çalışır |
| **Teknoloji** | Perl/PHP (legacy) | **Go + Vue 3** (modern) |
| **Kurulum** | Karmaşık, uzun | **Tek komut**, 30 saniye |
| **API** | Sınırlı | **Tam RESTful + WebSocket** |

## ✨ Özellikler

- 🌐 **Domain Yönetimi** - Virtual host, PHP sürüm seçimi (LSAPI)
- 📧 **E-Posta** - Hesap yönetimi, forwarder, DKIM/SPF/DMARC
- 🗄️ **Veritabanı** - MariaDB/MySQL yönetimi, phpMyAdmin
- 📁 **Dosya Yöneticisi** - Web tabanlı kod editörlü
- 🔒 **SSL/TLS** - Let's Encrypt otomatik, wildcard desteği
- 🔧 **DNS** - Zone yönetimi, tüm kayıt tipleri
- 💾 **Yedekleme** - Otomatik zamanlamalı, uzak hedef
- 📈 **Monitoring** - Gerçek zamanlı CPU/RAM/Disk
- 🛡️ **Güvenlik** - Fail2ban, ModSecurity, 2FA, rate limiting
- 🖥️ **Web Terminal** - Tarayıcıdan SSH erişimi

## 🚀 Tek Komut Kurulum

```bash
curl -fsSL https://raw.githubusercontent.com/mkoteknik/ospanel/main/install.sh | sudo bash
```

### Desteklenen İşletim Sistemleri

| OS | Sürümler |
|---|---|
| Ubuntu | 20.04, 22.04, 24.04 |
| Debian | 11, 12 |
| Rocky Linux | 8, 9 |
| AlmaLinux | 8, 9 |

## 🛠️ Geliştirme

```bash
# Repoyu klonla
git clone https://github.com/mkoteknik/ospanel.git
cd ospanel

# Gereksinimler: Go 1.22+, Node.js 20+

# Backend başlat
make dev-backend

# Frontend başlat (ayrı terminal)
make dev-frontend

# Build
make build
```

## 📦 Proje Yapısı

```
ospanel/
├── cmd/ospanel/       # Go ana uygulama (entry point)
├── internal/
│   ├── api/           # HTTP API (router, handler, middleware)
│   ├── service/       # İş mantığı katmanı
│   ├── adapter/       # Harici servis entegrasyonları (OLS, MySQL, DNS)
│   ├── store/         # Veritabanı katmanı (SQLite)
│   ├── model/         # Veri modelleri
│   └── pkg/           # Paylaşımlı yardımcılar
├── web/               # Vue 3 + TypeScript frontend
├── installer/         # Kurulum scriptleri
└── docs/              # Dokümantasyon
```

## 🔧 Teknoloji Yığını

| Katman | Teknoloji |
|---|---|
| Backend | Go 1.22+ (chi router, zap logger) |
| Frontend | Vue 3 + TypeScript + Naive UI + Vite |
| Veritabanı | SQLite (WAL modu) |
| Web Sunucu | OpenLiteSpeed |
| Auth | JWT (HS256), Argon2id |
| Build | Vite, Go embed |

## 🤝 Katkıda Bulunma

Katkılarınızı bekliyoruz! Lütfen [CONTRIBUTING.md](docs/CONTRIBUTING.md) dosyasını okuyun.

1. Fork'layın
2. Feature branch oluşturun (`git checkout -b feature/harika-ozellik`)
3. Commit'leyin (`git commit -m 'Harika özellik eklendi'`)
4. Push'layın (`git push origin feature/harika-ozellik`)
5. Pull Request açın

## 📄 Lisans

MIT License - detaylar için [LICENSE](LICENSE) dosyasına bakın.

---

<div align="center">
Made with ❤️ by the OpenSpeed Panel community
</div>
