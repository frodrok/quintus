<template>
  <div v-if="auth.loaded" class="app-shell">
    <header class="app-header">
      <div class="title">QueryExplorer</div>
      <nav class="nav">
  <router-link to="/">Home</router-link>
  <router-link v-if="auth.isAdmin" to="/connections">Connections</router-link>
  <router-link v-if="auth.isAdmin" to="/connections">Audit</router-link>
  <router-link v-if="auth.isEditor" to="/queries">Queries</router-link>
  <router-link v-if="auth.canAdhoc" to="/explore">Explore</router-link>
  <router-link to="/ai-explore">AI Explore</router-link>
  <router-link to="/runs">Runs</router-link>
  
</nav>
      <div class="spacer" />
      <div v-if="auth.me" class="user">
        <span>{{ auth.me.email }}</span>
        <button @click="auth.logout">Logout</button>
      </div>
    </header>

    <main class="app-main">
      <router-view />
    </main>
  </div>

  <div v-else class="loading">
    Loading...
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()

onMounted(async () => {
  await auth.load()
})
</script>

<style scoped>
.nav { display: flex; gap: 16px; font-size: 14px; }
.nav a { color: #2563eb; text-decoration: none; }
.nav a.router-link-active { font-weight: 600; }
.app-shell {
  min-height: 100vh;
  font-family: system-ui, sans-serif;
}
.app-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 20px;
  border-bottom: 1px solid #ddd;
}
.title {
  font-size: 20px;
  font-weight: 700;
}
.spacer {
  flex: 1;
}
.user {
  display: flex;
  align-items: center;
  gap: 12px;
}
.app-main {
  padding: 24px;
}
.loading {
  padding: 24px;
  font-family: system-ui, sans-serif;
}
</style>