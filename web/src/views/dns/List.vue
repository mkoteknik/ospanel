<template>
  <div class="page">
    <h2>🔧 DNS Yönetimi</h2>
    <div class="info-banner">
      ℹ️ DNS zone ve kayıt yönetimi BIND9 entegrasyonu ile çalışır. Domain detay sayfasından DNS kayıtlarını yönetebilirsiniz.
    </div>
    <div class="empty-state">
      <div class="empty-icon">🔧</div>
      <h3>DNS Yöneticisi</h3>
      <p>A, AAAA, CNAME, MX, TXT, NS, SRV kayıtlarını yönetin.</p>
    </div>
  </div>
</template>

<style scoped>
.page { max-width: 1200px; }
h2 { margin: 0 0 16px; }
.info-banner { background: #e8f4fd; color: #0f3460; padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 14px; }
.empty-state { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 8px; }
.empty-state p { color: #888; margin: 0; }
</style>
