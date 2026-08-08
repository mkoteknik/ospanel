<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'

const route = useRoute()
const { t } = useI18n()

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
const previewContent = ref('')
const previewLoading = ref(false)
let previewTimer: number | null = null
const chmodFile = ref<FileItem | null>(null)
const chmodMode = ref('')
const renameFile = ref<FileItem | null>(null)
const renameName = ref('')
const newFileName = ref('')
const newDirName = ref('')
const currentDir = ref('')

function previewFile(file: FileItem) {
  if (file.type === 'dir') return
  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  if (['zip','tar','gz','png','jpg','jpeg','gif','mp4','pdf'].includes(ext)) return
  if (previewTimer) clearTimeout(previewTimer)
  previewLoading.value = true
  previewTimer = window.setTimeout(async () => {
    try {
      const r = await api.post('/api/v1/files/read', { path: file.path })
      previewContent.value = (r.data.content || '').slice(0, 4000)
    } catch { previewContent.value = t('files.previewFailed') }
    finally { previewLoading.value = false }
  }, 100)
}

async function loadDir(path: string) {
  loading.value = true; error.value = ''
  try {
    const res = await api.get('/api/v1/files', { params: { path } })
    files.value = res.data.files || []
    breadcrumbs.value = res.data.breadcrumbs || []
    currentPath.value = res.data.path
    currentDir.value = res.data.path
  } catch { error.value = t('files.loadFailed') }
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
  if (file.type === 'dir') return '▣'
  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  const map: Record<string,string> = {
    html:'◎', htm:'◎', css:'◈', js:'›', ts:'›', jsx:'›', tsx:'›',
    json:'≡', md:'≡', txt:'≡', log:'≡', env:'·', yml:'·', yaml:'·', xml:'·', ini:'·', cfg:'·',
    php:'◐', py:'◐', go:'◐', rs:'◐', java:'◐', rb:'◐', c:'◐', cpp:'◐', h:'◐',
    png:'▣', jpg:'▣', jpeg:'▣', gif:'▣', svg:'▣', ico:'▣', webp:'▣', bmp:'▣',
    zip:'□', tar:'□', gz:'□', rar:'□', '7z':'□', bz2:'□',
    pdf:'▭', doc:'▭', docx:'▭', xls:'▭', xlsx:'▭', ppt:'▭', pptx:'▭',
    sql:'◧', db:'◧', sqlite:'◧',
    mp3:'♪', mp4:'▶', avi:'▶', mkv:'▶', mov:'▶', wav:'♪', flac:'♪',
    sh:'›', bash:'›', ps1:'›', bat:'›', exe:'·',
    htaccess:'·', htpasswd:'·',
    lock:'·', gitignore:'·',
  }
  return map[ext] || '≡'
}

// Actions
async function openFile(file: FileItem) {
  if (file.type === 'dir') { navigate(file.path); return }
  try {
    const res = await api.post('/api/v1/files/read', { path: file.path })
    editingFile.value = { path: file.path, content: res.data.content, name: file.name }
    showEditor.value = true
  } catch { error.value = t('files.readFailed') }
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
  catch (e: any) { error.value = e?.response?.data?.error || e?.message || 'Silinemedi' }
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
  } catch { error.value = t('files.chmodFailed') }
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
  } catch { error.value = t('files.renameFailed') }
}

async function createFile() {
  if (!newFileName.value) return
  try { await api.post('/api/v1/files/create', { dir: currentPath.value, name: newFileName.value }); showNewFile.value = false; newFileName.value = ''; loadDir(currentPath.value) }
  catch { error.value = t('files.createFileFailed') }
}

async function createDir() {
  if (!newDirName.value) return
  try { await api.post('/api/v1/files/mkdir', { parent: currentPath.value, name: newDirName.value }); showNewDir.value = false; newDirName.value = ''; loadDir(currentPath.value) }
  catch { error.value = t('files.createDirFailed') }
}

