<script setup lang="ts">
const features = [
  { icon: '🔒', title: 'Let\'s Encrypt', desc: 'Ücretsiz SSL sertifikası. Domain başına otomatik kurulum ve yenileme.', tag: 'Önerilen' },
  { icon: '🔄', title: 'Otomatik Yenileme', desc: 'Sertifika süresi dolmadan 30 gün önce otomatik yenileme.', tag: 'Otomatik' },
  { icon: '⭐', title: 'Wildcard Sertifika', desc: '*.site.com için wildcard SSL. DNS challenge ile doğrulama.', tag: 'Pro' },
  { icon: '📤', title: 'Manuel Yükleme', desc: 'Kendi SSL sertifikanızı yükleyin. PEM, CRT, KEY format desteği.', tag: 'Esnek' },
  { icon: '🛡️', title: 'HSTS', desc: 'HTTP Strict Transport Security. Tüm bağlantıları HTTPS\'e zorlar.', tag: 'Güvenlik' },
  { icon: '📋', title: 'Sertifika Zinciri', desc: 'Full chain doğrulama. Intermediate ve root CA kontrolü.', tag: 'Doğrulama' },
]
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>🔒 SSL Yönetimi</h2>
      <p>Domainlerinize ücretsiz SSL sertifikası kurun ve yönetin.</p>
    </div>

    <div class="card-grid">
      <div v-for="f in features" :key="f.title" class="ssl-card">
        <div class="ssl-icon">{{ f.icon }}</div>
        <h3>{{ f.title }}</h3>
        <p>{{ f.desc }}</p>
        <span class="ssl-tag">{{ f.tag }}</span>
      </div>
    </div>

    <div class="info-banner">
      💡 SSL sertifikalarını Domain Yönetimi → Domain Detay → SSL Kur butonundan kurabilirsiniz.
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

.ssl-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
  position: relative;
  transition: all 0.2s;
}
.ssl-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.08); }

.ssl-icon { font-size: 28px; margin-bottom: 12px; }
.ssl-card h3 { margin: 0 0 8px; font-size: 16px; color: #1a1a2e; }
.ssl-card p { margin: 0; font-size: 13px; color: #777; line-height: 1.5; }

.ssl-tag {
  position: absolute; top: 20px; right: 20px;
  font-size: 11px; padding: 3px 10px; border-radius: 20px;
  background: #e8f4fd; color: #0f3460; font-weight: 600;
}

.info-banner {
  background: #e8f4fd; color: #0f3460;
  padding: 14px 20px; border-radius: 10px; font-size: 14px;
}
</style>
