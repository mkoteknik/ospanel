#!/usr/bin/env bash
#
# OpenSpeed Panel - .htaccess Watchdog
# OLS .htaccess değişikliklerini izler ve otomatik reload yapar
#
# Apache her istekte .htaccess okur, OLS sadece başlangıçta okur.
# Bu servis inotify ile .htaccess değişikliklerini izler ve
# OLS'e graceful restart gönderir.
#

WATCH_DIRS="/home"
OLS_BIN="/usr/local/lsws/bin/lshttpd"
DEBOUNCE_SEC=3  # Aynı dosyaya ardışık değişiklikleri birleştir

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a /var/log/ospanel-htaccess-watchdog.log
}

# inotifywait kurulu mu?
if ! command -v inotifywait &>/dev/null; then
    log "inotifywait bulunamadı. Kurulum: apt install inotify-tools"
    exit 1
fi

# OLS binary var mı?
if [[ ! -f "$OLS_BIN" ]]; then
    log "OLS binary bulunamadı: $OLS_BIN"
    exit 1
fi

log "🔄 .htaccess Watchdog başlatıldı - İzlenen dizinler: $WATCH_DIRS"

LAST_RELOAD=0

# Sonsuz döngü - .htaccess dosyalarını izle
inotifywait -m -r \
    --include '(\.htaccess$|\.htpasswd$|vhconf\.xml$)' \
    -e modify,create,delete,move \
    "$WATCH_DIRS" 2>/dev/null | while read -r dir action file; do

    log "📝 Değişiklik: $action - $dir$file"

    # Debounce: son reload'dan bu yana $DEBOUNCE_SEC geçti mi?
    NOW=$(date +%s)
    if (( NOW - LAST_RELOAD < DEBOUNCE_SEC )); then
        continue
    fi

    # OLS konfigürasyon test
    if "$OLS_BIN" -t &>/dev/null; then
        log "✅ Konfigürasyon geçerli, graceful restart yapılıyor..."
        "$OLS_BIN" -r &>/dev/null
        LAST_RELOAD=$NOW
        log "🔄 OLS reload tamamlandı"
    else
        log "❌ Konfigürasyon hatası! Reload yapılmadı."
    fi
done
