<template>
  <v-container>
    <div class="d-flex mb-4">
      <h1 class="text-h5">Queries</h1>
      <v-spacer />
      <v-btn v-if="auth.isEditor" to="/queries/new">New query</v-btn>
    </div>

    <v-data-table :items="items" :headers="headers" />
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listQueries } from '@/api/queries'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const items = ref([])

const headers = [
  { title: 'Name', key: 'name' },
  { title: 'Description', key: 'description' },
  { title: 'Owner', key: 'owner_email' },
  { title: 'Updated', key: 'updated_at' },
]

onMounted(async () => {
  items.value = await listQueries()
})
</script>