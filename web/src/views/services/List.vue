<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

interface Service {
  name: string; display: string; icon: string; category: string
  systemd: string; installed: boolean; active: boolean; enabled: boolean
  install_cmd: string
}

const services = ref<Service[]>([])
const loading = ref(true)
const actionLoading = ref<Record<string, boolean>>({})

const catNames = computed<Record<string, { name: string; icon: string }>>(() => ({
  web: { name: t('services.webServer'), icon: '🌐' },
  database: { name: t('services.database'), icon: '🗄️' },
  cache: { name: t('services.cache'), icon: '⚡' },
  email: { name: t('services.email'), icon: '📧' },
  dns: { name: t('services.dns'), icon: '🔧' },
  security: { name: t('services.security'), icon: '🛡️' },
  container: { name: t('services.container'), icon: '🐳' },
}))

const categories = computed(() => Object.keys(catNames.value))

async function loadServices() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/services')
    services.value = res.data.services || []
  } catch { }
  finally { loading.value = false }
}

async function doAction(svc: Service, action: string) {
  const key = svc.name + '-' + action
  actionLoading.value[key] = true
  try {
    await api.post('/api/v1/services/action', { service: svc.name, action })
    await loadServices()
  } catch { }
  finally { delete actionLoading.value[key] }
}

function isLoading(svc: Service, action: string): boolean {
  return !!actionLoading.value[svc.name + '-' + action]
}

onMounted(loadServices)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>{{ t('services.title') }}</h2>
        <p>{{ t('services.subtitle') }}</p>
      </div>
      <button class="btn-refresh" @click="loadServices">🔄 {{ t('services.refresh') }}</button>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>{{ t('services.checking') }}</p>
    </div>

    <div v-else v-for="cat in categories" :key="cat" class="category-section">
      <div class="category-header">
        <span class="cat-icon">{{ catNames[cat].icon }}</span>
        <span class="cat-name">{{ catNames[cat].name }}</span>
        <span class="cat-count">{{ services.filter(s => s.category === cat).filter(s => s.installed).length }}/{{ services.filter(s => s.category === cat).length }} {{ t('services.activeCount') }}</span>
      </div>

      <div class="service-grid">
        <div
          v-for="s in services.filter(s => s.category === cat)"
          :key="s.name"
          class="service-card"
          :class="{ installed: s.installed, active: s.active }"
        >
          <!-- SOL: Icon + Info -->
          <div class="svc-left">
            <div class="svc-icon-wrap" :class="{ pulse: s.active }">
              <span class="svc-icon">{{ s.icon }}</span>
              <span v-if="s.active" class="active-dot"></span>
            </div>
            <div class="svc-info">
              <span class="svc-name">{{ s.display }}</span>
              <span class="svc-systemd">{{ s.systemd }}</span>
            </div>
          </div>

          <!-- ORTA: Status -->
          <div class="svc-status-area">
            <template v-if="!s.installed">
              <span class="status-tag not-installed">{{ t('services.notInstalled') }}</span>
            </template>
            <template v-else>
              <span v-if="s.active" class="status-tag running">{{ t('cache.running') }}</span>
              <span v-else class="status-tag stopped">{{ t('cache.stopped') }}</span>
            </template>
          </div>

          <!-- SAĞ: Actions -->
          <div class="svc-actions">
            <!-- Kurulu değilse: Sadece Kur butonu -->
            <template v-if="!s.installed">
              <button
                class="btn-install"
                :disabled="isLoading(s, 'install')"
                @click="doAction(s, 'install')"
              >
                <span v-if="isLoading(s, 'install')" class="btn-spinner"></span>
                {{ isLoading(s, 'install') ? t('cache.installing') : '📥 ' + t('services.install') }}
              </button>
            </template>

            <!-- Kuruluysa: Toggle + Diğer -->
            <template v-else>
              <!-- Ana Aç/Kapa Toggle -->
              <label class="toggle-switch" :class="{ loading: isLoading(s, s.active ? 'stop' : 'start') }">
                <input
                  type="checkbox"
                  :checked="s.active"
                  :disabled="isLoading(s, s.active ? 'stop' : 'start')"
                  @change="doAction(s, s.active ? 'stop' : 'start')"
                />
                <span class="toggle-slider"></span>
              </label>

              <!-- Restart -->
              <button
                class="btn-icon"
                title="Yeniden Başlat"
                :disabled="isLoading(s, 'restart') || !s.active"
                @click="doAction(s, 'restart')"
              >
                🔄
              </button>

              <!-- Boot auto-start -->
              <button
                v-if="s.enabled"
                class="btn-icon btn-boot-on"
                title="Boot'ta otomatik başlar"
                @click="doAction(s, 'disable')"
              >
                🟢
              </button>
              <button
                v-else
                class="btn-icon btn-boot-off"
                title="Boot'ta başlamaz"
                @click="doAction(s, 'enable')"
              >
                ⏻
              </button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 28px; }
