<template>
  <v-app>
    <v-app-bar>
      <v-toolbar-title>QueryExplorer</v-toolbar-title>
      <v-spacer />
      <div v-if="auth.me">{{ auth.me.email }}</div>
      <v-btn @click="auth.logout">Logout</v-btn>
    </v-app-bar>

    <v-navigation-drawer permanent>
      <v-list nav>
        <v-list-item to="/">Home</v-list-item>
        <v-list-item to="/queries">Queries</v-list-item>
        <v-list-item to="/runs">Runs</v-list-item>
        <v-list-item v-if="auth.isAdmin" to="/admin/audit">Audit</v-list-item>
      </v-list>
    </v-navigation-drawer>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
const auth = useAuthStore()
</script>