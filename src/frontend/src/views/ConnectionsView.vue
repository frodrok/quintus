<template>
  <div>
    <div class="page-header">
      <h2>Connections</h2>
      <button class="btn-primary" @click="openCreate">Add connection</button>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>

    <table v-if="connections.length" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Driver</th>
          <th>Read only</th>
          <th>Timeout (ms)</th>
          <th>Created by</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="c in connections" :key="c.id">
          <td>{{ c.name }}</td>
          <td>{{ c.driver }}</td>
          <td>{{ c.read_only ? 'Yes' : 'No' }}</td>
          <td>{{ c.statement_timeout_ms }}</td>
          <td>{{ c.created_by_email ?? '—' }}</td>
          <td class="actions">
            <button class="btn-sm" @click="test(c.id)">Test</button>
            <button class="btn-sm" @click="openEdit(c)">Edit</button>
            <button class="btn-sm btn-danger" @click="remove(c.id)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="empty">No connections yet.</p>

    <!-- Modal -->
    <div v-if="modal.open" class="modal-backdrop" @click.self="modal.open = false">
      <div class="modal">
        <h3>{{ modal.id ? 'Edit connection' : 'New connection' }}</h3>
        <div class="field">
          <label>Name</label>
          <input v-model="modal.name" type="text" />
        </div>
        <div class="field">
          <label>Driver</label>
          <select v-model="modal.driver">
            <option value="postgres">postgres</option>
          </select>
        </div>
        <div class="field">
  <label>DSN</label>
  <div class="input-with-toggle">
    <input v-model="modal.dsn" :type="showDsn ? 'text' : 'password'" placeholder="postgres://user:pass@host/db" />
    <button type="button" class="toggle-visibility" @click="showDsn = !showDsn">
      {{ showDsn ? '🙈' : '👁' }}
    </button>
  </div>
</div>
        <div class="field checkbox">
          <label><input v-model="modal.read_only" type="checkbox" /> Read only</label>
        </div>
        <div class="field">
          <label>Statement timeout (ms)</label>
          <input v-model.number="modal.statement_timeout_ms" type="number" />
        </div>
        <div v-if="modal.error" class="error-banner">{{ modal.error }}</div>
        <div class="modal-actions">
          <button class="btn-primary" @click="save" :disabled="modal.saving">
            {{ modal.saving ? 'Saving…' : 'Save' }}
          </button>
          <button @click="modal.open = false">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import {
  listConnections, createConnection, updateConnection,
  deleteConnection, testConnection, type Connection,
} from '../api/connections'

const connections = ref<Connection[]>([])
const error = ref('')

const showDsn = ref(false)

const modal = reactive({
  open: false,
  id: '',
  name: '',
  driver: 'postgres',
  dsn: '',
  read_only: true,
  statement_timeout_ms: 30000,
  saving: false,
  error: '',
})

async function load() {
  try {
    connections.value = await listConnections()
  } catch (e: any) {
    error.value = e.message
  }
}

function openCreate() {
  showDsn.value = false
  Object.assign(modal, { open: true, id: '', name: '', driver: 'postgres', dsn: '', read_only: true, statement_timeout_ms: 30000, saving: false, error: '' })
}

function openEdit(c: Connection) {
  showDsn.value = false
  Object.assign(modal, { open: true, id: c.id, name: c.name, driver: c.driver, dsn: '', read_only: c.read_only, statement_timeout_ms: c.statement_timeout_ms, saving: false, error: '' })
}

async function save() {
  modal.saving = true
  modal.error = ''
  try {
    const input = { name: modal.name, driver: modal.driver, dsn: modal.dsn, read_only: modal.read_only, statement_timeout_ms: modal.statement_timeout_ms }
    if (modal.id) {
      await updateConnection(modal.id, input)
    } else {
      await createConnection(input)
    }
    modal.open = false
    await load()
  } catch (e: any) {
    modal.error = e.response?.data?.error ?? e.message
  } finally {
    modal.saving = false
  }
}

async function test(id: string) {
  try {
    const r = await testConnection(id)
    alert(`Connection status: ${r.status}`)
  } catch (e: any) {
    alert(`Test failed: ${e.response?.data?.error ?? e.message}`)
  }
}

async function remove(id: string) {
  if (!confirm('Delete this connection?')) return
  try {
    await deleteConnection(id)
    await load()
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
}

onMounted(load)
</script>

<style scoped>
.input-with-toggle { display: flex; align-items: center; gap: 4px; }
.input-with-toggle input { flex: 1; }
.toggle-visibility { background: none; border: 1px solid #ddd; border-radius: 4px; padding: 6px 8px; cursor: pointer; font-size: 14px; line-height: 1; }
.toggle-visibility:hover { background: #f3f4f6; }
.page-header { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
.page-header h2 { margin: 0; }
.table { width: 100%; border-collapse: collapse; font-size: 14px; }
.table th, .table td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #eee; }
.table th { font-weight: 600; color: #555; }
.actions { display: flex; gap: 8px; }
.empty { color: #888; }
.error-banner { background: #fee; border: 1px solid #fcc; padding: 8px 12px; border-radius: 4px; margin-bottom: 16px; color: #c00; font-size: 14px; }
.btn-primary { background: #2563eb; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-primary:hover { background: #1d4ed8; }
.btn-primary:disabled { opacity: 0.6; cursor: default; }
.btn-sm { background: #f3f4f6; border: 1px solid #ddd; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-sm:hover { background: #e5e7eb; }
.btn-danger { color: #dc2626; }
.btn-danger:hover { background: #fee2e2; border-color: #fca5a5; }
.field { margin-bottom: 12px; }
.field label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 4px; }
.field input[type=text], .field input[type=password], .field input[type=number], .field select { width: 100%; padding: 6px 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; }
.field.checkbox label { display: flex; align-items: center; gap: 8px; }
.modal-backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { background: white; border-radius: 8px; padding: 24px; width: 480px; max-width: 95vw; }
.modal h3 { margin: 0 0 20px; }
.modal-actions { display: flex; gap: 8px; margin-top: 20px; }
</style>