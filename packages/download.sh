#!/usr/bin/env bash
#
# OpenSpeed Panel - Bağımlılık Paketlerini İndir
#
# Bu script TARGET Linux sunucuda (Ubuntu/Debian) çalıştırılır.
# Tüm kritik paketleri ve bağımlılıklarını packages/ klasörüne indirir.
#
# Kullanım:
#   sudo bash packages/download.sh
#

set -euo pipefail

PACKAGES_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PACKAGES_DIR"

echo "📦 OpenSpeed Panel - Paket İndirici"
echo "===================================="
echo ""

# Geçici repo ekle (sadece indirmek için, sonra sileceğiz)
cleanup_repo() {
    rm -f /etc/apt/sources.list.d/lst_debian_repo.list 2>/dev/null
    apt-get update -qq 2>/dev/null || true
}
trap cleanup_repo EXIT

# === OpenLiteSpeed Repo ===
echo "🔧 OLS repo ekleniyor (geçici)..."
wget -q -O - http://rpms.litespeedtech.com/debian/enable_lst_debian_repo.sh | bash 2>/dev/null || true
apt-get update -qq 2>/dev/null || true

# === Kritik Paketler ===
PACKAGES=(
    # OpenLiteSpeed
    "openlitespeed"

    # PHP LSAPI sürümleri
    "lsphp82"
    "lsphp83"
    "lsphp84"
    "lsphp82-common" "lsphp82-mysql" "lsphp82-curl" "lsphp82-json" "lsphp82-mbstring"
    "lsphp83-common" "lsphp83-mysql" "lsphp83-curl" "lsphp83-json" "lsphp83-mbstring"
    "lsphp84-common" "lsphp84-mysql" "lsphp84-curl" "lsphp84-json" "lsphp84-mbstring"

    # PowerDNS + SQLite
    "pdns-server"
    "pdns-backend-sqlite3"

    # Email servisleri
    "postfix"
    "postfix-mysql"
    "dovecot-core"
    "dovecot-imapd"
    "dovecot-pop3d"
    "dovecot-mysql"
    "opendkim"
    "opendkim-tools"

    # Veritabanı
    "mariadb-server"
    "postgresql"

    # Cache
    "redis-server"

    # Güvenlik
    "fail2ban"
    "spamassassin"
    "spamc"

    # Araçlar
    "inotify-tools"
    "podman"
)

echo ""
echo "📥 Paketler indiriliyor..."
echo ""

for pkg in "${PACKAGES[@]}"; do
    echo -n "  $pkg ... "
    if apt download "$pkg" 2>/dev/null; then
        echo "✅"
    else
        echo "⚠️  atlandı (bulunamadı)"
    fi
done

# === Adminer ===
echo -n "  adminer ... "
curl -fsSL -o adminer.php "https://www.adminer.org/latest.php" 2>/dev/null && echo "✅" || echo "⚠️"

# === Özet ===
echo ""
echo "===================================="
echo "📊 İndirilen paketler:"
ls -lh "$PACKAGES_DIR"/*.deb 2>/dev/null | wc -l | xargs echo "  .deb dosyası:"
du -sh "$PACKAGES_DIR" 2>/dev/null | awk '{print "  Toplam: " $1}'
echo ""
echo "✅ Tamamlandı. Şimdi git commit && git push yapın."
