<template>
  <div>
    <div class="page-header">
      <h2>Queries</h2>
      <router-link v-if="auth.isEditor" to="/queries/new" class="btn-primary">New query</router-link>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>

    <table v-if="queries.length" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Description</th>
          <th>Owner</th>
          <th>Updated</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="q in queries" :key="q.id">
          <td><router-link :to="`/queries/${q.id}/edit`">{{ q.name }}</router-link></td>
          <td class="muted">{{ q.description ?? '—' }}</td>
          <td class="muted">{{ q.owner_email }}</td>
          <td class="muted">{{ formatDate(q.updated_at) }}</td>
          <td class="actions">
            <router-link :to="`/queries/${q.id}/edit`" class="btn-sm">Open</router-link>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="empty">No queries yet.</p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listQueries, type Query } from '../api/queries'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const queries = ref<Query[]>([])
const error = ref('')

async function load() {
  try {
    queries.value = await listQueries()
  } catch (e: any) {
    error.value = e.message
  }
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('sv-SE', { dateStyle: 'short' })
}

onMounted(load)
</script>

<style scoped>
.page-header { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
.page-header h2 { margin: 0; }
.table { width: 100%; border-collapse: collapse; font-size: 14px; }
.table th, .table td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #eee; }
.table th { font-weight: 600; color: #555; }
.table a { color: #2563eb; text-decoration: none; }
.table a:hover { text-decoration: underline; }
.muted { color: #666; }
.actions { display: flex; gap: 8px; }
.empty { color: #888; }
.error-banner { background: #fee; border: 1px solid #fcc; padding: 8px 12px; border-radius: 4px; margin-bottom: 16px; color: #c00; font-size: 14px; }
.btn-primary { background: #2563eb; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; font-size: 14px; text-decoration: none; display: inline-block; }
.btn-primary:hover { background: #1d4ed8; }
.btn-sm { background: #f3f4f6; border: 1px solid #ddd; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 13px; text-decoration: none; display: inline-block; }
.btn-sm:hover { background: #e5e7eb; }
</style>