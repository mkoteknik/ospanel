<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface FileItem { name: string; path: string; type: 'file' | 'dir'; size: number; mode: string; modified: string }

const currentPath = ref('/')
const files = ref<FileItem[]>([])
const breadcrumbs = ref<{ name: string; path: string }[]>([])
const loading = ref(false)
const error = ref('')

// Modals
const showEditor = ref(false)
const showChmod = ref(false)
const showRename = ref(false)
const showNewFile = ref(false)
const showNewDir = ref(false)
const editingFile = ref({ path: '', content: '', name: '' })
const chmodFile = ref<FileItem | null>(null)
const chmodMode = ref('')
const renameFile = ref<FileItem | null>(null)
const renameName = ref('')
const newFileName = ref('')
const newDirName = ref('')
const currentDir = ref('')

async function loadDir(path: string) {
  loading.value = true; error.value = ''
  try {
    const res = await api.get('/api/v1/files', { params: { path } })
    files.value = res.data.files || []
    breadcrumbs.value = res.data.breadcrumbs || []
    currentPath.value = res.data.path
    currentDir.value = res.data.path
  } catch { error.value = 'Dizin yüklenemedi' }
  finally { loading.value = false }
}

function navigate(path: string) { loadDir(path) }

function formatSize(bytes: number): string {
  if (!bytes || bytes === 0) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1073741824) return (bytes / 1048576).toFixed(1) + ' MB'
  return (bytes / 1073741824).toFixed(2) + ' GB'
}

function formatDate(d: string): string {
  if (!d) return '-'
  return new Date(d).toLocaleString('tr-TR', { day:'2-digit', month:'2-digit', year:'numeric', hour:'2-digit', minute:'2-digit' })
}

function formatPerms(mode: string): string {
  if (!mode || mode.length < 9) return '----'
  const m = mode.slice(-9)
  return (m[0]==='r'?'r':'-')+(m[1]==='w'?'w':'-')+(m[2]==='x'?'x':'-')+
         (m[3]==='r'?'r':'-')+(m[4]==='w'?'w':'-')+(m[5]==='x'?'x':'-')+
         (m[6]==='r'?'r':'-')+(m[7]==='w'?'w':'-')+(m[8]==='x'?'x':'-')
}

function getNumericMode(mode: string): string {
  if (!mode || mode.length < 9) return '0000'
  const m = mode.slice(-9)
  let n = 0
  if (m[0]==='r') n+=4; if (m[1]==='w') n+=2; if (m[2]==='x') n+=1
  let o = n*8; n = 0
  if (m[3]==='r') n+=4; if (m[4]==='w') n+=2; if (m[5]==='x') n+=1
  o = o*8 + n; n = 0
  if (m[6]==='r') n+=4; if (m[7]==='w') n+=2; if (m[8]==='x') n+=1
  o = o*8 + n
  return '0' + o.toString(8)
}

function getFileIcon(file: FileItem): string {
  if (file.type === 'dir') return '📁'
  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  const map: Record<string,string> = {
    html:'🌐', htm:'🌐', css:'🎨', js:'📜', ts:'📘', jsx:'⚛️', tsx:'⚛️',
    json:'📋', md:'📝', txt:'📄', log:'📋', env:'🔧', yml:'⚙️', yaml:'⚙️', xml:'⚙️', ini:'⚙️', cfg:'⚙️',
    php:'🐘', py:'🐍', go:'🔵', rs:'🦀', java:'☕', rb:'💎', c:'⚙️', cpp:'⚙️', h:'⚙️',
    png:'🖼️', jpg:'🖼️', jpeg:'🖼️', gif:'🖼️', svg:'🖼️', ico:'🖼️', webp:'🖼️', bmp:'🖼️',
    zip:'📦', tar:'📦', gz:'📦', rar:'📦', '7z':'📦', bz2:'📦',
    pdf:'📕', doc:'📘', docx:'📘', xls:'📊', xlsx:'📊', ppt:'📊', pptx:'📊',
    sql:'🗄️', db:'🗄️', sqlite:'🗄️',
    mp3:'🎵', mp4:'🎬', avi:'🎬', mkv:'🎬', mov:'🎬', wav:'🎵', flac:'🎵',
    sh:'💻', bash:'💻', ps1:'💻', bat:'💻', exe:'⚙️',
    htaccess:'🔒', htpasswd:'🔒',
    lock:'🔒', gitignore:'🔧',
  }
  return map[ext] || '📄'
}

