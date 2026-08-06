<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const sidebarOpen = ref(true)

function isActive(path: string) {
  const current = '/' + (route.path.split('/')[1] || '')
  return current === path
}

const menuItems = [
  { path: '/', label: '📊 Dashboard' },
  { path: '/domains', label: '🌐 Domainler' },
  { path: '/email', label: '📧 E-Posta' },
  { path: '/databases', label: '🗄️ Veritabanları' },
  { path: '/files', label: '📁 Dosyalar' },
  { path: '/ssl', label: '🔒 SSL' },
  { path: '/dns', label: '🔧 DNS' },
  { path: '/backup', label: '💾 Yedekleme' },
  { path: '/monitor', label: '📈 Monitoring' },
  { path: '/security', label: '🛡️ Güvenlik' },
]
</script>

<template>
  <div class="dashboard-layout">
    <aside class="sidebar" :class="{ collapsed: !sidebarOpen }">
      <div class="sidebar-header">
        <span v-if="sidebarOpen" class="logo">⚡ OSPanel</span>
        <span v-else class="logo-sm">⚡</span>
      </div>
      <nav class="sidebar-nav">
        <a
          v-for="item in menuItems"
          :key="item.path"
          :href="item.path"
          :class="{ active: isActive(item.path) }"
          @click.prevent="router.push(item.path)"
        >
          {{ sidebarOpen ? item.label : item.label.split(' ')[0] }}
        </a>
      </nav>
    </aside>

    <div class="main-area">
      <header class="topbar">
        <button class="toggle-btn" @click="sidebarOpen = !sidebarOpen">
          {{ sidebarOpen ? '◀' : '▶' }}
        </button>
        <div class="topbar-right">
          <span class="user-info">👤 {{ authStore.user?.username || 'Kullanıcı' }}</span>
          <button class="logout-btn" @click="authStore.logout()">Çıkış</button>
        </div>
      </header>

      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.dashboard-layout {
  display: flex;
  height: 100vh;
  background: #f5f6fa;
}

.sidebar {
  width: 240px;
  background: #1a1a2e;
  color: white;
  display: flex;
  flex-direction: column;
  transition: width 0.2s;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid rgba(255,255,255,0.1);
}

.logo { font-size: 20px; font-weight: 700; }
.logo-sm { font-size: 24px; }

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 8px;
  overflow-y: auto;
}

.sidebar-nav a {
  color: rgba(255,255,255,0.7);
  text-decoration: none;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.15s;
  white-space: nowrap;
  overflow: hidden;
}

.sidebar-nav a:hover {
  background: rgba(255,255,255,0.1);
  color: white;
}

.sidebar-nav a.active {
  background: #0f3460;
  color: white;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  height: 64px;
  background: white;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}

.toggle-btn {
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
}

.toggle-btn:hover { background: #f0f0f0; }

.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-info { font-size: 14px; color: #333; }

.logout-btn {
  background: #e74c3c;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.logout-btn:hover { background: #c0392b; }

.content {
  flex: 1;
  padding: 32px;
  overflow-y: auto;
  display: flex;
  justify-content: center;
  background: #f0f2f5;
}

.content > * {
  width: 100%;
  max-width: 1200px;
}
</style>