.page-header h2 { margin: 0; font-size: 22px; color: #1a1a2e; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 13px; }
.btn-refresh { padding: 8px 18px; background: white; border: 1px solid #e0e0e0; border-radius: 8px; font-size: 13px; cursor: pointer; color: #555; transition: all 0.2s; }
.btn-refresh:hover { background: #f5f5f5; border-color: #ccc; }

/* Loading */
.loading-state { text-align: center; padding: 80px 0; }
.spinner { width: 36px; height: 36px; border: 3px solid #e0e0e0; border-top-color: #0f3460; border-radius: 50%; animation: spin 0.8s linear infinite; margin: 0 auto 16px; }
@keyframes spin { to { transform: rotate(360deg); } }
.loading-state p { color: #888; font-size: 14px; }

/* Category */
.category-section { margin-bottom: 32px; }
.category-header { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; padding-bottom: 10px; border-bottom: 2px solid #f0f0f0; }
.cat-icon { font-size: 18px; }
.cat-name { font-weight: 700; font-size: 14px; color: #1a1a2e; text-transform: uppercase; letter-spacing: 0.5px; }
.cat-count { font-size: 12px; color: #888; margin-left: auto; }

/* Grid */
.service-grid { display: flex; flex-direction: column; gap: 8px; }

/* Card */
.service-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 20px;
  background: white;
  border-radius: 10px;
  border: 1px solid #f0f0f0;
  transition: all 0.2s;
}
.service-card:hover { border-color: #e0e0e0; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
.service-card.installed { border-left: 3px solid #e0e0e0; }
.service-card.active { border-left-color: #27ae60; background: #fafffe; }

/* Sol */
.svc-left { display: flex; align-items: center; gap: 14px; min-width: 220px; }
.svc-icon-wrap { position: relative; width: 42px; height: 42px; display: flex; align-items: center; justify-content: center; background: #f5f6fa; border-radius: 10px; }
.svc-icon { font-size: 22px; }
.active-dot { position: absolute; bottom: -2px; right: -2px; width: 10px; height: 10px; background: #27ae60; border: 2px solid white; border-radius: 50%; }
.pulse .active-dot { animation: pulse-dot 2s infinite; }
@keyframes pulse-dot { 0%, 100% { box-shadow: 0 0 0 0 rgba(39,174,96,0.4); } 50% { box-shadow: 0 0 0 6px rgba(39,174,96,0); } }
.svc-info { display: flex; flex-direction: column; }
.svc-name { font-weight: 600; font-size: 14px; color: #1a1a2e; }
.svc-systemd { font-size: 11px; color: #888; font-family: 'SF Mono', 'Consolas', monospace; }

/* Status */
.svc-status-area { flex: 1; display: flex; align-items: center; }
.status-tag { padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.status-tag.not-installed { background: #f0f0f0; color: #888; }
.status-tag.running { background: #d4edda; color: #155724; }
.status-tag.stopped { background: #f8d7da; color: #721c24; }

/* Actions */
.svc-actions { display: flex; align-items: center; gap: 8px; }

/* Install Button */
.btn-install {
  display: flex; align-items: center; gap: 6px;
  padding: 9px 18px;
  background: linear-gradient(135deg, #0f3460, #1a4a7a);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 13px; font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-install:hover:not(:disabled) { background: linear-gradient(135deg, #1a4a7a, #2563a0); transform: translateY(-1px); box-shadow: 0 4px 12px rgba(15,52,96,0.3); }
.btn-install:disabled { opacity: 0.7; cursor: not-allowed; }
.btn-spinner { width: 14px; height: 14px; border: 2px solid rgba(255,255,255,0.3); border-top-color: white; border-radius: 50%; animation: spin 0.6s linear infinite; }

/* Toggle Switch */
.toggle-switch { position: relative; display: inline-block; width: 48px; height: 26px; cursor: pointer; }
.toggle-switch input { opacity: 0; width: 0; height: 0; }
.toggle-slider { position: absolute; inset: 0; background: #ccc; border-radius: 26px; transition: 0.3s; }
.toggle-slider::before { content: ''; position: absolute; height: 20px; width: 20px; left: 3px; bottom: 3px; background: white; border-radius: 50%; transition: 0.3s; box-shadow: 0 1px 3px rgba(0,0,0,0.2); }
.toggle-switch input:checked + .toggle-slider { background: #27ae60; }
.toggle-switch input:checked + .toggle-slider::before { transform: translateX(22px); }
.toggle-switch.loading { opacity: 0.6; pointer-events: none; }

/* Icon Buttons */
.btn-icon { width: 34px; height: 34px; display: flex; align-items: center; justify-content: center; border: 1px solid #e0e0e0; border-radius: 8px; background: white; cursor: pointer; font-size: 14px; transition: all 0.15s; }
.btn-icon:hover:not(:disabled) { background: #f5f5f5; border-color: #ccc; }
.btn-icon:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-boot-on { border-color: #c3e6cb; background: #f0fff4; }
.btn-boot-off { border-color: #e0e0e0; background: #fafafa; color: #aaa; }
</style>
