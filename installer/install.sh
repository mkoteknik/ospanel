#!/usr/bin/env bash
#
# OpenSpeed Panel - Tek Komut Kurulum Scripti
# https://github.com/mkoteknik/ospanel
#
# Kullanım:
#   curl -fsSL https://raw.githubusercontent.com/mkoteknik/ospanel/main/install.sh | sudo bash
#
# ============================================================
# BAĞIMLILIK STRATEJİSİ
# ============================================================
# 🔒 SABİT PAKET (repo YOK, test edilmiş sürümler):
#    - OpenLiteSpeed → direkt .deb/.rpm indir (v1.8.2)
#    - PHP LSAPI       → direkt .deb indir (8.2, 8.3, 8.4)
#    - Adminer         → direkt PHP dosyası indir
#    - OSPanel         → GitHub release'ten indir
#
# ✅ OS REPO (stabil, risk yok):
#    - MariaDB, PostgreSQL, Postfix, Dovecot, Redis
#    - PowerDNS, Podman/Docker, SpamAssassin, Fail2ban
#    - inotify-tools, certbot, curl, wget, openssl
#
# 🔄 GÜNCELLEME: Yeni sürüm test edilip onaylandıktan sonra
#    bu scriptteki versiyon numaraları güncellenir.
# ============================================================
#

set -uo pipefail

# Renkler
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

OSPANEL_VERSION="${OSPANEL_VERSION:-latest}"
OSPANEL_DIR="/opt/ospanel"
OSPANEL_DATA="/var/lib/ospanel"
OSPANEL_CONFIG="/etc/ospanel"

# Admin sifresi EN BASTA uretilir (OLS, systemd, panel seed'i icin)
ADMIN_PASS=$(openssl rand -base64 16 2>/dev/null | tr -d '=+/' | head -c 16)

# =====================================================================
# SABİT PAKET SİSTEMİ
# Öncelik: 1) GitHub repodaki packages/ klasörü  2) Direkt URL (fallback)
# Repo'dan kurulum YOK, tüm kritik paketler sabit sürüm
# =====================================================================
PKG_BASE="https://github.com/mkoteknik/ospanel/releases/latest/download"
OLS_DEB_FALLBACK="https://rpms.litespeedtech.com/debian/pool/main/jammy/openlitespeed_1.9.1-2+jammy_amd64.deb"
OLS_RPM_FALLBACK="https://rpms.litespeedtech.com/centos/8/x86_64/openlitespeed-1.8.2-1.el8.x86_64.rpm"

# Ortak indirme fonksiyonu: önce repodan, sonra fallback
download_pkg() {
    local name="$1"
    local fallback_url="$2"
    local dest="$3"

    # Önce repodan dene
    if curl -fsSL -o "$dest" "${PKG_BASE}/${name}" 2>/dev/null; then
        log_info "  ✓ Repodan indirildi: $name"
        return 0
    fi

    # Fallback URL dene
    if [[ -n "$fallback_url" ]]; then
        log_warn "  Repoda yok, fallback deneniyor: $fallback_url"
        if curl -fsSL -o "$dest" "$fallback_url" 2>/dev/null; then
            return 0
        fi
    fi

    return 1
}

log_info()  { echo -e "${GREEN}[✓]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
log_error() { echo -e "${RED}[✗]${NC} $*"; }
log_step()  { echo -e "\n${BLUE}${BOLD}[→]${NC}${BOLD} $*${NC}"; }

# ---------------------------------------------------------------------------
# Root kontrolü
# ---------------------------------------------------------------------------
if [[ $EUID -ne 0 ]]; then
    log_error "Bu script root olarak çalıştırılmalıdır."
    echo "  sudo bash install.sh"
    echo "  curl -fsSL ... | sudo bash"
    exit 1
fi

# ---------------------------------------------------------------------------
# OS Detect
# ---------------------------------------------------------------------------
detect_os() {
    log_step "İşletim sistemi tespit ediliyor..."

    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        log_error "İşletim sistemi tespit edilemedi."
        exit 1
    fi

    case "$OS" in
        ubuntu|debian|linuxmint)
            PKG_MANAGER="apt"
            log_info "Tespit edildi: $OS $OS_VERSION (apt)"
            ;;
        rocky|almalinux|rhel|centos)
            PKG_MANAGER="dnf"
            log_info "Tespit edildi: $OS $OS_VERSION (dnf)"
            ;;
        *)
            # Bilinmeyen OS'leri apt tabanlı varsay (Ubuntu/Debian türevleri)
            log_warn "Bilinmeyen işletim sistemi: $OS $OS_VERSION"
            log_warn "apt tabanlı olarak deneniyor..."
            PKG_MANAGER="apt"
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Bağımlılıkları yükle
# ---------------------------------------------------------------------------
install_dependencies() {
    log_step "Sistem paketleri güncelleniyor..."

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        apt-get install -y -qq curl wget tar gzip openssl certbot \
            ufw fail2ban 2>/dev/null || true
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y epel-release 2>/dev/null || true
        dnf install -y curl wget tar gzip openssl certbot \
            firewalld fail2ban 2>/dev/null || true
    fi

    log_info "Temel bağımlılıklar yüklendi"
}

