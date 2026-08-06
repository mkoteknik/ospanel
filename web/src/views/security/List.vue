<script setup lang="ts">
const features = [
  {
    icon: '🔐', title: 'Fail2ban',
    desc: 'Saldırı tespit ve engelleme sistemi. SSH, HTTP brute-force saldırılarını otomatik bloklar.',
    status: 'active', statusText: 'Aktif',
  },
  {
    icon: '🛡️', title: 'ModSecurity WAF',
    desc: 'Web uygulama güvenlik duvarı. SQL injection, XSS ve diğer web saldırılarına karşı korur.',
    status: 'active', statusText: 'Aktif',
  },
  {
    icon: '🚫', title: 'IP Engelleme',
    desc: 'IP kara liste / beyaz liste yönetimi. Ülke bazlı veya özel IP engelleme kuralları.',
    status: 'inactive', statusText: 'Yapılandırılmadı',
  },
  {
    icon: '🔑', title: 'İki Faktörlü (2FA)',
    desc: 'Google Authenticator ile iki faktörlü kimlik doğrulama. Panel girişinde ek güvenlik katmanı.',
    status: 'inactive', statusText: 'Kurulmadı',
  },
  {
    icon: '📋', title: 'Audit Log',
    desc: 'Tüm panel işlemlerinin kaydı. Kim, ne zaman, ne yaptı? Tam denetim izi.',
    status: 'active', statusText: 'Aktif',
  },
  {
    icon: '🔒', title: 'SSL/TLS Zorunluluğu',
    desc: 'Tüm domainlerde HTTPS zorunlu. HSTS, güvenli cipher suite ve TLS 1.2+ desteği.',
    status: 'active', statusText: 'Aktif',
  },
  {
    icon: '🕐', title: 'Rate Limiting',
    desc: 'API ve login endpointlerinde istek sınırlama. DDoS ve brute-force koruması.',
    status: 'active', statusText: 'Aktif',
  },
  {
    icon: '🔍', title: 'Dosya Integrity Monitor',
    desc: 'Kritik sistem dosyalarında değişiklik tespiti. Şüpheli değişikliklerde anında uyarı.',
    status: 'inactive', statusText: 'Yakında',
  },
]
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>🛡️ Güvenlik</h2>
        <p>Sunucu güvenlik ayarlarını yapılandırın ve izleyin.</p>
      </div>
    </div>

    <div class="card-grid">
      <div v-for="f in features" :key="f.title" class="feature-card">
        <div class="card-icon">{{ f.icon }}</div>
        <div class="card-body">
          <div class="card-title-row">
            <h3>{{ f.title }}</h3>
            <span :class="'status-badge ' + (f.status === 'active' ? 'on' : 'off')">
              {{ f.statusText }}
            </span>
          </div>
          <p>{{ f.desc }}</p>
        </div>
      </div>
    </div>

    <div class="stats-row">
      <div class="mini-stat">
        <div class="mini-val">5</div>
        <div class="mini-label">Aktif Koruma</div>
      </div>
      <div class="mini-stat">
        <div class="mini-val">0</div>
        <div class="mini-label">Son 24s Blok</div>
      </div>
      <div class="mini-stat">
        <div class="mini-val">3</div>
        <div class="mini-label">Bekleyen</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.feature-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  display: flex;
  gap: 16px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
  transition: all 0.2s;
}

.feature-card:hover {
  box-shadow: 0 4px 16px rgba(0,0,0,0.08);
  border-color: #e0e0e0;
}

.card-icon { font-size: 32px; flex-shrink: 0; }
.card-body { flex: 1; min-width: 0; }

.card-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.card-title-row h3 { margin: 0; font-size: 16px; color: #1a1a2e; }
.card-body p { margin: 0; font-size: 13px; color: #777; line-height: 1.5; }

.status-badge {
  padding: 3px 10px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.status-badge.on { background: #d4edda; color: #155724; }
.status-badge.off { background: #f8f9fa; color: #888; }

.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.mini-stat {
  background: white;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
}

.mini-val { font-size: 28px; font-weight: 700; color: #1a1a2e; }
.mini-label { font-size: 13px; color: #888; margin-top: 4px; }
</style>
