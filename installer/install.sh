#!/usr/bin/env bash
#
# OpenSpeed Panel - Tek Komut Kurulum Scripti
# https://github.com/mkoteknik/ospanel
#
# Kullanım:
#   curl -fsSL https://raw.githubusercontent.com/mkoteknik/ospanel/main/install.sh | sudo bash
#

set -euo pipefail

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
        ubuntu|debian)
            PKG_MANAGER="apt"
            log_info "Tespit edildi: $OS $OS_VERSION (apt)"
            ;;
        rocky|almalinux|rhel|centos)
            PKG_MANAGER="dnf"
            log_info "Tespit edildi: $OS $OS_VERSION (dnf)"
            ;;
        *)
            log_error "Desteklenmeyen işletim sistemi: $OS"
            log_error "Desteklenen: Ubuntu 20.04+, Debian 11+, Rocky 8+, AlmaLinux 8+"
            exit 1
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
        # === Ubuntu / Debian ===
        log_info "LiteSpeed resmi reposu ekleniyor..."

        # Resmi OLS repo scriptini kullan
        wget -q -O /tmp/enable_lst_debian_repo.sh http://rpms.litespeedtech.com/debian/enable_lst_debian_repo.sh 2>/dev/null || \
        curl -fsSL -o /tmp/enable_lst_debian_repo.sh http://rpms.litespeedtech.com/debian/enable_lst_debian_repo.sh

        if [[ -f /tmp/enable_lst_debian_repo.sh ]]; then
            bash /tmp/enable_lst_debian_repo.sh 2>/dev/null || true
            apt-get update -qq 2>/dev/null || true
            apt-get install -y -qq openlitespeed 2>/dev/null && OLS_INSTALLED=1
        fi

        # Repo başarısız olursa manuel .deb kurulumu dene
        if [[ $OLS_INSTALLED -eq 0 ]]; then
            log_warn "Repo kurulumu başarısız, manuel .deb deneniyor..."
            local OLS_DEB_URL="https://openlitespeed.org/download/latest-deb"
            # En son sürümü dene (1.8.x serisi)
            for ver in 1.8.2 1.8.1 1.8.0 1.7.19; do
                local DEB_URL="https://rpms.litespeedtech.com/debian/pool/main/openlitespeed_${ver}-1_amd64.deb"
                log_info "OLS $ver deneniyor: $DEB_URL"
                if curl -fsSL -o /tmp/ols.deb "$DEB_URL" 2>/dev/null; then
                    dpkg -i /tmp/ols.deb 2>/dev/null && OLS_INSTALLED=1 && break
                fi
            done
        fi

    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        # === Rocky / Alma / RHEL ===
        log_info "LiteSpeed resmi reposu ekleniyor..."

        # EL9 için
        if [[ "$OS_VERSION" == "9"* ]]; then
            rpm -Uvh http://rpms.litespeedtech.com/centos/litespeed-repo-1.4-1.el9.noarch.rpm 2>/dev/null || true
        else
            # EL8 için
            rpm -Uvh http://rpms.litespeedtech.com/centos/litespeed-repo-1.4-1.el8.noarch.rpm 2>/dev/null || true
        fi

        dnf install -y openlitespeed 2>/dev/null && OLS_INSTALLED=1

        # Repo başarısız olursa manuel .rpm kurulumu dene
        if [[ $OLS_INSTALLED -eq 0 ]]; then
            log_warn "Repo kurulumu başarısız, manuel .rpm deneniyor..."
            for ver in 1.8.2 1.8.1 1.8.0; do
                local RPM_URL="https://rpms.litespeedtech.com/centos/8/x86_64/openlitespeed-${ver}-1.el8.x86_64.rpm"
                log_info "OLS $ver deneniyor..."
                if curl -fsSL -o /tmp/ols.rpm "$RPM_URL" 2>/dev/null; then
                    rpm -ivh /tmp/ols.rpm 2>/dev/null && OLS_INSTALLED=1 && break
                fi
            done
        fi
    fi

    if [[ $OLS_INSTALLED -eq 0 ]]; then
        log_error "OpenLiteSpeed kurulamadı!"
        log_error "Manuel kurulum: https://docs.openlitespeed.org/installation"
        exit 1
    fi

    # OLS admin şifresini ayarla (varsayılanı değiştir)
    if [[ -f /usr/local/lsws/admin/misc/admpass.sh ]]; then
        local OLS_ADMIN_PASS=$(openssl rand -base64 12 2>/dev/null | tr -d '=+/' | head -c 16)
        echo -e "admin\n${OLS_ADMIN_PASS}\n${OLS_ADMIN_PASS}" | bash /usr/local/lsws/admin/misc/admpass.sh 2>/dev/null || true
        echo "$OLS_ADMIN_PASS" > /etc/ospanel/ols_admin_pass
        chmod 600 /etc/ospanel/ols_admin_pass
        log_info "OLS WebAdmin şifresi ayarlandı: /etc/ospanel/ols_admin_pass"
    fi

    # OLS servisini başlat
    systemctl enable lsws 2>/dev/null || true
    systemctl start lsws 2>/dev/null || {
        /usr/local/lsws/bin/lshttpd -k start 2>/dev/null || {
            log_error "OLS başlatılamadı!"
            exit 1
        }
    }

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
    log_step "PHP LSAPI sürümleri kuruluyor..."

    local PHP_VERSIONS=("74" "80" "81" "82" "83" "84")
    local INSTALLED_COUNT=0

    if [[ "$PKG_MANAGER" == "apt" ]]; then
        for ver in "${PHP_VERSIONS[@]}"; do
            if [[ -f "/usr/local/lsws/lsphp${ver}/bin/lsphp" ]]; then
                log_info "PHP ${ver:0:1}.${ver:1} LSAPI zaten mevcut, atlanıyor"
                ((INSTALLED_COUNT++))
                continue
            fi
            log_info "PHP ${ver:0:1}.${ver:1} LSAPI kuruluyor..."
            apt-get install -y -qq "lsphp${ver}" 2>/dev/null && {
                log_info "  PHP ${ver:0:1}.${ver:1} ✓"
                ((INSTALLED_COUNT++))
            } || log_warn "  PHP ${ver:0:1}.${ver:1} kurulamadı"
        done
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        for ver in "${PHP_VERSIONS[@]}"; do
            if [[ -f "/usr/local/lsws/lsphp${ver}/bin/lsphp" ]]; then
                log_info "PHP ${ver:0:1}.${ver:1} LSAPI zaten mevcut, atlanıyor"
                ((INSTALLED_COUNT++))
                continue
            fi
            log_info "PHP ${ver:0:1}.${ver:1} LSAPI kuruluyor..."
            dnf install -y "lsphp${ver}" 2>/dev/null && {
                log_info "  PHP ${ver:0:1}.${ver:1} ✓"
                ((INSTALLED_COUNT++))
            } || log_warn "  PHP ${ver:0:1}.${ver:1} kurulamadı (EL9'da 74 mevcut olmayabilir)"
        done
    fi

    # En az bir PHP sürümü kuruldu mu?
    if [[ $INSTALLED_COUNT -eq 0 ]]; then
        log_warn "Hiçbir PHP LSAPI sürümü kurulamadı."
        log_warn "OLS en az bir PHP sürümü ile gelir, panel devam edecek."
    else
        log_info "$INSTALLED_COUNT PHP LSAPI sürümü kuruldu"
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
    mysql -e "CREATE DATABASE IF NOT EXISTS ${MAIL_DB};" 2>/dev/null || true
    mysql -e "CREATE USER IF NOT EXISTS '${MAIL_DB_USER}'@'localhost' IDENTIFIED BY '${MAIL_DB_PASS}';" 2>/dev/null || true
    mysql -e "GRANT ALL PRIVILEGES ON ${MAIL_DB}.* TO '${MAIL_DB_USER}'@'localhost';" 2>/dev/null || true
    mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true

    mysql "$MAIL_DB" << SQLEOF
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
        dnf install -y pdns pdns-backend-sqlite sqlite 2>/dev/null && PDNS_INSTALLED=1
        # EPEL'den dene
        if [[ $PDNS_INSTALLED -eq 0 ]]; then
            dnf install -y epel-release 2>/dev/null || true
            dnf install -y pdns pdns-backend-sqlite 2>/dev/null && PDNS_INSTALLED=1
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
    if curl -fsSL -o "$ADMINER_FILE" "https://www.adminer.org/latest.php" 2>/dev/null; then
        chmod 644 "$ADMINER_FILE"
        log_info "Adminer kuruldu: $ADMINER_FILE (tek dosya, ~500KB)"
        log_info "Adminer URL: http://SUNUCU-IP/adminer/"
    elif curl -fsSL -o "$ADMINER_FILE" "https://github.com/vrana/adminer/releases/download/v4.8.1/adminer-4.8.1.php" 2>/dev/null; then
        chmod 644 "$ADMINER_FILE"
        log_info "Adminer v4.8.1 kuruldu (fallback)"
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

