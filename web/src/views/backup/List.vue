<script setup lang="ts">
const backupCards = [
  { icon: '📅', title: 'Zamanlanmış Yedekleme', desc: 'Günlük, haftalık veya özel cron ile otomatik yedekleme.', status: 'Hazır' },
  { icon: '📦', title: 'Tam Yedekleme', desc: 'Tüm dosya ve veritabanlarının komple yedeği.', status: 'Hazır' },
  { icon: '🔄', title: 'Artımlı Yedekleme', desc: 'Sadece değişen dosyaları yedekleyerek disk tasarrufu.', status: 'Hazır' },
  { icon: '☁️', title: 'Uzak Hedef', desc: 'S3, FTP, SFTP veya SCP ile uzak sunucuya yedekleme.', status: 'Yakında' },
  { icon: '⏪', title: 'Geri Yükleme', desc: 'Tek tıkla yedekten geri dönme. Dosya veya veritabanı seviyesinde.', status: 'Hazır' },
  { icon: '📊', title: 'Yedek Raporu', desc: 'Yedekleme geçmişi, boyut istatistikleri ve başarı durumu.', status: 'Yakında' },
]
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>💾 Yedekleme</h2>
      <p>Web sitelerinizi ve veritabanlarınızı güvenceye alın.</p>
    </div>

    <div class="card-grid">
      <div v-for="c in backupCards" :key="c.title" class="backup-card">
        <div class="bc-icon">{{ c.icon }}</div>
        <h3>{{ c.title }}</h3>
        <p>{{ c.desc }}</p>
        <span class="bc-status">{{ c.status }}</span>
      </div>
    </div>

    <div class="cta-card">
      <div>
        <h3>🚀 Otomatik yedekleme başlatın</h3>
        <p>Linux sunucuda cron job tanımlanarak yedekleme otomatik çalışır.</p>
      </div>
      <button class="btn-action">Yedekleme Planla</button>
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
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.backup-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
  transition: all 0.2s;
  position: relative;
}

.backup-card:hover {
  box-shadow: 0 4px 16px rgba(0,0,0,0.08);
}

.bc-icon { font-size: 28px; margin-bottom: 12px; }
.backup-card h3 { margin: 0 0 8px; font-size: 16px; color: #1a1a2e; }
.backup-card p { margin: 0; font-size: 13px; color: #777; line-height: 1.5; }

.bc-status {
  position: absolute;
  top: 20px;
  right: 20px;
  font-size: 11px;
  padding: 3px 10px;
  border-radius: 20px;
  background: #d4edda;
  color: #155724;
  font-weight: 600;
}

.cta-card {
  background: linear-gradient(135deg, #1a1a2e, #0f3460);
  border-radius: 12px;
  padding: 28px 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: white;
}

.cta-card h3 { margin: 0 0 4px; font-size: 18px; }
.cta-card p { margin: 0; opacity: 0.8; font-size: 14px; }

.btn-action {
  padding: 10px 24px;
  background: white;
  color: #0f3460;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  white-space: nowrap;
}

.btn-action:hover { background: #f0f0f0; }
</style>