// Actions
async function openFile(file: FileItem) {
  if (file.type === 'dir') { navigate(file.path); return }
  try {
    const res = await api.post('/api/v1/files/read', { path: file.path })
    editingFile.value = { path: file.path, content: res.data.content, name: file.name }
    showEditor.value = true
  } catch { error.value = 'Dosya okunamadı' }
}

async function saveFile() {
  try {
    await api.post('/api/v1/files/write', { path: editingFile.value.path, content: editingFile.value.content })
    showEditor.value = false; loadDir(currentPath.value)
  } catch { error.value = 'Kaydedilemedi' }
}

function downloadFile(file: FileItem) {
  window.open('/api/v1/files/download?path=' + encodeURIComponent(file.path), '_blank')
}

async function deleteItem(file: FileItem) {
  if (!confirm(`${file.name} silinecek. Emin misiniz?`)) return
  try { await api.delete('/api/v1/files', { params: { path: file.path } }); loadDir(currentPath.value) }
  catch { error.value = 'Silinemedi' }
}

function openChmod(file: FileItem) {
  chmodFile.value = file
  chmodMode.value = getNumericMode(file.mode)
  showChmod.value = true
}

async function doChmod() {
  if (!chmodFile.value) return
  try {
    await api.post('/api/v1/files/chmod', { path: chmodFile.value.path, mode: chmodMode.value })
    showChmod.value = false; loadDir(currentPath.value)
  } catch { error.value = 'Chmod başarısız' }
}

function openRename(file: FileItem) {
  renameFile.value = file
  renameName.value = file.name
  showRename.value = true
}

async function doRename() {
  if (!renameFile.value) return
  try {
    await api.post('/api/v1/files/rename', { path: renameFile.value.path, new_name: renameName.value })
    showRename.value = false; loadDir(currentPath.value)
  } catch { error.value = 'Yeniden adlandırılamadı' }
}

async function createFile() {
  if (!newFileName.value) return
  try { await api.post('/api/v1/files/create', { dir: currentPath.value, name: newFileName.value }); showNewFile.value = false; newFileName.value = ''; loadDir(currentPath.value) }
  catch { error.value = 'Dosya oluşturulamadı' }
}

async function createDir() {
  if (!newDirName.value) return
  try { await api.post('/api/v1/files/mkdir', { parent: currentPath.value, name: newDirName.value }); showNewDir.value = false; newDirName.value = ''; loadDir(currentPath.value) }
  catch { error.value = 'Klasör oluşturulamadı' }
}