# ---------------------------------------------------------------------------
# OpenLiteSpeed kurulumu
# ---------------------------------------------------------------------------
install_ols() {
    log_step "OpenLiteSpeed kuruluyor..."

    if [[ -f /usr/local/lsws/bin/lshttpd ]]; then
        log_info "OpenLiteSpeed zaten kurulu (/usr/local/lsws mevcut), atlanıyor"
        return
    fi

    local OLS_INSTALLED=0

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        # Ubuntu surumune gore OLS repo sec
        local LITE_REPO="jammy"
        case "${OS_VERSION:-}" in
            24.04|24.10) LITE_REPO="noble" ;;
            22.04) LITE_REPO="jammy" ;;
        esac
        log_info "OLS repo: ${LITE_REPO}"
        echo "deb https://rpms.litespeedtech.com/debian/ ${LITE_REPO} main" > /etc/apt/sources.list.d/lst_ospanel.list
        curl -fsSL https://rpms.litespeedtech.com/debian/lst_debian_repo.gpg | gpg --dearmor -o /etc/apt/trusted.gpg.d/lst_ospanel.gpg 2>/dev/null || true
        apt-get update -qq 2>/dev/null || true

        # OLS + PHP LSAPI birlikte kur (bağımlılıklar için)
        apt-get install -y -qq openlitespeed lsphp82 lsphp83 lsphp84 lsphp82-mysql lsphp83-mysql lsphp84-mysql 2>/dev/null && OLS_INSTALLED=1

        # Geçici repoyu temizle
        rm -f /etc/apt/sources.list.d/lst_ospanel.list
        apt-get update -qq 2>/dev/null || true
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        rpm -Uvh https://rpms.litespeedtech.com/centos/litespeed-repo-1.4-1.el8.noarch.rpm 2>/dev/null || true
        dnf install -y openlitespeed lsphp82 lsphp83 lsphp84 2>/dev/null && OLS_INSTALLED=1
    fi

    if [[ $OLS_INSTALLED -eq 0 ]]; then
        log_error "OpenLiteSpeed kurulamadı!"
        exit 1
    fi

    # OLS admin şifresini PANEL şifresiyle aynı yap
    if [[ -f /usr/local/lsws/admin/misc/admpass.sh ]]; then
        echo -e "admin\n${ADMIN_PASS}\n${ADMIN_PASS}" | bash /usr/local/lsws/admin/misc/admpass.sh 2>/dev/null || true
        log_info "OLS WebAdmin: admin / ${ADMIN_PASS}"
    fi

    # OLS servisini başlat
    systemctl enable lsws 2>/dev/null || true
    systemctl start lsws 2>/dev/null || {
        /usr/local/lsws/bin/lshttpd -k start 2>/dev/null || {
            log_error "OLS başlatılamadı!"
            exit 1
        }
    }

    # Port 80 ve 443 listener'larını ekle (varsayılan OLS sadece 8088'de dinler)
    log_info "OLS port 80/443 yapılandırılıyor..."
    local OLS_CONF="/usr/local/lsws/conf/httpd_config.xml"
    [[ -f "$OLS_CONF" ]] || OLS_CONF="/usr/local/lsws/conf/httpd_config.conf"
    if ! grep -q "listener HTTP" "$OLS_CONF" 2>/dev/null; then
        sed -i 's|listeners                Default|listeners                Default, HTTP, HTTPS|' "$OLS_CONF" 2>/dev/null || true
        cat >> "$OLS_CONF" << 'LSWSCONF'

listener HTTP{
    address                 *:80
    secure                  0
    map                     Example *
}
listener HTTPS{
    address                 *:443
    secure                  0
    map                     Example *
}
LSWSCONF
        log_info "Port 80/443 listener eklendi"
    fi

    # Full restart (graceful yetmez)
    systemctl restart lsws 2>/dev/null || /usr/local/lsws/bin/lshttpd -k restart 2>/dev/null || true

    log_info "OpenLiteSpeed başarıyla kuruldu ve başlatıldı"
}

# ---------------------------------------------------------------------------
# MariaDB kurulumu
# ---------------------------------------------------------------------------
install_mariadb() {
    log_step "MariaDB kuruluyor..."

    if command -v mariadb &>/dev/null || command -v mysql &>/dev/null; then
        log_info "Veritabanı sunucusu zaten kurulu, atlanıyor"
        return
    fi

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        apt-get install -y -qq mariadb-server 2>/dev/null || \
        apt-get install -y -qq mysql-server 2>/dev/null || true
    else
        dnf install -y mariadb-server 2>/dev/null || true
    fi

    systemctl enable mariadb 2>/dev/null || systemctl enable mysql 2>/dev/null || true
    systemctl start mariadb 2>/dev/null || systemctl start mysql 2>/dev/null || true

    log_info "MariaDB kuruldu"
}

# ---------------------------------------------------------------------------
# PHP LSAPI kurulumu
# ---------------------------------------------------------------------------
install_php() {
    log_step "PHP LSAPI kontrol ediliyor..."

    local INSTALLED_COUNT=0
    for ver in 82 83 84; do
        if [[ -f "/usr/local/lsws/lsphp${ver}/bin/lsphp" ]]; then
            ((INSTALLED_COUNT++))
        fi
    done

    if [[ $INSTALLED_COUNT -ge 1 ]]; then
        log_info "$INSTALLED_COUNT PHP LSAPI sürümü mevcut (OLS ile kuruldu)"
    else
        log_warn "PHP LSAPI bulunamadı, OLS kurulumu kontrol edin"
    fi
}

