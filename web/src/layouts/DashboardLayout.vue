<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'

const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const sidebarOpen = ref(true)

function toggleLocale() {
  const next = getLocale() === 'tr' ? 'en' : 'tr'
  setLocale(next)
  locale.value = next as any
}

// Dropdown states — default open for primary, closed for others on mobile
const openMain = ref(true)
const openInfra = ref(true)
const openTools = ref(false)
const openAdmin = ref(false)

onMounted(() => {
  authStore.fetchMe()
  if (window.innerWidth < 1024) {
    sidebarOpen.value = false
    openInfra.value = false
  }
})

async function openOLS() {
  try {
    const res = await fetch('/api/v1/ols/info', { headers: { Authorization: 'Bearer ' + authStore.accessToken } })
    const data = await res.json()
    if (data.direct_url) window.open(data.direct_url, '_blank')
    else if (data.ols_admin_url) window.open(data.ols_admin_url, '_blank')
    else window.open('http://' + window.location.hostname + ':7080', '_blank')
  } catch {
    window.open('http://' + window.location.hostname + ':7080', '_blank')
  }
}

function isActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path === path || route.path.startsWith(path + '/')
}

const mainNav = computed(() => [
  { path: '/', label: t('nav.overview'), desc: t('nav.overviewDesc'), icon: 'grid' },
  { path: '/domains', label: t('nav.domains'), desc: t('nav.domainsDesc'), icon: 'globe' },
  { path: '/databases', label: t('nav.databases'), desc: t('nav.databasesDesc'), icon: 'database' },
  { path: '/files', label: t('nav.files'), desc: t('nav.filesDesc'), icon: 'folder' },
  { path: '/email', label: t('nav.email'), desc: t('nav.emailDesc'), icon: 'mail' },
  { path: '/dns', label: t('nav.dns'), desc: t('nav.dnsDesc'), icon: 'dns' },
  { path: '/ssl', label: t('nav.ssl'), desc: t('nav.sslDesc'), icon: 'lock' },
])

const infraNav = computed(() => [
  { path: '/monitor', label: t('nav.monitor'), desc: t('nav.monitorDesc'), icon: 'activity' },
  { path: '/services', label: t('nav.services'), desc: t('nav.servicesDesc'), icon: 'cpu' },
  { path: '/logs', label: t('nav.logs'), desc: t('nav.logsDesc'), icon: 'file' },
  { path: '/security', label: t('nav.security'), desc: t('nav.securityDesc'), icon: 'shield' },
  { path: '/backup', label: t('nav.backup'), desc: t('nav.backupDesc'), icon: 'harddrive' },
  { path: '/cron', label: t('nav.cron'), desc: t('nav.cronDesc'), icon: 'clock' },
])

const toolsNav = computed(() => [
  { path: '/cache', label: t('nav.cache'), desc: t('nav.cacheDesc'), icon: 'zap' },
  { path: '/containers', label: t('nav.containers'), desc: t('nav.containersDesc'), icon: 'box' },
  { path: '/ols', label: t('nav.ols'), desc: t('nav.olsDesc'), icon: 'sliders' },
  { path: '/cloudflare', label: t('nav.cloudflare'), desc: t('nav.cloudflareDesc'), icon: 'cloud' },
])

const adminNav = computed(() => [
  { path: '/admin/users', label: t('nav.users'), icon: 'users' },
  { path: '/admin/settings', label: t('nav.settings'), icon: 'settings' },
  { path: '/admin/audit', label: t('nav.audit'), icon: 'clipboard' },
])

const collapsed = computed(() => !sidebarOpen.value)
</script>