async function uploadFile(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  const form = new FormData()
  form.append('file', input.files[0]); form.append('dir', currentPath.value)
  try { await api.post('/api/v1/files/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } }); loadDir(currentPath.value) }
  catch { error.value = 'Yüklenemedi' }
  input.value = ''
}

onMounted(() => loadDir(currentPath.value))
</script>

<template>
  <div class="fm">
    <!-- Header -->
    <div class="fm-header">
      <h2>📁 Dosya Yöneticisi</h2>
      <div class="fm-actions">
        <label class="btn fm-btn">📤 Yükle <input type="file" hidden @change="uploadFile" /></label>
        <button class="btn fm-btn" @click="showNewFile = true">📄 Yeni Dosya</button>
        <button class="btn fm-btn" @click="showNewDir = true">📂 Klasör</button>
        <button class="btn fm-btn" @click="loadDir(currentPath)">🔄</button>
      </div>
    </div>

    <!-- Breadcrumb -->
    <div class="breadcrumb">
      <span class="bc-path">📂 {{ currentPath }}</span>
      <div class="bc-links">
        <a v-for="(c, i) in breadcrumbs" :key="c.path" href="#" @click.prevent="navigate(c.path)" class="bc-link">
          {{ i > 0 ? ' › ' : '' }}{{ c.name }}
        </a>
      </div>
    </div>

    <div v-if="error" class="fm-error">{{ error }}</div>
    <div v-if="loading" class="fm-loading">Yükleniyor...</div>

    <!-- File Table -->
    <div v-if="!loading" class="file-table-wrap">
      <div class="file-table">
        <div class="ft-header">
          <span class="col-name">Ad</span>
          <span class="col-size">Boyut</span>
          <span class="col-perm">İzinler</span>
          <span class="col-date">Değiştirme</span>
          <span class="col-act">İşlem</span>
        </div>

        <div v-if="files.length === 0" class="ft-empty">Dizin boş</div>

        <div v-for="f in files" :key="f.path" class="ft-row">
          <span class="col-name">
            <span class="fi-icon" @click="openFile(f)">{{ getFileIcon(f) }}</span>
            <span class="fi-name" @click="openFile(f)">{{ f.name }}</span>
          </span>
          <span class="col-size">{{ f.type === 'dir' ? '-' : formatSize(f.size) }}</span>
          <span class="col-perm">
            <code class="perm-code">{{ formatPerms(f.mode) }}</code>
          </span>
          <span class="col-date">{{ formatDate(f.modified) }}</span>
          <span class="col-act">
            <button title="İndir" @click="downloadFile(f)" :disabled="f.type==='dir'">⬇</button>
            <button title="İzinler" @click="openChmod(f)">🔐</button>
            <button title="Ad Değiştir" @click="openRename(f)">✏️</button>
            <button title="Sil" @click="deleteItem(f)">🗑️</button>
          </span>
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showEditor" class="modal-overlay" @click.self="showEditor=false">
      <div class="modal modal-xl">
        <div class="modal-header"><h3>✏️ {{ editingFile.name }}</h3><button class="modal-close" @click="showEditor=false">✕</button></div>
        <div class="modal-body"><textarea v-model="editingFile.content" class="code-editor" rows="24" spellcheck="false"></textarea></div>
        <div class="modal-footer"><button class="btn-cancel" @click="showEditor=false">İptal</button><button class="btn-primary" @click="saveFile">💾 Kaydet</button></div>
      </div>
    </div>

    <!-- Chmod Modal -->
    <div v-if="showChmod && chmodFile" class="modal-overlay" @click.self="showChmod=false">
      <div class="modal modal-sm">
        <div class="modal-header"><h3>🔐 İzinler: {{ chmodFile.name }}</h3><button class="modal-close" @click="showChmod=false">✕</button></div>
        <div class="modal-body">
          <p style="font-size:13px;color:#888;margin-bottom:12px">Sekizlik (octal) formatta girin: 0755, 0644, 0600 gibi</p>
          <input v-model="chmodMode" class="fm-input" placeholder="0755" />
          <div style="margin-top:8px;font-size:13px;color:#666">Mevcut: <code>{{ formatPerms(chmodFile.mode) }}</code></div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showChmod=false">İptal</button><button class="btn-primary" @click="doChmod">Uygula</button></div>
      </div>
    </div>

    <!-- Rename Modal -->
    <div v-if="showRename && renameFile" class="modal-overlay" @click.self="showRename=false">
      <div class="modal modal-sm">
        <div class="modal-header"><h3>✏️ Yeniden Adlandır</h3><button class="modal-close" @click="showRename=false">✕</button></div>
        <div class="modal-body"><input v-model="renameName" class="fm-input" @keyup.enter="doRename" autofocus /></div>
        <div class="modal-footer"><button class="btn-cancel" @click="showRename=false">İptal</button><button class="btn-primary" @click="doRename">Kaydet</button></div>
      </div>
    </div>

    <!-- New File Modal -->
    <div v-if="showNewFile" class="modal-overlay" @click.self="showNewFile=false">
      <div class="modal modal-sm">
        <div class="modal-header"><h3>📄 Yeni Dosya</h3><button class="modal-close" @click="showNewFile=false">✕</button></div>
        <div class="modal-body"><input v-model="newFileName" class="fm-input" placeholder="dosya.txt" @keyup.enter="createFile" autofocus /></div>
        <div class="modal-footer"><button class="btn-cancel" @click="showNewFile=false">İptal</button><button class="btn-primary" @click="createFile">Oluştur</button></div>
      </div>
    </div>

    <!-- New Dir Modal -->
    <div v-if="showNewDir" class="modal-overlay" @click.self="showNewDir=false">
      <div class="modal modal-sm">
        <div class="modal-header"><h3>📂 Yeni Klasör</h3><button class="modal-close" @click="showNewDir=false">✕</button></div>
        <div class="modal-body"><input v-model="newDirName" class="fm-input" placeholder="yeni-klasor" @keyup.enter="createDir" autofocus /></div>
        <div class="modal-footer"><button class="btn-cancel" @click="showNewDir=false">İptal</button><button class="btn-primary" @click="createDir">Oluştur</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fm { width: 100%; }
.fm-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.fm-header h2 { margin: 0; font-size: 20px; }
.fm-actions { display: flex; gap: 6px; }
.btn { padding: 8px 14px; border-radius: 8px; font-size: 13px; cursor: pointer; }
.fm-btn { background: white; border: 1px solid #ddd; color: #333; }
.fm-btn:hover { background: #f5f5f5; }

.breadcrumb { margin-bottom: 16px; }
.bc-path { font-size: 12px; color: #888; display: block; font-family: monospace; margin-bottom: 6px; }
.bc-link { color: #0f3460; text-decoration: none; font-size: 14px; }
.bc-link:hover { text-decoration: underline; }

.fm-error { background: #ffe0e0; color: #c0392b; padding: 10px 16px; border-radius: 8px; margin-bottom: 12px; font-size: 13px; }
.fm-loading { text-align: center; padding: 40px; color: #888; }

.file-table-wrap { background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0; overflow: hidden; }
.file-table { width: 100%; }
.ft-header { display: grid; grid-template-columns: 1fr 90px 110px 140px 120px; padding: 12px 16px; background: #fafafa; border-bottom: 2px solid #e5e5e5; font-size: 11px; font-weight: 700; color: #888; text-transform: uppercase; }
.ft-row { display: grid; grid-template-columns: 1fr 90px 110px 140px 120px; padding: 10px 16px; border-bottom: 1px solid #f5f5f5; font-size: 13px; align-items: center; transition: background 0.1s; }
.ft-row:hover { background: #f8f9fa; }
.ft-empty { text-align: center; padding: 40px; color: #888; }

.col-name { display: flex; align-items: center; gap: 10px; overflow: hidden; }
.fi-icon { font-size: 18px; cursor: pointer; flex-shrink: 0; }
.fi-name { cursor: pointer; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #333; }
.fi-name:hover { color: #0f3460; text-decoration: underline; }

.col-size { color: #666; font-family: monospace; font-size: 12px; }
.col-perm { font-family: monospace; }
.perm-code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; font-size: 12px; color: #555; }
.col-date { color: #888; font-size: 12px; }
.col-act { display: flex; gap: 4px; }
.col-act button { background: none; border: 1px solid #e0e0e0; border-radius: 4px; padding: 4px 6px; font-size: 13px; cursor: pointer; }
.col-act button:hover { background: #f0f0f0; }
.col-act button:disabled { opacity: 0.3; cursor:not-allowed; }

.modal-overlay { position: fixed; inset:0; background:rgba(0,0,0,0.5); display:flex; align-items:center; justify-content:center; z-index:1000; }
.modal { background:white; border-radius:12px; width:90%; box-shadow:0 20px 60px rgba(0,0,0,0.3); }
.modal-sm { max-width:420px; }
.modal-xl { max-width:900px; }
.modal-header { display:flex; justify-content:space-between; align-items:center; padding:18px 24px; border-bottom:1px solid #f0f0f0; }
.modal-header h3 { margin:0; }
.modal-close { background:none; border:none; font-size:20px; cursor:pointer; color:#888; }
.modal-body { padding:24px; }
.modal-footer { display:flex; justify-content:flex-end; gap:8px; padding:16px 24px; border-top:1px solid #f0f0f0; }
.btn-cancel { padding:10px 20px; background:#f0f0f0; border:none; border-radius:8px; font-size:14px; cursor:pointer; }
.btn-primary { padding:10px 20px; background:#0f3460; color:white; border:none; border-radius:8px; font-size:14px; font-weight:600; cursor:pointer; }
.btn-primary:hover { background:#1a4a7a; }

.fm-input { width:100%; padding:10px 12px; border:2px solid #e0e0e0; border-radius:8px; font-size:14px; }
.fm-input:focus { outline:none; border-color:#0f3460; }

.code-editor { width:100%; font-family:'Consolas','Monaco',monospace; font-size:13px; line-height:1.5; padding:16px; border:2px solid #e0e0e0; border-radius:8px; background:#1a1a2e; color:#e0e0e0; resize:vertical; }
.code-editor:focus { outline:none; border-color:#0f3460; }
</style>