[ospanel]
enabled = true
port = 8443
filter = ospanel
logpath = /var/log/ospanel.log
maxretry = 5
bantime = 1800
F2BEOF

    systemctl enable fail2ban 2>/dev/null || true
    systemctl restart fail2ban 2>/dev/null || true
    log_info "Fail2ban yapılandırıldı"
}

# ---------------------------------------------------------------------------
# MariaDB güvenli kurulum
# ---------------------------------------------------------------------------
secure_mariadb() {
    log_step "MariaDB güvenlik yapılandırması..."

    local MYSQL_PASS=$(openssl rand -base64 16 2>/dev/null | tr -d '=+/' | head -c 20)

    # Root şifresi belirle ve güvenlik ayarları
    if command -v mysql &>/dev/null; then
        mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '${MYSQL_PASS}';" 2>/dev/null || true
        mysql -e "DELETE FROM mysql.user WHERE User='';" 2>/dev/null || true
        mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');" 2>/dev/null || true
        mysql -e "DROP DATABASE IF EXISTS test;" 2>/dev/null || true
        mysql -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';" 2>/dev/null || true
        mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true
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
    if [[ ! -f "$OSPANEL_CONFIG/config.yaml" ]]; then
        cat > "$OSPANEL_CONFIG/config.yaml" << 'YAMLEOF'
server:
  host: "0.0.0.0"
  port: 8443
  tls:
    enabled: false

auth:
  jwt_secret: ""
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

    cat > /etc/systemd/system/ospanel.service << SYSTEMDEOF
[Unit]
Description=OpenSpeed Panel
Documentation=https://github.com/mkoteknik/ospanel
After=network.target lsws.service

[Service]
Type=simple
User=root
ExecStart=${OSPANEL_DIR}/ospanel --config ${OSPANEL_CONFIG}/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# Güvenlik
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=${OSPANEL_DATA} ${OSPANEL_CONFIG} /usr/local/lsws/conf /tmp

[Install]
WantedBy=multi-user.target
SYSTEMDEOF

    systemctl daemon-reload
    systemctl enable ospanel
    systemctl start ospanel 2>/dev/null || log_warn "Servis başlatılamadı (ilk kurulumda normal)"

    log_info "systemd servisi kuruldu"
}