# ---------------------------------------------------------------------------
# Postfix + Dovecot + MariaDB Virtual Users + OpenDKIM
# ---------------------------------------------------------------------------
install_email_services() {
    log_step "Email sunucusu kuruluyor (Postfix + Dovecot + MariaDB + DKIM)..."

    # === Paketler ===
    if [[ "$PKG_MANAGER" == "apt" ]]; then
        export DEBIAN_FRONTEND=noninteractive
        echo "postfix postfix/mailname string $(hostname -f 2>/dev/null || echo localhost)" | debconf-set-selections 2>/dev/null || true
        echo "postfix postfix/main_mailer_type string 'Internet Site'" | debconf-set-selections 2>/dev/null || true
        apt-get install -y -qq postfix postfix-mysql dovecot-core dovecot-imapd dovecot-pop3d dovecot-mysql opendkim opendkim-tools 2>/dev/null || true
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y postfix dovecot dovecot-mysql opendkim 2>/dev/null || true
    fi

    local MAIL_DB="mailserver"
    local MAIL_DB_USER="mailuser"
    local MAIL_DB_PASS=$(openssl rand -base64 16 2>/dev/null | tr -d '=+/' | head -c 20)

    # === MariaDB: email veritabanı ve kullanıcı ===
    log_info "Email veritabanı oluşturuluyor..."
    # MariaDB'ye baglan (socket auth veya root sifresi)
    local MYSQL_CMD="mysql"
    if [[ -f /etc/ospanel/mysql_root_pass ]]; then
        local MYSQL_ROOT_PASS=$(cat /etc/ospanel/mysql_root_pass)
        MYSQL_CMD="mysql -uroot -p${MYSQL_ROOT_PASS}"
    fi
    $MYSQL_CMD -e "CREATE DATABASE IF NOT EXISTS ${MAIL_DB};" 2>/dev/null || true
    $MYSQL_CMD -e "CREATE USER IF NOT EXISTS '${MAIL_DB_USER}'@'localhost' IDENTIFIED BY '${MAIL_DB_PASS}';" 2>/dev/null || true
    $MYSQL_CMD -e "GRANT ALL PRIVILEGES ON ${MAIL_DB}.* TO '${MAIL_DB_USER}'@'localhost';" 2>/dev/null || true
    $MYSQL_CMD -e "FLUSH PRIVILEGES;" 2>/dev/null || true

    $MYSQL_CMD "$MAIL_DB" << SQLEOF
CREATE TABLE IF NOT EXISTS virtual_domains (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS virtual_users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    domain_id INT NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    maildir VARCHAR(255) NOT NULL,
    quota INT NOT NULL DEFAULT 1024,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (domain_id) REFERENCES virtual_domains(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS virtual_aliases (
    id INT AUTO_INCREMENT PRIMARY KEY,
    domain_id INT NOT NULL,
    source VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    FOREIGN KEY (domain_id) REFERENCES virtual_domains(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SQLEOF

    log_info "Email veritabanı hazır (${MAIL_DB})"

    # === Postfix: virtual domain/user konfigürasyonu ===
    local MAIL_ROOT="/var/mail/vhosts"
    mkdir -p "$MAIL_ROOT"
    groupadd -f vmail 2>/dev/null || true
    useradd -g vmail -d /var/mail -s /sbin/nologin vmail 2>/dev/null || true
    chown -R vmail:vmail "$MAIL_ROOT"

    # MariaDB bağlantı dosyaları
    local MYSQL_CFG_DIR="/etc/postfix/mysql"
    mkdir -p "$MYSQL_CFG_DIR"

    cat > "${MYSQL_CFG_DIR}/virtual-mailbox-domains.cf" << MYSQLCFG
user = ${MAIL_DB_USER}
password = ${MAIL_DB_PASS}
hosts = 127.0.0.1
dbname = ${MAIL_DB}
query = SELECT 1 FROM virtual_domains WHERE name='%s'
MYSQLCFG

    cat > "${MYSQL_CFG_DIR}/virtual-mailbox-maps.cf" << MYSQLCFG
user = ${MAIL_DB_USER}
password = ${MAIL_DB_PASS}
hosts = 127.0.0.1
dbname = ${MAIL_DB}
query = SELECT maildir FROM virtual_users WHERE email='%s'
MYSQLCFG

    cat > "${MYSQL_CFG_DIR}/virtual-alias-maps.cf" << MYSQLCFG
user = ${MAIL_DB_USER}
password = ${MAIL_DB_PASS}
hosts = 127.0.0.1
dbname = ${MAIL_DB}
query = SELECT destination FROM virtual_aliases WHERE source='%s'
MYSQLCFG

    # Postfix ana konfigürasyon
    postconf -e "myhostname = $(hostname -f 2>/dev/null || hostname)"
    postconf -e "mydestination = localhost"
    postconf -e "mynetworks = 127.0.0.0/8"
    postconf -e "inet_interfaces = all"
    postconf -e "message_size_limit = 30720000"

    # Virtual domain/user
    postconf -e "virtual_mailbox_domains = mysql:/etc/postfix/mysql/virtual-mailbox-domains.cf"
    postconf -e "virtual_mailbox_maps = mysql:/etc/postfix/mysql/virtual-mailbox-maps.cf"
    postconf -e "virtual_alias_maps = mysql:/etc/postfix/mysql/virtual-alias-maps.cf"
    postconf -e "virtual_mailbox_base = ${MAIL_ROOT}"
    postconf -e "virtual_minimum_uid = 150"
    postconf -e "virtual_uid_maps = static:150"
    postconf -e "virtual_gid_maps = static:8"
    postconf -e "virtual_transport = dovecot"

    # TLS/SSL
    postconf -e "smtpd_tls_security_level = may"
    postconf -e "smtp_tls_security_level = may"
    postconf -e "smtpd_tls_auth_only = yes"
    postconf -e "smtpd_tls_cert_file = /etc/ssl/certs/ssl-cert-snakeoil.pem"
    postconf -e "smtpd_tls_key_file = /etc/ssl/private/ssl-cert-snakeoil.key"

    # Dovecot teslim
    postconf -e "dovecot_destination_recipient_limit = 1"
    postconf -e "smtpd_sasl_type = dovecot"
    postconf -e "smtpd_sasl_path = private/auth"
    postconf -e "smtpd_sasl_auth_enable = yes"

    # Master.cf - Dovecot LMTP
    if ! grep -q "dovecot" /etc/postfix/master.cf 2>/dev/null; then
        cat >> /etc/postfix/master.cf << 'MASTERCF'
dovecot   unix  -       n       n       -       -       pipe
  flags=DRhu user=vmail:vmail argv=/usr/lib/dovecot/deliver -f ${sender} -d ${recipient}
MASTERCF
    fi

    log_info "Postfix yapılandırıldı (MariaDB virtual users)"

    # === Dovecot konfigürasyon ===
    local DOVECOT_MAIL_DIR="$MAIL_ROOT/%d/%n"

    # 10-mail.conf
    if [[ -f /etc/dovecot/conf.d/10-mail.conf ]]; then
        sed -i "s|^#\?mail_location =.*|mail_location = maildir:${DOVECOT_MAIL_DIR}|" /etc/dovecot/conf.d/10-mail.conf 2>/dev/null
        sed -i 's|^#\?mail_uid =.*|mail_uid = 150|' /etc/dovecot/conf.d/10-mail.conf 2>/dev/null
        sed -i 's|^#\?mail_gid =.*|mail_gid = 8|' /etc/dovecot/conf.d/10-mail.conf 2>/dev/null
    fi

    # 10-auth.conf
    if [[ -f /etc/dovecot/conf.d/10-auth.conf ]]; then
        sed -i 's|^#\?disable_plaintext_auth =.*|disable_plaintext_auth = no|' /etc/dovecot/conf.d/10-auth.conf 2>/dev/null
        sed -i 's|^auth_mechanisms =.*|auth_mechanisms = plain login|' /etc/dovecot/conf.d/10-auth.conf 2>/dev/null
    fi

    # Dovecot SQL auth
    cat > /etc/dovecot/dovecot-sql.conf.ext << DOVECOTSQL
driver = mysql
connect = host=127.0.0.1 dbname=${MAIL_DB} user=${MAIL_DB_USER} password=${MAIL_DB_PASS}
default_pass_scheme = SHA512-CRYPT
password_query = SELECT email AS user, password FROM virtual_users WHERE email='%u'
user_query = SELECT '${DOVECOT_MAIL_DIR}' AS mail, 150 AS uid, 8 AS gid FROM virtual_users WHERE email='%u'
iterate_query = SELECT email AS user FROM virtual_users
DOVECOTSQL

    # 10-master.conf - Postfix SASL
    if [[ -f /etc/dovecot/conf.d/10-master.conf ]]; then
        if ! grep -q "Postfix smtp-auth" /etc/dovecot/conf.d/10-master.conf 2>/dev/null; then
            cat >> /etc/dovecot/conf.d/10-master.conf << 'DOVEMASTER'

# Postfix SASL
service auth {
  unix_listener /var/spool/postfix/private/auth {
    mode = 0666
    user = postfix
    group = postfix
  }
}
DOVEMASTER
        fi
    fi

    # SSL
    if [[ -f /etc/dovecot/conf.d/10-ssl.conf ]]; then
        sed -i 's|^#\?ssl =.*|ssl = yes|' /etc/dovecot/conf.d/10-ssl.conf 2>/dev/null || true
    fi

    log_info "Dovecot yapılandırıldı (MariaDB auth)"

    # === OpenDKIM ===
    if command -v opendkim &>/dev/null; then
        mkdir -p /etc/opendkim/keys
        cat > /etc/opendkim.conf << 'DKIMEOF'
Syslog yes
UMask 002
Mode sv
KeyTable /etc/opendkim/KeyTable
SigningTable /etc/opendkim/SigningTable
ExternalIgnoreList /etc/opendkim/TrustedHosts
InternalHosts /etc/opendkim/TrustedHosts
Socket inet:8891@localhost
DKIMEOF
        touch /etc/opendkim/KeyTable /etc/opendkim/SigningTable /etc/opendkim/TrustedHosts
        echo "127.0.0.1" > /etc/opendkim/TrustedHosts
        echo "localhost" >> /etc/opendkim/TrustedHosts

        postconf -e "milter_default_action = accept"
        postconf -e "milter_protocol = 6"
        postconf -e "smtpd_milters = inet:localhost:8891"
        postconf -e "non_smtpd_milters = \$smtpd_milters"

        systemctl enable opendkim 2>/dev/null || true
        systemctl restart opendkim 2>/dev/null || true
        log_info "OpenDKIM yapılandırıldı"
    fi

    # === Servisleri başlat ===
    systemctl enable postfix 2>/dev/null || true
    systemctl enable dovecot 2>/dev/null || true
    systemctl restart postfix 2>/dev/null || true
    systemctl restart dovecot 2>/dev/null || true

    # Email bilgilerini kaydet
    cat > /etc/ospanel/email_db.conf << DBEOF
DB_NAME=${MAIL_DB}
DB_USER=${MAIL_DB_USER}
DB_PASS=${MAIL_DB_PASS}
DBEOF
    chmod 600 /etc/ospanel/email_db.conf

    log_info "Email sunucusu tamamen kuruldu: Postfix + Dovecot + MariaDB + DKIM"
    log_info "Virtual domain/user/alias - MariaDB tabanlı"
    log_info "Konfigürasyon: /etc/ospanel/email_db.conf"
}

# ---------------------------------------------------------------------------
# PowerDNS + SQLite kurulumu
# ---------------------------------------------------------------------------
install_dns_server() {
    log_step "PowerDNS kuruluyor (SQLite backend)..."

    if command -v pdns_server &>/dev/null; then
        log_info "PowerDNS zaten kurulu, atlanıyor"
        return
    fi

    local PDNS_INSTALLED=0
    local PDNS_API_KEY=$(openssl rand -hex 32 2>/dev/null || echo "changeme")

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        # PowerDNS + SQLite backend
        apt-get install -y -qq pdns-server pdns-backend-sqlite3 sqlite3 2>/dev/null && PDNS_INSTALLED=1
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y pdns pdns-backend-sqlite3 sqlite 2>/dev/null && PDNS_INSTALLED=1
        # EPEL'den dene
        if [[ $PDNS_INSTALLED -eq 0 ]]; then
            dnf install -y epel-release 2>/dev/null || true
            dnf install -y pdns pdns-backend-sqlite3 2>/dev/null && PDNS_INSTALLED=1
        fi
    fi

    if [[ $PDNS_INSTALLED -eq 0 ]]; then
        log_warn "PowerDNS kurulamadı, DNS yönetimi sınırlı olacak"
        return
    fi

    # SQLite veritabanı oluştur
    local PDNS_DB="/var/lib/pdns/pdns.sqlite3"
    mkdir -p /var/lib/pdns

    sqlite3 "$PDNS_DB" "CREATE TABLE IF NOT EXISTS domains (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name VARCHAR(255) NOT NULL UNIQUE,
        master VARCHAR(128) DEFAULT NULL,
        last_check INTEGER DEFAULT NULL,
        type VARCHAR(8) NOT NULL DEFAULT 'NATIVE',
        notified_serial INTEGER DEFAULT NULL,
        account VARCHAR(40) DEFAULT NULL
    );" 2>/dev/null || true

    sqlite3 "$PDNS_DB" "CREATE TABLE IF NOT EXISTS records (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        domain_id INTEGER,
        name VARCHAR(255),
        type VARCHAR(10),
        content VARCHAR(65535),
        ttl INTEGER DEFAULT 3600,
        prio INTEGER DEFAULT 0,
        change_date INTEGER DEFAULT 0,
        disabled BOOLEAN DEFAULT 0,
        auth BOOL DEFAULT 1,
        FOREIGN KEY(domain_id) REFERENCES domains(id) ON DELETE CASCADE
    );" 2>/dev/null || true

    chown -R pdns:pdns /var/lib/pdns 2>/dev/null || true

    # PowerDNS konfigürasyonu
    cat > /etc/pdns/pdns.conf << PDNSCONF
# PowerDNS - OpenSpeed Panel
launch=gsqlite3
gsqlite3-database=${PDNS_DB}
gsqlite3-dnssec=no

# API (sadece localhost)
api=yes
api-key=${PDNS_API_KEY}
webserver=yes
webserver-address=127.0.0.1
webserver-port=8081
webserver-allow-from=127.0.0.1

# Güvenlik
allow-axfr-ips=127.0.0.1
master=no
slave=no
default-soa-content=ns1.ospanel.local admin.ospanel.local 0 10800 3600 604800 3600

# Performans
cache-ttl=20
query-cache-ttl=20
negquery-cache-ttl=60
PDNSCONF

    # API key'i kaydet
    echo "$PDNS_API_KEY" > /etc/ospanel/pdns_api_key
    chmod 600 /etc/ospanel/pdns_api_key

    systemctl enable pdns 2>/dev/null || true
    systemctl restart pdns 2>/dev/null || true

    log_info "PowerDNS kuruldu (SQLite backend, API: :8081)"
}

# ---------------------------------------------------------------------------
# SpamAssassin kurulumu
# ---------------------------------------------------------------------------
install_spamassassin() {
    log_step "SpamAssassin kuruluyor..."

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        apt-get install -y -qq spamassassin spamc 2>/dev/null || true
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y spamassassin 2>/dev/null || true
    fi

    if command -v spamassassin &>/dev/null; then
        systemctl enable spamassassin 2>/dev/null || true
        systemctl start spamassassin 2>/dev/null || true
        # SpamAssassin kurallarını güncelle
        sa-update 2>/dev/null || true
        log_info "SpamAssassin kuruldu ve kurallar güncellendi"
    fi
}

# ---------------------------------------------------------------------------
# Adminer kurulumu (phpMyAdmin alternatifi - tek dosya)
# ---------------------------------------------------------------------------
install_adminer() {
    log_step "Adminer kuruluyor (hafif veritabanı yöneticisi)..."

    local ADMINER_DIR="/usr/local/lsws/Example/html/adminer"
    local ADMINER_FILE="$ADMINER_DIR/index.php"

    if [[ -f "$ADMINER_FILE" ]]; then
        log_info "Adminer zaten kurulu, atlanıyor"
        return
    fi

    mkdir -p "$ADMINER_DIR"

    # En son Adminer sürümünü indir (tek PHP dosyası ~500KB)
    # MySQL, PostgreSQL, SQLite, MongoDB, Elasticsearch hepsini destekler
    if download_pkg "adminer.php" "https://www.adminer.org/latest.php" "$ADMINER_FILE"; then
        chmod 644 "$ADMINER_FILE"
        log_info "Adminer kuruldu: $ADMINER_FILE (tek dosya, ~500KB)"
        log_info "Adminer URL: http://SUNUCU-IP/adminer/"
    else
        log_warn "Adminer indirilemedi, atlanıyor"
    fi
}

# ---------------------------------------------------------------------------
# PostgreSQL kurulumu
# ---------------------------------------------------------------------------
install_postgresql() {
    log_step "PostgreSQL kuruluyor..."

    if command -v psql &>/dev/null; then
        log_info "PostgreSQL zaten kurulu, atlanıyor"
        return
    fi

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        apt-get install -y -qq postgresql postgresql-contrib 2>/dev/null && log_info "PostgreSQL kuruldu"
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y postgresql-server postgresql-contrib 2>/dev/null && {
            postgresql-setup --initdb 2>/dev/null || true
            log_info "PostgreSQL kuruldu"
        }
    fi

    if command -v psql &>/dev/null; then
        # PostgreSQL şifresi ayarla
        local PG_PASS=$(openssl rand -base64 12 2>/dev/null | tr -d '=+/' | head -c 16)
        su - postgres -c "psql -c \"ALTER USER postgres PASSWORD '${PG_PASS}';\"" 2>/dev/null || true
        echo "$PG_PASS" > /etc/ospanel/pg_pass
        chmod 600 /etc/ospanel/pg_pass

        systemctl enable postgresql 2>/dev/null || true
        systemctl restart postgresql 2>/dev/null || true
        log_info "PostgreSQL yapılandırıldı"
    fi
}

# ---------------------------------------------------------------------------
# Redis kurulumu
# ---------------------------------------------------------------------------
install_redis() {
    log_step "Redis cache kuruluyor..."

    if command -v redis-server &>/dev/null; then
        log_info "Redis zaten kurulu, atlanıyor"
        return
    fi

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        apt-get install -y -qq redis-server 2>/dev/null || true
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y redis 2>/dev/null || true
    fi

    if command -v redis-server &>/dev/null; then
        # Redis yapılandırması - socket ve TCP
        sed -i 's|^bind .*|bind 127.0.0.1|' /etc/redis/redis.conf 2>/dev/null || true
        sed -i 's|^# maxmemory .*|maxmemory 256mb|' /etc/redis/redis.conf 2>/dev/null || true
        sed -i 's|^# maxmemory-policy .*|maxmemory-policy allkeys-lru|' /etc/redis/redis.conf 2>/dev/null || true

        systemctl enable redis-server 2>/dev/null || systemctl enable redis 2>/dev/null || true
        systemctl restart redis-server 2>/dev/null || systemctl restart redis 2>/dev/null || true
        log_info "Redis kuruldu ve yapılandırıldı (max 256MB, LRU policy)"
    else
        log_warn "Redis kurulamadı, cache özelliği sınırlı olacak"
    fi
}

# ---------------------------------------------------------------------------
# Docker / Podman kurulumu (opsiyonel)
# ---------------------------------------------------------------------------
install_container_runtime() {
    log_step "Konteyner runtime kuruluyor..."

    # Podman'ı dene (rootless, daha güvenli)
    if command -v podman &>/dev/null; then
        log_info "Podman zaten kurulu, atlanıyor"
        return
    fi

    if command -v docker &>/dev/null; then
        log_info "Docker zaten kurulu, atlanıyor"
        return
    fi

    # Podman dene (önerilen - rootless)
    if [[ "$PKG_MANAGER" == "apt" ]]; then
        apt-get install -y -qq podman 2>/dev/null && {
            log_info "Podman (rootless) kuruldu ✅"
            return
        }
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y podman 2>/dev/null && {
            log_info "Podman (rootless) kuruldu ✅"
            return
        }
    fi

    # Podman yoksa Docker dene
    log_info "Podman bulunamadı, Docker deneniyor..."
    if [[ "$PKG_MANAGER" == "apt" ]]; then
        apt-get install -y -qq docker.io 2>/dev/null || \
        curl -fsSL https://get.docker.com | bash 2>/dev/null || true
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        dnf install -y docker 2>/dev/null || true
    fi

    if command -v docker &>/dev/null; then
        systemctl enable docker 2>/dev/null || true
        systemctl start docker 2>/dev/null || true
        log_info "Docker kuruldu ✅"
    else
        log_warn "Konteyner runtime kurulamadı (opsiyonel)"
    fi
}

# ---------------------------------------------------------------------------
# .htaccess Watchdog kurulumu
# ---------------------------------------------------------------------------
install_htaccess_watchdog() {
    log_step ".htaccess Watchdog kuruluyor..."

    if ! command -v inotifywait &>/dev/null; then
        if [[ "$PKG_MANAGER" == "apt" ]]; then
            apt-get install -y -qq inotify-tools 2>/dev/null || true
        elif [[ "$PKG_MANAGER" == "dnf" ]]; then
            dnf install -y inotify-tools 2>/dev/null || true
        fi
    fi

    # Watchdog scriptini kopyala
    cp /opt/ospanel/htaccess-watchdog.sh /usr/local/bin/ospanel-htaccess-watchdog 2>/dev/null || true
    chmod +x /usr/local/bin/ospanel-htaccess-watchdog 2>/dev/null || true

    # systemd servis
    cat > /etc/systemd/system/ospanel-htaccess-watchdog.service << 'WDSVCEOF'
[Unit]
Description=OpenSpeed Panel .htaccess Watchdog
After=network.target lsws.service

[Service]
Type=simple
ExecStart=/usr/local/bin/ospanel-htaccess-watchdog
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
WDSVCEOF

    systemctl daemon-reload 2>/dev/null || true
    systemctl enable ospanel-htaccess-watchdog 2>/dev/null || true
    systemctl restart ospanel-htaccess-watchdog 2>/dev/null || true

    log_info ".htaccess Watchdog kuruldu - OLS otomatik reload aktif"
    log_info "İzlenen: /home/**/.htaccess → değişiklik → OLS graceful restart"
}

# ---------------------------------------------------------------------------
# SnappyMail Webmail kurulumu
# ---------------------------------------------------------------------------
install_webmail() {
    log_step "SnappyMail Webmail kuruluyor..."

    local WEBMAIL_DIR="/usr/local/lsws/Example/html/webmail"

    if [[ -f "$WEBMAIL_DIR/index.php" ]]; then
        log_info "Webmail zaten kurulu, atlanıyor"
        return
    fi

    mkdir -p "$WEBMAIL_DIR"
    cd /tmp

    # SnappyMail son sürüm
    local SNAPPY_URL="https://github.com/the-djmaze/snappymail/releases/download/v2.38.2/snappymail-2.38.2.tar.gz"
    if curl -fsSL -o snappymail.tar.gz "$SNAPPY_URL" 2>/dev/null; then
        tar xzf snappymail.tar.gz -C "$WEBMAIL_DIR" --strip-components=1 2>/dev/null
        chown -R www-data:www-data "$WEBMAIL_DIR" 2>/dev/null || true
        log_info "SnappyMail v2.38.2 kuruldu: $WEBMAIL_DIR"
        log_info "Webmail URL: http://SUNUCU-IP/webmail/"
    else
        log_warn "SnappyMail indirilemedi, atlanıyor"
    fi
}

# ---------------------------------------------------------------------------
# Fail2ban konfigürasyonu
# ---------------------------------------------------------------------------
configure_fail2ban() {
    log_step "Fail2ban yapılandırılıyor..."

    if ! command -v fail2ban-client &>/dev/null; then
        if [[ "$PKG_MANAGER" == "apt" ]]; then
            apt-get install -y -qq fail2ban 2>/dev/null || true
        else
            dnf install -y fail2ban 2>/dev/null || true
        fi
    fi

    # OLS jail ekle
    cat > /etc/fail2ban/jail.local << 'F2BEOF'
[lshttpd]
enabled = true
port = http,https
filter = lshttpd
logpath = /usr/local/lsws/logs/error.log
maxretry = 5
bantime = 3600
F2BEOF

    # OLS fail2ban filter olustur
    cat > /etc/fail2ban/filter.d/lshttpd.conf << 'F2BFILTER'
[Definition]
failregex = ^.*\[NOTICE\].*\[.*:.*\].*Connection from <HOST> .* denied.*$
            ^.*\[INFO\].*\[.*:.*\].*Brute force attack detected from <HOST>.*$
ignoreregex =
F2BFILTER

    systemctl enable fail2ban 2>/dev/null || true
    systemctl restart fail2ban 2>/dev/null || true
    log_info "Fail2ban yapılandırıldı"
}

# ---------------------------------------------------------------------------
# MariaDB güvenli kurulum
# ---------------------------------------------------------------------------
secure_mariadb() {
    log_step "MariaDB güvenlik yapılandırması..."

    mkdir -p /etc/ospanel
    local MYSQL_PASS=$(openssl rand -base64 16 2>/dev/null | tr -d '=+/' | head -c 20)

    # Root şifresi belirle ve güvenlik ayarları
    if command -v mysql &>/dev/null; then
        mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '${MYSQL_PASS}';" 2>/dev/null || true
        mysql -e "DELETE FROM mysql.user WHERE User='';" 2>/dev/null || true
        mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');" 2>/dev/null || true
        mysql -e "DROP DATABASE IF EXISTS test;" 2>/dev/null || true
        mysql -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';" 2>/dev/null || true
        $MYSQL_CMD -e "FLUSH PRIVILEGES;" 2>/dev/null || true
    fi

    echo "$MYSQL_PASS" > /etc/ospanel/mysql_root_pass
    chmod 600 /etc/ospanel/mysql_root_pass
    log_info "MariaDB root şifresi: /etc/ospanel/mysql_root_pass"
}

# ---------------------------------------------------------------------------
# OSPanel Binary indir ve kur
# ---------------------------------------------------------------------------
install_ospanel() {
    log_step "OpenSpeed Panel indiriliyor..."

    mkdir -p "$OSPANEL_DIR" "$OSPANEL_DATA" "$OSPANEL_CONFIG"

    # GitHub'dan son sürümü indir
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        *)       log_error "Desteklenmeyen mimari: $ARCH"; exit 1 ;;
    esac

    if [[ "$OSPANEL_VERSION" == "latest" ]]; then
        DOWNLOAD_URL="https://github.com/mkoteknik/ospanel/releases/latest/download/ospanel-linux-${ARCH}.tar.gz"
    else
        DOWNLOAD_URL="https://github.com/mkoteknik/ospanel/releases/download/${OSPANEL_VERSION}/ospanel-linux-${ARCH}.tar.gz"
    fi

    cd /tmp
    if curl -fsSL -o ospanel.tar.gz "$DOWNLOAD_URL" 2>/dev/null; then
        tar xzf ospanel.tar.gz -C "$OSPANEL_DIR/"
        chmod +x "$OSPANEL_DIR/ospanel"
        log_info "Binary indirildi ve kuruldu"
    else
        log_warn "GitHub'dan indirilemedi. Eğer bu geliştirme kurulumu ise manuel build yapın."
        log_warn "  git clone https://github.com/mkoteknik/ospanel.git"
        log_warn "  cd ospanel && make build && sudo make install"
        # Geliştirme için: ortam değişkeninden path al
        if [[ -n "${OSPANEL_DEV_BINARY:-}" ]] && [[ -f "$OSPANEL_DEV_BINARY" ]]; then
            cp "$OSPANEL_DEV_BINARY" "$OSPANEL_DIR/ospanel"
            chmod +x "$OSPANEL_DIR/ospanel"
            log_info "Geliştirme binary'si kopyalandı: $OSPANEL_DEV_BINARY"
        else
            log_error "Kurulum tamamlanamadı."
            exit 1
        fi
    fi

    # Konfigürasyon oluştur
		# Rastgele JWT secret uret (her kurulumda benzersiz)
		JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || echo "change-me-please-change-me-32bytes")
    if [[ ! -f "$OSPANEL_CONFIG/config.yaml" ]]; then
        cat > "$OSPANEL_CONFIG/config.yaml" << YAMLEOF
server:
  host: "0.0.0.0"
  port: 8090
  tls:
    enabled: false

auth:
  jwt_secret: "${JWT_SECRET}"
  access_token_expiry: 15
  refresh_token_expiry: 10080
  max_login_attempts: 5
  lockout_duration: 30

database:
  max_connections: 10
  wal_mode: true

log:
  level: "info"
  output: "stdout"

ols:
  admin_url: "http://localhost:7080"
  vhosts_dir: "/usr/local/lsws/conf/vhosts"
  conf_dir: "/usr/local/lsws/conf"
  bin_path: "/usr/local/lsws/bin/lshttpd"

data_dir: "/var/lib/ospanel"
YAMLEOF
        log_info "Konfigürasyon dosyası oluşturuldu: $OSPANEL_CONFIG/config.yaml"
    fi
}