<template>
  <div class="aura-layout" :class="{ collapsed: collapsed }" :data-theme="themeStore.resolved">
    <aside class="aura-sidebar">
      <div class="sidebar-head">
        <div class="brand">
          <div class="brand-mark">A</div>
          <div v-if="sidebarOpen" class="brand-text">
            <div class="brand-name">Aura Panel</div>
            <div class="brand-meta">{{ t('nav.admin') === 'Admin' ? 'Server management' : 'Sunucu yönetimi' }}</div>
          </div>
        </div>
        <button class="icon-btn collapse-btn" @click="sidebarOpen = !sidebarOpen" :title="sidebarOpen ? 'Daralt' : 'Genişlet'">
          <span class="chev" :class="{ open: sidebarOpen }">‹</span>
        </button>
      </div>

      <div class="sidebar-scroll">
        <!-- Çalışma — dropdown -->
        <div class="nav-section">
          <button v-if="sidebarOpen" class="nav-section-head" @click="openMain = !openMain">
            <span class="nav-label">{{ t('nav.work') }}</span>
            <span class="section-chevron" :class="{ open: openMain }">›</span>
          </button>
          <div v-show="sidebarOpen ? openMain : true" class="nav-items">
            <a v-for="item in mainNav" :key="item.path" :class="{ active: isActive(item.path) }" @click.prevent="router.push(item.path)" :href="item.path">
              <span class="nav-icon" :class="item.icon">
                <svg v-if="item.icon === 'grid'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/><rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/></svg>
                <svg v-else-if="item.icon === 'globe'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="6"/><path d="M2 8h12M8 2a10 10 0 0 1 0 12M8 2a10 10 0 0 0 0 12"/></svg>
                <svg v-else-if="item.icon === 'database'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><ellipse cx="8" cy="4" rx="5" ry="2.5"/><path d="M3 4v8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5V4"/><path d="M3 8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5"/></svg>
                <svg v-else-if="item.icon === 'folder'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 5.5A1.5 1.5 0 0 1 3.5 4H6l1.2 1.5H12.5A1.5 1.5 0 0 1 14 7v5.5A1.5 1.5 0 0 1 12.5 14H3.5A1.5 1.5 0 0 1 2 12.5V5.5Z"/></svg>
                <svg v-else-if="item.icon === 'mail'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="3" width="12" height="10" rx="1.5"/><path d="M2.5 4.5 8 8.5 13.5 4.5"/></svg>
                <svg v-else-if="item.icon === 'dns'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="2.5"/><path d="M8 1v1.5M8 13.5V15M1 8h1.5M13.5 8H15M3.2 3.2l1 1M11.8 11.8l1 1M12.8 3.2l-1 1M4.2 11.8l-1 1"/></svg>
                <svg v-else-if="item.icon === 'lock'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="3.5" y="6.5" width="9" height="7" rx="1.2"/><path d="M6 6.5V4.8a2 2 0 0 1 4 0v1.7"/><circle cx="8" cy="10" r="1"/></svg>
              </span>
              <span class="nav-main">
                <span class="nav-label-text">{{ item.label }}</span>
                <span v-if="sidebarOpen" class="nav-desc">{{ item.desc }}</span>
              </span>
              <span v-if="isActive(item.path)" class="active-bar"></span>
            </a>
          </div>
        </div>

        <div class="nav-section">
          <button v-if="sidebarOpen" class="nav-section-head" @click="openInfra = !openInfra">
            <span class="nav-label">{{ t('nav.infra') }}</span>
            <span class="section-chevron" :class="{ open: openInfra }">›</span>
          </button>
          <div v-show="sidebarOpen ? openInfra : true" class="nav-items">
            <a v-for="item in infraNav" :key="item.path" :class="{ active: isActive(item.path) }" @click.prevent="router.push(item.path)" :href="item.path">
              <span class="nav-icon" :class="item.icon">
                <svg v-if="item.icon === 'activity'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M1 8h3l2-4 2 8 2-4h5"/></svg>
                <svg v-else-if="item.icon === 'cpu'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="4" y="4" width="8" height="8" rx="1.5"/><path d="M8 1v2M8 13v2M1 8h2M13 8h2M3 3l1.2 1.2M11.8 11.8l1.2 1.2M13 3l-1.2 1.2M4.2 11.8l-1.2 1.2"/></svg>
                <svg v-else-if="item.icon === 'file'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5 2.5A1.5 1.5 0 0 1 6.5 1H9l3 3v8.5A1.5 1.5 0 0 1 10.5 14H6.5A1.5 1.5 0 0 1 5 12.5V2.5Z"/><path d="M9 1v3.5H12"/></svg>
                <svg v-else-if="item.icon === 'shield'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M8 1.5 2.5 4v4.2c0 2.6 1.7 4.9 5.5 6.3 3.8-1.4 5.5-3.7 5.5-6.3V4L8 1.5Z"/><path d="M6.2 8.3 7.6 9.7 9.8 6.5"/></svg>
                <svg v-else-if="item.icon === 'harddrive'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2.5" y="4" width="11" height="8" rx="1.5"/><path d="M2.5 7H13.5"/><circle cx="5.5" cy="10" r="0.9" fill="currentColor" stroke="none"/><circle cx="8" cy="10" r="0.9" fill="currentColor" stroke="none"/></svg>
                <svg v-else-if="item.icon === 'clock'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="5.5"/><path d="M8 5.5V8l2.5 1.5"/></svg>
              </span>
              <span class="nav-main">
                <span class="nav-label-text">{{ item.label }}</span>
                <span v-if="sidebarOpen" class="nav-desc">{{ item.desc }}</span>
              </span>
              <span v-if="isActive(item.path)" class="active-bar"></span>
            </a>
          </div>
        </div>

        <div class="nav-section">
          <button v-if="sidebarOpen" class="nav-section-head" @click="openTools = !openTools">
            <span class="nav-label">{{ t('nav.tools') }}</span>
            <span class="section-chevron" :class="{ open: openTools }">›</span>
          </button>
          <div v-show="sidebarOpen ? openTools : true" class="nav-items">
            <a v-for="item in toolsNav" :key="item.path" :class="{ active: isActive(item.path) }" @click.prevent="router.push(item.path)" :href="item.path">
              <span class="nav-icon" :class="item.icon">
                <svg v-if="item.icon === 'zap'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M9 1 3.5 8.5H7L7 15 12.5 7.5H9L9 1Z"/></svg>
                <svg v-else-if="item.icon === 'box'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2.5 5.2 8 2 13.5 5.2v5.6L8 14 2.5 10.8V5.2Z"/><path d="M2.5 5.2 8 8 13.5 5.2"/><path d="M8 8v6"/></svg>
                <svg v-else-if="item.icon === 'sliders'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M3 5H2M14 5H9M3 11H5M14 11H8"/><circle cx="6.5" cy="5" r="1.8"/><circle cx="9.5" cy="11" r="1.8"/></svg>
                <svg v-else-if="item.icon === 'cloud'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M4.5 10.5A2.5 2.5 0 0 1 4.5 5.5 3.5 3.5 0 0 1 11 5.2 2.5 2.5 0 0 1 11.5 10.5H4.5Z"/></svg>
              </span>
              <span class="nav-main">
                <span class="nav-label-text">{{ item.label }}</span>
                <span v-if="sidebarOpen" class="nav-desc">{{ item.desc }}</span>
              </span>
              <span v-if="isActive(item.path)" class="active-bar"></span>
            </a>
          </div>
        </div>

        <div v-if="authStore.isAdmin" class="nav-section">
          <button v-if="sidebarOpen" class="nav-section-head" @click="openAdmin = !openAdmin">
            <span class="nav-label">{{ t('nav.admin') }}</span>
            <span class="section-chevron" :class="{ open: openAdmin }">›</span>
          </button>
          <div v-show="sidebarOpen ? openAdmin : true" class="nav-items">
            <a v-for="item in adminNav" :key="item.path" :class="{ active: isActive(item.path) }" @click.prevent="router.push(item.path)" :href="item.path">
              <span class="nav-icon admin">
                <svg v-if="item.icon === 'users'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="6" cy="5.5" r="2.2"/><path d="M2.5 12.5c0-2 1.8-3.5 3.5-3.5s3.5 1.5 3.5 3.5"/><circle cx="11" cy="6" r="1.8"/><path d="M10 11.5c0-1.2 1-2 2-2s2 0.8 2 2"/></svg>
                <svg v-else-if="item.icon === 'settings'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="2.2"/><path d="M8 1v1.6M8 13.4V15M1 8h1.6M13.4 8H15M3 3l1.1 1.1M11.9 11.9l1.1 1.1M13 3l-1.1 1.1M4.1 11.9l-1.1 1.1"/></svg>
                <svg v-else-if="item.icon === 'clipboard'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="4" y="3.5" width="8" height="10" rx="1.2"/><path d="M6 3.5V2.8a2 2 0 0 1 4 0v0.7"/><path d="M6.5 7H9.5M6.5 9.5H9.5"/></svg>
              </span>
              <span class="nav-main"><span class="nav-label-text">{{ item.label }}</span></span>
              <span v-if="isActive(item.path)" class="active-bar"></span>
            </a>
          </div>
        </div>
      </div>

      <div class="sidebar-foot">
        <div v-if="sidebarOpen" class="foot-card">
          <div class="foot-title">Aura v1.0</div>
          <div class="foot-sub">Go · Vue 3 · OLS</div>
        </div>
        <div v-else class="foot-dot"></div>
      </div>
    </aside>

    <div class="aura-main">
      <header class="aura-header">
        <div class="header-left">
          <button class="icon-btn menu-btn" @click="sidebarOpen = !sidebarOpen" aria-label="Menü">
            <span class="hamburger"><span></span><span></span><span></span></span>
          </button>
          <div class="header-title"><div class="crumb">{{ route.name || 'Panel' }}</div></div>
        </div>
        <div class="header-right">
          <button class="aura-btn aura-btn-ghost header-action" @click="openOLS">OLS WebAdmin</button>
          <button class="aura-btn aura-btn-ghost" @click="toggleLocale()" :title="t('language.switch')" style="padding:8px 10px; font-size:12px; font-weight:650;">
            {{ locale === 'tr' ? 'EN' : 'TR' }}
          </button>
          <button class="theme-btn" @click="themeStore.toggle()" :title="themeStore.mode">
            <span v-if="themeStore.isDark">☀️</span><span v-else>🌙</span>
          </button>
          <div class="user-chip">
            <span class="user-avatar">{{ (authStore.user?.username || 'A')[0].toUpperCase() }}</span>
            <span v-if="sidebarOpen" class="user-name">{{ authStore.user?.username || t('common.noData') }}</span>
            <span class="user-role">{{ authStore.user?.role || '' }}</span>
          </div>
          <button class="aura-btn aura-btn-primary" @click="authStore.logout()">{{ t('auth.logout') }}</button>
        </div>
      </header>
      <main class="aura-content"><div class="content-inner"><router-view /></div></main>
    </div>
  </div>
