# OpenSpeed Panel - Bağımlılık Paketleri

Bu klasör, panelin ihtiyaç duyduğu tüm kritik paketleri içerir.
Paketler test edilmiş ve panel ile uyumlu sürümlerdir.

## Paketleri İndirme

Linux sunucuda çalıştırın:
```bash
sudo bash packages/download.sh
```

Bu script tüm .deb paketlerini ve bağımlılıklarını indirir.

## İndirilen Paketler

| Paket | Açıklama |
|---|---|
| openlitespeed.deb | Web sunucusu |
| lsphp8*.deb | PHP LSAPI sürümleri |
| pdns-*.deb | PowerDNS + SQLite backend |
| postfix*.deb | Email SMTP |
| dovecot*.deb | Email IMAP/POP3 |
| mariadb*.deb | Veritabanı |
| redis*.deb | Cache |
| fail2ban*.deb | Güvenlik |
| adminer.php | Veritabanı yöneticisi |

## Neden Repoda?

- Kurulum sırasında dış bağımlılık yok
- Her sunucuda aynı sürüm kurulur
- İnternet bağlantısı kesilse bile kurulum yapılabilir
- Sürüm güncellemeleri kontrollü yapılır
