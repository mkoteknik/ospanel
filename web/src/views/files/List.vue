<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface FileItem {
  name: string
  path: string
  type: 'file' | 'dir'
  size: number
  mode: string
  modified: string
}

const currentPath = ref('/')
const files = ref<FileItem[]>([])
const breadcrumbs = ref<{ name: string; path: string }[]>([])
const loading = ref(false)
const error = ref('')
const showCreateDir = ref(false)
const showEditor = ref(false)
const editingFile = ref({ path: '', content: '', name: '' })
const newDirName = ref('')
const selectedFiles = ref<Set<string>>(new Set())

async function loadDir(path: string) {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/api/v1/files', { params: { path } })
    files.value = res.data.files || []
    breadcrumbs.value = res.data.breadcrumbs || []
    currentPath.value = res.data.path
  } catch {
    error.value = 'Dizin yüklenemedi'
    files.value = []
  } finally {
    loading.value = false
  }
}

function navigate(path: string) { loadDir(path) }

function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatDate(d: string): string {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('tr-TR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

async function openFile(file: FileItem) {
  try {
    const res = await api.post('/api/v1/files/read', { path: file.path })
    editingFile.value = {
      path: file.path,
      content: res.data.content,
      name: file.name,
    }
    showEditor.value = true
  } catch {
    error.value = 'Dosya okunamadı'
  }
}

async function saveFile() {
  try {
    await api.post('/api/v1/files/write', {
      path: editingFile.value.path,
      content: editingFile.value.content,
    })
    showEditor.value = false
    loadDir(currentPath.value)
  } catch {
    error.value = 'Dosya kaydedilemedi'
  }
}

async function deleteFile(file: FileItem) {
  const msg = file.type === 'dir'
    ? `"${file.name}" dizinini ve içindekileri silmek istediğinize emin misiniz?`
    : `"${file.name}" dosyasını silmek istediğinize emin misiniz?`
  if (!confirm(msg)) return
  try {
    await api.delete('/api/v1/files', { params: { path: file.path } })
    loadDir(currentPath.value)
  } catch {
    error.value = 'Silinemedi'
  }
}

async function createDir() {
  if (!newDirName.value) return
  try {
    await api.post('/api/v1/files/mkdir', { parent: currentPath.value, name: newDirName.value })
    showCreateDir.value = false
    newDirName.value = ''
    loadDir(currentPath.value)
  } catch {
    error.value = 'Dizin oluşturulamadı'
  }
}

async function uploadFile(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  const form = new FormData()
  form.append('file', input.files[0])
  form.append('dir', currentPath.value)
  try {
    await api.post('/api/v1/files/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    loadDir(currentPath.value)
  } catch {
    error.value = 'Dosya yüklenemedi'
  }
  input.value = ''
}

function getFileIcon(file: FileItem): string {
  if (file.type === 'dir') return '📁'
  const ext = file.name.split('.').pop()?.toLowerCase()
  const icons: Record<string, string> = {
    html: '🌐', htm: '🌐', css: '🎨', js: '📜', ts: '📘',
    json: '📋', md: '📝', txt: '📄', log: '📋',
    php: '🐘', py: '🐍', go: '🔵', rs: '🦀', java: '☕',
    png: '🖼️', jpg: '🖼️', jpeg: '🖼️', gif: '🖼️', svg: '🖼️', ico: '🖼️',
    zip: '📦', tar: '📦', gz: '📦', rar: '📦',
    pdf: '📕', doc: '📘', xls: '📊', ppt: '📊',
    sql: '🗄️', yml: '⚙️', yaml: '⚙️', xml: '⚙️', conf: '⚙️', ini: '⚙️',
  }
  return icons[ext || ''] || '📄'
}

onMounted(() => loadDir(currentPath.value))
</script>

<template>
  <div class="file-manager">
    <div class="fm-header">
      <h2>📁 Dosya Yöneticisi</h2>
      <div class="fm-actions">
        <label class="btn-upload">
          📤 Yükle
          <input type="file" hidden @change="uploadFile" />
        </label>
        <button class="btn-action" @click="showCreateDir = true">📂 Klasör</button>
        <button class="btn-action" @click="loadDir(currentPath)">🔄 Yenile</button>
      </div>
    </div>

    <!-- Breadcrumbs -->
    <div class="breadcrumbs">
      <span v-for="(crumb, i) in breadcrumbs" :key="crumb.path">
        <span v-if="i > 0" class="crumb-sep">›</span>
        <a href="#" @click.prevent="navigate(crumb.path)" class="crumb-link">
          {{ crumb.name }}
        </a>
      </span>
    </div>

    <div class="path-bar">📂 {{ currentPath }}</div>

    <!-- Hata -->
    <div v-if="error" class="fm-error">{{ error }}</div>

    <!-- Loading -->
    <div v-if="loading" class="fm-loading">Yükleniyor...</div>

    <!-- Dosya tablosu -->
    <div v-if="!loading" class="file-table">
      <div class="file-row header">
        <span class="col-name">Ad</span>
        <span class="col-size">Boyut</span>
        <span class="col-date">Değiştirme</span>
        <span class="col-actions"></span>
      </div>

      <div v-if="files.length === 0" class="file-row empty">
        <span>Bu dizin boş</span>
      </div>

      <div
        v-for="file in files"
        :key="file.path"
        class="file-row"
        :class="{ 'is-dir': file.type === 'dir' }"
      >
        <span class="col-name">
          <span
            class="file-icon"
            @click="file.type === 'dir' ? navigate(file.path) : openFile(file)"
            style="cursor:pointer"
          >
            {{ getFileIcon(file) }}
          </span>
          <span
            class="file-link"
            @click="file.type === 'dir' ? navigate(file.path) : openFile(file)"
          >
            {{ file.name }}
          </span>
        </span>
        <span class="col-size">{{ file.type === 'dir' ? '-' : formatSize(file.size) }}</span>
        <span class="col-date">{{ formatDate(file.modified) }}</span>
        <span class="col-actions">
          <button v-if="file.type === 'file'" class="btn-icon" title="Düzenle" @click="openFile(file)">✏️</button>
          <button class="btn-icon" title="Sil" @click="deleteFile(file)">🗑️</button>
        </span>
      </div>
    </div>

    <!-- Klasör Oluşturma Modal -->
    <div v-if="showCreateDir" class="modal-overlay" @click.self="showCreateDir = false">
      <div class="modal modal-sm">
        <div class="modal-header">
          <h3>📂 Yeni Klasör</h3>
          <button class="modal-close" @click="showCreateDir = false">✕</button>
        </div>
        <div class="modal-body">
          <input
            v-model="newDirName"
            type="text"
            placeholder="Klasör adı"
            @keyup.enter="createDir"
            autofocus
            class="fm-input"
          />
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showCreateDir = false">İptal</button>
          <button class="btn-add" @click="createDir">Oluştur</button>
        </div>
      </div>
    </div>

    <!-- Dosya Editör Modal -->
    <div v-if="showEditor" class="modal-overlay" @click.self="showEditor = false">
      <div class="modal modal-lg">
        <div class="modal-header">
          <h3>✏️ {{ editingFile.name }}</h3>
          <button class="modal-close" @click="showEditor = false">✕</button>
        </div>
        <div class="modal-body">
          <textarea
            v-model="editingFile.content"
            class="code-editor"
            rows="20"
            spellcheck="false"
          ></textarea>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showEditor = false">İptal</button>
          <button class="btn-add" @click="saveFile">💾 Kaydet</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-manager { max-width: 1200px; }

.fm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.fm-header h2 { margin: 0; }

.fm-actions { display: flex; gap: 8px; }

.btn-action, .btn-upload {
  padding: 8px 16px;
  background: white;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-action:hover, .btn-upload:hover { background: #f5f5f5; border-color: #ccc; }

.breadcrumbs {
  padding: 8px 0;
  font-size: 14px;
  margin-bottom: 8px;
}
.crumb-link { color: #0f3460; text-decoration: none; }
.crumb-link:hover { text-decoration: underline; }
.crumb-sep { margin: 0 6px; color: #ccc; }

.path-bar {
  font-size: 12px;
  color: #888;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
  margin-bottom: 12px;
  font-family: monospace;
}

.fm-error { background: #ffe0e0; color: #c0392b; padding: 10px 16px; border-radius: 8px; margin-bottom: 12px; font-size: 14px; }
.fm-loading { text-align: center; padding: 40px; color: #888; }

.file-table {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
  overflow: hidden;
}

.file-row {
  display: grid;
  grid-template-columns: 1fr 100px 160px 80px;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;
  transition: background 0.1s;
}
.file-row:hover { background: #f8f9fa; }
.file-row.header {
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
  color: #888;
  background: #fafafa;
  border-bottom: 2px solid #e0e0e0;
}
.file-row.empty { text-align: center; padding: 40px; color: #888; display: block; }

.file-icon { margin-right: 8px; font-size: 18px; }
.file-link { color: #333; cursor: pointer; }
.file-link:hover { color: #0f3460; text-decoration: underline; }
.is-dir .file-link { font-weight: 600; }

.btn-icon {
  background: none; border: none; font-size: 16px; cursor: pointer;
  padding: 4px; border-radius: 4px;
}
.btn-icon:hover { background: #f0f0f0; }

.fm-input {
  width: 100%;
  padding: 10px 12px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
}
.fm-input:focus { outline: none; border-color: #0f3460; }

.code-editor {
  width: 100%;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  padding: 16px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  background: #1a1a2e;
  color: #e0e0e0;
  resize: vertical;
}
.code-editor:focus { outline: none; border-color: #0f3460; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-sm { max-width: 400px; }
.modal-lg { max-width: 800px; }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-close:hover { color: #333; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; font-size: 14px; cursor: pointer; }
.btn-cancel:hover { background: #e0e0e0; }
.btn-add {
  padding: 10px 20px; background: #0f3460; color: white; border: none;
  border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer;
}
.btn-add:hover { background: #1a4a7a; }
</style>
