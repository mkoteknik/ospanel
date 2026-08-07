<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface User {
  id: number; username: string; email: string; role: string
  status: string; quota_limit: number; last_login_at: string; created_at: string
}

const users = ref<User[]>([])
const loading = ref(false)
const showAdd = ref(false)
const newUser = ref({ username: '', email: '', password: '', role: 'user' })

async function loadUsers() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/admin/users')
    users.value = res.data.users || []
  } catch { }
  finally { loading.value = false }
}

async function createUser() {
  try {
    await api.post('/api/v1/admin/users', {
      username: newUser.value.username,
      email: newUser.value.email,
      password: newUser.value.password,
      role: newUser.value.role,
    })
    showAdd.value = false
    newUser.value = { username: '', email: '', password: '', role: 'user' }
    await loadUsers()
    alert('Kullanici olusturuldu!')
  } catch { }
}

async function deleteUser(id: number) {
  if (!confirm('Bu kullanici silinecek!')) return
  try { await api.delete('/api/v1/admin/users/' + id); await loadUsers() }
  catch { }
}

async function updateUser(id: number, key: string, value: any) {
  try { await api.put('/api/v1/admin/users/' + id, { [key]: value }); await loadUsers() }
  catch { }
}

onMounted(loadUsers)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>Kullanici Yonetimi</h2>
        <p>Sistem kullanicilarini yonetin.</p>
      </div>
      <button class="btn-primary" @click="showAdd = true">+ Kullanici Ekle</button>
    </div>

    <div v-if="loading" class="loading">Yukleniyor...</div>

    <div v-else class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>ID</th><th>Kullanici</th><th>Email</th><th>Rol</th><th>Durum</th><th>Kota (MB)</th><th>Son Giris</th><th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>{{ u.id }}</td>
            <td><strong>{{ u.username }}</strong></td>
            <td>{{ u.email }}</td>
            <td>
              <select :value="u.role" @change="updateUser(u.id, 'role', ($event.target as HTMLSelectElement).value)" class="sel-sm">
                <option value="admin">Admin</option>
                <option value="reseller">Reseller</option>
                <option value="user">User</option>
              </select>
            </td>
            <td>
              <select :value="u.status" @change="updateUser(u.id, 'status', ($event.target as HTMLSelectElement).value)" class="sel-sm">
                <option value="active">Aktif</option>
                <option value="inactive">Pasif</option>
              </select>
            </td>
            <td>{{ u.quota_limit || '-' }}</td>
            <td>{{ u.last_login_at?.split('T')[0] || '-' }}</td>
            <td><button class="btn-danger-sm" @click="deleteUser(u.id)">Sil</button></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showAdd" class="modal-overlay" @click.self="showAdd=false">
      <div class="modal">
        <div class="modal-header"><h3>+ Yeni Kullanici</h3><button class="modal-close" @click="showAdd=false">X</button></div>
        <div class="modal-body">
          <div class="form-group"><label>Kullanici Adi</label><input v-model="newUser.username" placeholder="kullaniciadi" /></div>
          <div class="form-group"><label>Email</label><input v-model="newUser.email" type="email" placeholder="user@site.com" /></div>
          <div class="form-group"><label>Sifre</label><input v-model="newUser.password" type="password" placeholder="En az 8 karakter" /></div>
          <div class="form-group"><label>Rol</label>
            <select v-model="newUser.role" class="sel">
              <option value="user">User</option>
              <option value="reseller">Reseller</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showAdd=false">Iptal</button><button class="btn-primary" @click="createUser">Olustur</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }
.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.loading { text-align: center; padding: 60px; color: #888; }
.table-wrap { background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); overflow: hidden; }
table { width: 100%; border-collapse: collapse; }
th { text-align: left; padding: 12px 16px; background: #fafafa; border-bottom: 2px solid #e5e5e5; font-size: 11px; color: #888; text-transform: uppercase; }
td { padding: 12px 16px; border-bottom: 1px solid #f5f5f5; font-size: 13px; }
tr:hover { background: #f8f9fa; }
.sel-sm { padding: 4px 8px; border: 1px solid #e0e0e0; border-radius: 4px; font-size: 12px; }
.btn-danger-sm { padding: 4px 12px; background: white; color: #d32f2f; border: 1px solid #d32f2f; border-radius: 4px; font-size: 12px; cursor: pointer; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 500px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 6px; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus { outline: none; border-color: #0f3460; }
.sel { width: 100%; padding: 10px 14px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; background: white; cursor: pointer; }
</style>