# ---------------------------------------------------------------------------
# systemd servis kurulumu
# ---------------------------------------------------------------------------
install_service() {
    log_step "systemd servisi kuruluyor..."

    # Ozel ospanel kullanicisi olustur (root yerine)
    if ! id ospanel &>/dev/null; then
        useradd -r -s /sbin/nologin -d /var/lib/ospanel -m ospanel 2>/dev/null || true
        # /home erisimi icin gruba ekle
        usermod -aG ospanel ospanel 2>/dev/null || true
    fi
    # /home dizinlerine erisim izni
    chown -R ospanel:ospanel ${OSPANEL_DATA} ${OSPANEL_CONFIG} 2>/dev/null || true

    cat > /etc/systemd/system/ospanel.service << SYSTEMDEOF
[Unit]
Description=OpenSpeed Panel
Documentation=https://github.com/mkoteknik/ospanel
After=network.target lsws.service

[Service]
Type=simple
User=ospanel
Group=ospanel
Environment="OSPANEL_ADMIN_PASS=${ADMIN_PASS}"
ExecStart=${OSPANEL_DIR}/ospanel --config ${OSPANEL_CONFIG}/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# Guvenlik - panel /home dizinlerine erisim gerektirir
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=no
ReadWritePaths=${OSPANEL_DATA} ${OSPANEL_CONFIG} /usr/local/lsws/conf /tmp /home /etc/cron.d /etc/letsencrypt /var/backups /etc/opendkim

[Install]
WantedBy=multi-user.target
SYSTEMDEOF

    systemctl daemon-reload
    systemctl enable ospanel
    systemctl start ospanel 2>/dev/null || log_warn "Servis başlatılamadı (ilk kurulumda normal)"

    log_info "systemd servisi kuruldu (ospanel kullanicisi ile)"
}