async function uploadFile(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  const form = new FormData()
  form.append('file', input.files[0]); form.append('dir', currentPath.value)
  try { await api.post('/api/v1/files/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } }); loadDir(currentPath.value) }
  catch { error.value = t('files.loadFailed') }
  input.value = ''
}

onMounted(() => {
  const q = route.query.path as string | undefined
  loadDir(q && q.trim() ? q : currentPath.value)
})
watch(() => route.query.path, (p) => { if (typeof p === 'string' && p) loadDir(p) })
</script>

<template>
  <div class="fm-page">
    <div class="page-head">
      <div>
        <h2>{{ t('files.title') }}</h2>
        <p>{{ t('files.subtitle') }}</p>
      </div>
      <div class="head-actions">
        <label class="aura-btn aura-btn-ghost">{ t('files.upload') }<input type="file" hidden @change="uploadFile" /></label>
        <button class="aura-btn aura-btn-ghost" @click="showNewFile = true">{{ t('files.newFile') }}</button>
        <button class="aura-btn aura-btn-primary" @click="showNewDir = true">{{ t('files.newFolder') }}</button>
        <button class="aura-btn aura-btn-ghost icon-only" :title="t('common.refresh')" @click="loadDir(currentPath)">↻</button>
      </div>
    </div>

    <!-- Breadcrumb -->
    <div class="aura-card breadcrumb">
      <span class="kicker">{{ t('files.location') }}</span>
      <code class="bc-path">{{ currentPath }}</code>
      <div v-if="breadcrumbs.length" class="bc-links">
        <template v-for="(c, i) in breadcrumbs" :key="c.path">
          <span v-if="i > 0" class="sep">/</span>
          <a href="#" class="bc-link" @click.prevent="navigate(c.path)">{{ c.name || '/' }}</a>
        </template>
      </div>
    </div>

    <div v-if="error" class="alert">{{ error }}</div>
    <div v-if="loading" class="state muted">{{ t('common.loading') }}</div>

    <!-- File table -->
    <div v-if="!loading" class="aura-card table-wrap">
      <div class="ft-header">
        <span class="kicker">{{ t('common.name') }}</span>
        <span class="kicker">Boyut</span>
        <span class="kicker">{{ t('auto.bb1f87') }}</span>
        <span class="kicker">{{ t('auto.6e488e') }}</span>
        <span class="kicker" style="text-align:right">{{ t('common.operation') }}</span>
      </div>
      <div v-if="files.length === 0" class="ft-empty">{{ t('files.empty') }}</div>
      <div v-for="f in files" :key="f.path" class="ft-row">
        <span class="col-name" @click="openFile(f)">
          <span class="fi-icon" :class="{ dir: f.type==='dir' }">{{ getFileIcon(f) }}</span>
          <span class="fi-name" :class="{ dir: f.type==='dir' }">{{ f.name }}</span>
        </span>
        <span class="col-size">{{ f.type === 'dir' ? '—' : formatSize(f.size) }}</span>
        <span class="col-perm"><code class="perm">{{ formatPerms(f.mode) }}</code></span>
        <span class="col-date">{{ formatDate(f.modified) }}</span>
        <span class="col-act">
          <button class="icon-btn" :title="t('files.download')" :disabled="f.type==='dir'" @click="downloadFile(f)">↓</button>
          <button class="icon-btn" :title="t('files.permissions')" @click="openChmod(f)">◐</button>
          <button class="icon-btn" :title="t('files.rename')" @click="openRename(f)">✎</button>
          <button class="icon-btn danger" :title="t('common.delete')" @click="deleteItem(f)">×</button>
        </span>
      </div>
    </div>

    <!-- Edit modal -->
    <div v-if="showEditor" class="overlay" @click.self="showEditor = false">
      <div class="aura-card modal modal-xl">
        <div class="modal-head">
          <div><span class="kicker">{{ t('common.edit') }}</span><h3 class="modal-title">{{ editingFile.name }}</h3></div>
          <button class="icon-btn" @click="showEditor = false">×</button>
        </div>
        <div class="modal-body"><textarea v-model="editingFile.content" class="code-editor" rows="22" spellcheck="false"></textarea></div>
        <div class="modal-foot"><button class="aura-btn aura-btn-ghost" @click="showEditor = false">{{ t('common.cancel') }}</button><button class="aura-btn aura-btn-primary" @click="saveFile">{{ t('common.save') }}</button></div>
      </div>
    </div>

    <!-- Chmod modal -->
    <div v-if="showChmod && chmodFile" class="overlay" @click.self="showChmod = false">
      <div class="aura-card modal modal-sm">
        <div class="modal-head"><div><span class="kicker">{{ t('files.permissions') }}</span><h3 class="modal-title">{{ chmodFile.name }}</h3></div><button class="icon-btn" @click="showChmod = false">×</button></div>
        <div class="modal-body">
          <p class="hint">{{ t('files.octalHint') }}</p>
          <input v-model="chmodMode" class="aura-input" placeholder="0755" />
          <div class="hint" style="margin-top:8px">{{ t('files.current') }}: <code class="perm">{{ formatPerms(chmodFile.mode) }}</code></div>
        </div>
        <div class="modal-foot"><button class="aura-btn aura-btn-ghost" @click="showChmod = false">{{ t('common.cancel') }}</button><button class="aura-btn aura-btn-primary" @click="doChmod">{{ t('common.confirm') }}</button></div>
      </div>
    </div>

    <!-- Rename modal -->
    <div v-if="showRename && renameFile" class="overlay" @click.self="showRename = false">
      <div class="aura-card modal modal-sm">
        <div class="modal-head"><div><span class="kicker">{{ t('files.rename') }}</span><h3 class="modal-title">{{ renameFile.name }}</h3></div><button class="icon-btn" @click="showRename = false">×</button></div>
        <div class="modal-body"><input v-model="renameName" class="aura-input" @keyup.enter="doRename" autofocus /></div>
        <div class="modal-foot"><button class="aura-btn aura-btn-ghost" @click="showRename = false">{{ t('common.cancel') }}</button><button class="aura-btn aura-btn-primary" @click="doRename">{{ t('common.save') }}</button></div>
      </div>
    </div>

    <!-- New file modal -->
    <div v-if="showNewFile" class="overlay" @click.self="showNewFile = false">
      <div class="aura-card modal modal-sm">
        <div class="modal-head"><div><span class="kicker">{{ t('files.newFile') }}</span><h3 class="modal-title">{{ t('files.fileCreate') }}</h3></div><button class="icon-btn" @click="showNewFile = false">×</button></div>
        <div class="modal-body"><input v-model="newFileName" class="aura-input" :placeholder="t('files.placeholderFile')" @keyup.enter="createFile" autofocus /></div>
        <div class="modal-foot"><button class="aura-btn aura-btn-ghost" @click="showNewFile = false">{{ t('common.cancel') }}</button><button class="aura-btn aura-btn-primary" @click="createFile">{{ t('common.create') }}</button></div>
      </div>
    </div>

    <!-- New dir modal -->
    <div v-if="showNewDir" class="overlay" @click.self="showNewDir = false">
      <div class="aura-card modal modal-sm">
        <div class="modal-head"><div><span class="kicker">{{ t('files.newFolder') }}</span><h3 class="modal-title">{{ t('files.folderCreate') }}</h3></div><button class="icon-btn" @click="showNewDir = false">×</button></div>
        <div class="modal-body"><input v-model="newDirName" class="aura-input" :placeholder="t('files.placeholderFolder')" @keyup.enter="createDir" autofocus /></div>
        <div class="modal-foot"><button class="aura-btn aura-btn-ghost" @click="showNewDir = false">{{ t('common.cancel') }}</button><button class="aura-btn aura-btn-primary" @click="createDir">{{ t('common.create') }}</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fm-page { display: flex; flex-direction: column; gap: 14px; }
.page-head { display: flex; justify-content: space-between; align-items: flex-end; gap: 12px; flex-wrap: wrap; }
.page-head h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); }
.page-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); }
.head-actions { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.icon-only { padding: 10px 12px; }

.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }

.breadcrumb { padding: 12px 16px; display: flex; flex-direction: column; gap: 6px; }
.bc-path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; color: var(--aura-text-muted); word-break: break-all; }
.bc-links { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.bc-link { font-size: 13px; font-weight: 550; color: var(--aura-accent); text-decoration: none; }
.bc-link:hover { text-decoration: underline; }
.sep { color: var(--aura-text-faint); font-size: 12px; }

.alert { padding: 10px 14px; border-radius: 10px; background: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; font-size: 13px; }
[data-theme="dark"] .alert { background: #451a1a; border-color: #7f1d1d; color: #fecaca; }
.state { text-align: center; padding: 40px; font-size: 13px; }
.state.muted { color: var(--aura-text-muted); }

.table-wrap { overflow: hidden; padding: 0; }
.ft-header { display: grid; grid-template-columns: 1fr 90px 110px 150px 140px; gap: 8px; padding: 10px 14px; background: var(--aura-bg-subtle); border-bottom: 1px solid var(--aura-border); }
.ft-row { display: grid; grid-template-columns: 1fr 90px 110px 150px 140px; gap: 8px; padding: 10px 14px; border-bottom: 1px solid var(--aura-border); align-items: center; font-size: 13px; }
.ft-row:last-child { border-bottom: none; }
.ft-row:hover { background: var(--aura-surface-hover); }
.ft-empty { text-align: center; padding: 32px; color: var(--aura-text-muted); font-size: 13px; }

.col-name { display: flex; align-items: center; gap: 10px; overflow: hidden; cursor: pointer; min-width: 0; }
.fi-icon { width: 28px; height: 28px; border-radius: 8px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); font-size: 12px; color: var(--aura-text-muted); flex-shrink: 0; }
.fi-icon.dir { background: var(--aura-accent-soft); border-color: transparent; color: var(--aura-accent); }
.fi-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--aura-text); }
.fi-name.dir { font-weight: 600; }
.fi-name:hover { color: var(--aura-accent); }

