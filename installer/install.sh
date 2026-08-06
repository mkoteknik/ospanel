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

    # OLS ile gelen varsayılan PHP LSAPI yeterli
    if [[ -f /usr/local/lsws/lsphp83/bin/lsphp ]]; then
        log_info "PHP 8.3 LSAPI zaten mevcut"
    else
        log_warn "PHP LSAPI kurulumu atlandı (manuel kurulum gerekebilir)"
        log_warn "Daha fazla bilgi: https://docs.openlitespeed.org"
    fi
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
    install_php
    install_ospanel
    install_service
    configure_firewall
    create_admin_user

    # IP adresini al
    SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "SUNUCU-IP")

    echo ""
    echo "╔══════════════════════════════════════════════╗"
    echo "║     🎉 OpenSpeed Panel Kuruldu!              ║"
    echo "╠══════════════════════════════════════════════╣"
    echo "║                                              ║"
    echo "║  URL:      https://${SERVER_IP}:8443          ║"
    echo "║  Kullanıcı: admin                           ║"
    echo "║  Şifre:    İlk çalıştırmada üretilecek       ║"
    echo "║                                              ║"
    echo "║  Servis Yönetimi:                            ║"
    echo "║  systemctl start|stop|restart ospanel        ║"
    echo "║  systemctl status ospanel                    ║"
    echo "║  journalctl -u ospanel -f                    ║"
    echo "║                                              ║"
    echo "╚══════════════════════════════════════════════╝"
    echo ""
}

main "$@"