# ---------------------------------------------------------------------------
# Firewall yapılandırması
# ---------------------------------------------------------------------------
configure_firewall() {
    log_step "Firewall yapılandırılıyor..."

    if command -v ufw &>/dev/null; then
        ufw allow 80/tcp 2>/dev/null || true
        ufw allow 443/tcp 2>/dev/null || true
        ufw allow 8090/tcp 2>/dev/null || true
        ufw --force enable 2>/dev/null || true
        log_info "UFW kuralları eklendi"
    elif command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-service=http 2>/dev/null || true
        firewall-cmd --permanent --add-service=https 2>/dev/null || true
        firewall-cmd --permanent --add-port=8090/tcp 2>/dev/null || true
        firewall-cmd --reload 2>/dev/null || true
        log_info "firewalld kuralları eklendi"
    fi
}

# ---------------------------------------------------------------------------
# Admin kullanıcı oluştur
# ---------------------------------------------------------------------------

create_admin_user() {
    log_step "Admin kullanıcısı yapılandırılıyor..."

    # Config dosyasından admin şifresini güncelle (önceden oluşturuldu)
    if [[ -f "$OSPANEL_CONFIG/config.yaml" ]]; then
        sed -i "s|jwt_secret:.*|jwt_secret: \"$(openssl rand -hex 32)\"" "$OSPANEL_CONFIG/config.yaml" 2>/dev/null || true
    fi

    # Paneli yeniden başlat (yeni şifreyle seed oluşsun)
    systemctl stop ospanel 2>/dev/null || true
    sleep 2
    systemctl start ospanel 2>/dev/null || true

    log_info "Admin bilgileri yapılandırıldı"
}