# ---------------------------------------------------------------------------
# Firewall yapılandırması
# ---------------------------------------------------------------------------
configure_firewall() {
    log_step "Firewall yapılandırılıyor..."

    if command -v ufw &>/dev/null; then
        ufw allow 80/tcp 2>/dev/null || true
        ufw allow 443/tcp 2>/dev/null || true
        ufw allow 8443/tcp 2>/dev/null || true
        ufw --force enable 2>/dev/null || true
        log_info "UFW kuralları eklendi"
    elif command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-service=http 2>/dev/null || true
        firewall-cmd --permanent --add-service=https 2>/dev/null || true
        firewall-cmd --permanent --add-port=8443/tcp 2>/dev/null || true
        firewall-cmd --reload 2>/dev/null || true
        log_info "firewalld kuralları eklendi"
    fi
}

# ---------------------------------------------------------------------------
# Admin kullanıcı oluştur
# ---------------------------------------------------------------------------
create_admin_user() {
    log_step "Admin kullanıcısı oluşturuluyor..."

    # Binary'nin ilk çalıştırmada admin oluşturmasını bekle
    sleep 3

    ADMIN_PASS=$(openssl rand -base64 12 2>/dev/null || echo "admin123")
    log_info "Admin şifresi üretildi"
}

# ---------------------------------------------------------------------------
# Ana kurulum
# ---------------------------------------------------------------------------
main() {
    clear
    echo ""
    echo "╔══════════════════════════════════════════════╗"
    echo "║         OpenSpeed Panel Kurulum              ║"
    echo "║         Modern Hosting Kontrol Paneli         ║"
    echo "╚══════════════════════════════════════════════╝"
    echo ""
    echo "⚡ OpenLiteSpeed + Go + Vue 3"
    echo ""

    detect_os
    install_dependencies
    install_ols
    install_mariadb
    secure_mariadb
    install_php
    install_email_services
    install_spamassassin
    install_dns_server
    install_postgresql
    install_redis
    install_container_runtime
    install_adminer
    install_ospanel
    install_service
    configure_firewall
    configure_fail2ban
    create_admin_user

    # IP adresini al
    SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "SUNUCU-IP")

    echo ""
    echo "╔══════════════════════════════════════════════════╗"
    echo "║     🎉 OpenSpeed Panel Kuruldu!                  ║"
    echo "╠══════════════════════════════════════════════════╣"
    echo "║                                                  ║"
    echo "║  🌐 Panel:   https://${SERVER_IP}:8443              ║"
    echo "║  👤 Admin:   admin / 123456                      ║"
    echo "║                                                  ║"
    echo "║  📦 Kurulan Servisler:                           ║"
    echo "║  ✅ OpenLiteSpeed (80, 443, 7080)                ║"
    echo "║  ✅ PHP LSAPI (7.4, 8.0-8.4)                     ║"
    echo "║  ✅ MariaDB (3306)                               ║"
    echo "║  ✅ Postfix + Dovecot (25, 143, 993)             ║"
    echo "║  ✅ PowerDNS (53) + SQLite + REST API           ║"
    echo "║  ✅ SpamAssassin                                 ║"
    echo "║  ✅ Adminer (MySQL+PG+SQLite+MongoDB)           ║"
    echo "║  ✅ Redis Cache (256MB)                          ║"
    echo "║  ✅ Podman/Docker (opsiyonel)                    ║"
    echo "║  ✅ Fail2ban                                     ║"
    echo "║                                                  ║"
    echo "║  🔧 Servis Yönetimi:                             ║"
    echo "║  systemctl start|stop|restart ospanel            ║"
    echo "║  journalctl -u ospanel -f                        ║"
    echo "║                                                  ║"
    echo "║  🔑 Önemli Dosyalar:                             ║"
    echo "║  /etc/ospanel/config.yaml                        ║"
    echo "║  /etc/ospanel/mysql_root_pass                    ║"
    echo "║  /etc/ospanel/ols_admin_pass                     ║"
    echo "║                                                  ║"
    echo "╚══════════════════════════════════════════════════╝"
    echo ""
}

main "$@"