.col-size { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; color: var(--aura-text-muted); }
.col-perm .perm { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; padding: 3px 8px; border-radius: 6px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-muted); }
.col-date { font-size: 12px; color: var(--aura-text-muted); }
.col-act { display: flex; gap: 4px; justify-content: flex-end; }

.icon-btn { width: 28px; height: 28px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; font-size: 12px; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); border-color: var(--aura-border-strong); }
.icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.icon-btn.danger:hover { color: var(--aura-danger); border-color: color-mix(in srgb, var(--aura-danger) 20%, var(--aura-border)); }

.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; overflow: hidden; }
.modal-sm { max-width: 420px; }
.modal-xl { max-width: 880px; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 16px 18px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 14px; font-weight: 700; color: var(--aura-text); margin-top: 2px; word-break: break-all; }
.modal-body { padding: 18px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 18px; border-top: 1px solid var(--aura-border); }

.aura-input { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); }
.aura-input:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.aura-input:placeholder { color: var(--aura-text-faint); }
.hint { font-size: 12px; color: var(--aura-text-muted); line-height: 1.5; }

.code-editor { width: 100%; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; line-height: 1.55; padding: 14px; border: 1px solid var(--aura-border); border-radius: 10px; background: #0f172a; color: #e2e8f0; resize: vertical; }
.code-editor:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }

@media (max-width: 860px) {
  .ft-header, .ft-row { grid-template-columns: 1fr 80px 90px; }
  .ft-header span:nth-child(4), .ft-row span:nth-child(4) { display: none; }
  .ft-header span:nth-child(2), .ft-row span:nth-child(2) { display: none; }
}
</style>