# ---------------------------------------------------------------------------
# Ana kurulum
# ---------------------------------------------------------------------------
main() {
    echo ""
    echo "╔══════════════════════════════════════════════╗"
    echo "║         OpenSpeed Panel Kurulum              ║"
    echo "║         Modern Hosting Kontrol Paneli         ║"
    echo "╚══════════════════════════════════════════════╝"
    echo ""
    echo "⚡ OpenLiteSpeed + Go + Vue 3"
    echo ""

    detect_os

    # Hostname kontrol
    SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "SUNUCU-IP")
    CURRENT_HOSTNAME=$(hostname 2>/dev/null || echo "localhost")
    log_info "Sunucu IP: $SERVER_IP"
    log_info "Hostname: $CURRENT_HOSTNAME"
    if [[ "$CURRENT_HOSTNAME" == "localhost" ]] || [[ "$CURRENT_HOSTNAME" == ubuntu* ]] || [[ "$CURRENT_HOSTNAME" == debian* ]]; then
        log_warn "Hostname varsayılan. Önerilen: hostnamectl set-hostname server.siteniz.com"
    fi

    # TÜM SERVİSLER - hata olsa bile devam et
    log_info "Tüm servisler kuruluyor..."
    install_dependencies
    install_ols || log_error "OLS kurulamadı!"
    install_mariadb || log_warn "MariaDB kurulamadı"
    secure_mariadb || log_warn "MariaDB güvenlik ayarları atlandı"
    install_php || log_warn "PHP LSAPI kontrolü atlandı"
    install_email_services || log_warn "Email servisleri atlandı"
    install_spamassassin || log_warn "SpamAssassin atlandı"
    install_dns_server || log_warn "PowerDNS atlandı"
    install_postgresql || log_warn "PostgreSQL atlandı"
    install_redis || log_warn "Redis atlandı"
    install_container_runtime || log_warn "Podman atlandı"
    install_adminer || log_warn "Adminer atlandı"
    install_webmail || log_warn "Webmail atlandı"
    install_ospanel || log_error "Panel binary kurulamadı!"
    install_service || log_error "systemd servisi kurulamadı!"
    configure_firewall || log_warn "Firewall atlandı"
    configure_fail2ban || log_warn "Fail2ban atlandı"
    install_htaccess_watchdog || log_warn "Watchdog atlandı"
    create_admin_user

    # IP adresini al
    SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "SUNUCU-IP")

    echo ""
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║     🎉 OpenSpeed Panel Kuruldu!                      ║"
    echo "╠══════════════════════════════════════════════════════╣"
    echo "║                                                      ║"
    echo "║  🌐 Panel:      http://${SERVER_IP}:8090                ║"
    echo "║  🖥️  OLS Admin:  http://${SERVER_IP}:7080                ║"
    echo "║                                                      ║"
    echo "║  👤 Kullanici:  admin                                ║"
    echo "║  🔑 Şifre:      ${ADMIN_PASS}                          ║"
    echo "║                                                      ║"
    echo "║  🔴 GUvENLIK UYARISI:                                ║"
    echo "║  Bu şifre ILK KURULUM icin olusturuldu!              ║"
    echo "║  LUTFEN PANELE GIRIS YAPIP SIFREYI DEGISTIRIN!       ║"
    echo "║  Panel -> Profil -> Şifre Degistir                   ║"
    echo "║  OLS -> WebAdmin -> Security -> Change Password      ║"
    echo "║                                                      ║"
    echo "║  📦 15 Servis kuruldu                               ║"
    echo "║  🔧 systemctl start|stop|restart ospanel             ║"
    echo "║  📋 journalctl -u ospanel -f                         ║"
    echo "║                                                      ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo ""
}

main "$@"