</template>

<style scoped>
.aura-layout {
  display: flex;
  height: 100vh;
  background: var(--aura-bg);
  overflow: hidden;
}
.aura-sidebar {
  width: var(--aura-sidebar-w);
  background: var(--aura-surface);
  border-right: 1px solid var(--aura-border);
  display: flex; flex-direction: column;
  box-shadow: 1px 0 0 var(--aura-border), 8px 0 24px rgba(15,23,42,0.04);
  transition: width 0.22s cubic-bezier(0.2,0.8,0.2,1);
  will-change: width;
  z-index: 10;
}
[data-theme="dark"] .aura-sidebar {
  box-shadow: 1px 0 0 var(--aura-border), 12px 0 32px rgba(0,0,0,0.25);
}
.aura-layout.collapsed .aura-sidebar { width: var(--aura-sidebar-collapsed); }
.sidebar-head {
  height: var(--aura-header-h);
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid var(--aura-border);
  background: var(--aura-surface);
  flex-shrink: 0;
}
.brand { display: flex; align-items: center; gap: 10px; min-width: 0; }
.brand-mark {
  width: 37.8px; height: 37.8px; border-radius: 10px;
  background: var(--aura-accent); color: var(--aura-accent-text);
  display: grid; place-items: center;
  font-weight: 800; font-size: 16.8px; letter-spacing: -0.02em;
  box-shadow: 0 2px 8px var(--aura-accent-ring);
  flex-shrink: 0;
}
.brand-text { min-width: 0; }
.brand-name { font-size: 14.7px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); line-height: 1; }
.brand-meta { font-size: 11.6px; color: var(--aura-text-faint); margin-top: 2px; }
.icon-btn {
  width: 32px; height: 32px; border-radius: 8px;
  border: 1px solid transparent;
  background: transparent; color: var(--aura-text-muted);
  display: grid; place-items: center;
  cursor: pointer; transition: all 0.15s;
}
.icon-btn:hover { background: var(--aura-bg-subtle); border-color: var(--aura-border); color: var(--aura-text); }
.collapse-btn { width: 28px; height: 28px; }
.chev { font-size: 18px; line-height: 1; transition: transform 0.2s; display: block; }
.chev.open { transform: rotate(0deg); }
.collapsed .chev { transform: rotate(180deg); }
.hamburger { display: flex; flex-direction: column; gap: 3px; }
.hamburger span { display: block; width: 14px; height: 2px; border-radius: 999px; background: currentColor; }
.sidebar-scroll {
  flex: 1; min-height: 0;
  overflow-y: auto; overflow-x: hidden;
  padding: 12px 10px 12px;
  display: flex; flex-direction: column; gap: 12px;
}
.nav-section { display: flex; flex-direction: column; gap: 2px; }
.nav-section-head {
  display: flex; align-items: center; justify-content: space-between;
  width: 100%;
  padding: 6px 8px;
  background: transparent; border: none;
  cursor: pointer;
  border-radius: 8px;
  transition: background 0.15s;
}
.nav-section-head:hover { background: var(--aura-bg-subtle); }
.nav-label {
  font-size: 10.5px; font-weight: 650; letter-spacing: 0.06em; text-transform: uppercase;
  color: var(--aura-text-faint);
}
.section-chevron {
  font-size: 12.6px; color: var(--aura-text-faint);
  transition: transform 0.15s;
  line-height: 1;
}
.section-chevron.open { transform: rotate(90deg); }
.nav-items { display: flex; flex-direction: column; gap: 2px; }
.nav-group a, .nav-items a {
  display: flex; align-items: center; gap: 10px;
  padding: 7px 10px;
  border-radius: 10px;
  text-decoration: none;
  color: #334155;
  border: 1px solid transparent;
  transition: all 0.15s;
  cursor: pointer;
  position: relative;
}
[data-theme="dark"] .nav-items a { color: #e2e8f0; }
.nav-items a:hover { background: var(--aura-bg-subtle); color: var(--aura-text); border-color: var(--aura-border); }
.nav-items a.active {
  background: var(--aura-accent-soft);
  color: var(--aura-text);
  border-color: color-mix(in srgb, var(--aura-accent) 18%, transparent);
}
[data-theme="dark"] .nav-items a.active { background: #134e4a; color: var(--aura-text); }
.nav-icon {
  width: 16.8px; height: 16.8px;
  display: grid; place-items: center;
  flex-shrink: 0;
  color: var(--aura-text-muted);
}
.nav-icon svg { width: 14.7px; height: 14.7px; }
.nav-items a.active .nav-icon { color: var(--aura-accent); }
.nav-icon.admin { color: var(--aura-warning); }
.nav-main { display: flex; flex-direction: column; min-width: 0; }
.nav-label-text { font-size: 13.7px; font-weight: 550; line-height: 1.1; white-space: nowrap; }
.nav-desc { font-size: 11.6px; color: var(--aura-text-faint); line-height: 1; margin-top: 2px; }
.aura-layout.collapsed .nav-desc,
.aura-layout.collapsed .brand-text,
.aura-layout.collapsed .nav-label,
.aura-layout.collapsed .nav-section-head .nav-label { display: none; }
.aura-layout.collapsed .nav-items a { justify-content: center; padding: 10px 6px; }
.aura-layout.collapsed .nav-section-head { justify-content: center; }
.sidebar-foot {
  padding: 12px;
  border-top: 1px solid var(--aura-border);
  flex-shrink: 0;
}
.foot-card {
  background: var(--aura-bg-subtle);
  border: 1px solid var(--aura-border);
  border-radius: 10px;
  padding: 10px 12px;
  text-align: center;
}
.foot-title { font-size: 11px; font-weight: 700; color: var(--aura-text); }
.foot-sub { font-size: 10px; color: var(--aura-text-faint); margin-top: 2px; }
.foot-dot { width: 8px; height: 8px; border-radius: 999px; background: var(--aura-accent); margin: 0 auto; box-shadow: 0 0 0 4px var(--aura-accent-ring); }
.aura-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.aura-header {
  height: var(--aura-header-h);
  background: color-mix(in srgb, var(--aura-surface) 92%, transparent);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--aura-border);
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 16px;
  flex-shrink: 0;
  position: sticky; top: 0; z-index: 5;
}
.header-left { display: flex; align-items: center; gap: 12px; min-width: 0; }
.header-title { min-width: 0; }
.crumb { font-size: 14px; font-weight: 650; color: var(--aura-text); letter-spacing: -0.01em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.header-right { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.header-action { padding: 8px 12px; font-size: 12px; }
.theme-btn {
  width: 36px; height: 36px; border-radius: 10px;
  border: 1px solid var(--aura-border);
  background: var(--aura-surface);
  display: grid; place-items: center;
  cursor: pointer; font-size: 14px;
  transition: all 0.15s;
}
.theme-btn:hover { background: var(--aura-bg-subtle); }
.user-chip {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 10px 6px 6px;
  border-radius: 999px;
  background: var(--aura-bg-subtle);
  border: 1px solid var(--aura-border);
}
.user-avatar {
  width: 28px; height: 28px; border-radius: 999px;
  background: var(--aura-accent); color: var(--aura-accent-text);
  display: grid; place-items: center;
  font-size: 11px; font-weight: 700;
}
.user-name { font-size: 13px; font-weight: 600; color: var(--aura-text); max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.user-role { font-size: 10px; font-weight: 650; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); padding: 2px 6px; border-radius: 999px; background: var(--aura-surface); border: 1px solid var(--aura-border); }
.aura-content {
  flex: 1; min-height: 0;
  overflow-y: auto;
  padding: 24px;
  background: var(--aura-bg);
}
.content-inner { width: 100%; max-width: 1280px; margin: 0 auto; }
.active-bar {
  position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
  width: 3px; height: 16px; border-radius: 999px;
  background: var(--aura-accent);
}
@media (max-width: 768px) {
  .aura-sidebar { position: fixed; inset: 0 auto 0 0; transform: translateX(0); z-index: 20; }
  .aura-layout.collapsed .aura-sidebar { transform: translateX(-100%); width: var(--aura-sidebar-w); }
  .header-right .header-action { display: none; }
}
</style>
