<template>
  <v-app>
    <v-app-bar>
      <v-toolbar-title>QueryExplorer</v-toolbar-title>
      <v-spacer />
      <div v-if="auth.me">{{ auth.me.email }} - {{ auth.canAdhoc}}</div>
      <v-btn @click="auth.logout">Logout</v-btn>
    </v-app-bar>

    <v-navigation-drawer permanent>
      <v-list nav>
        <v-list-item to="/">Home</v-list-item>
        <v-list-item to="/queries">Queries</v-list-item>
        <v-list-item to="/runs">Runs</v-list-item>
        <v-list-item v-if="auth.isAdmin" to="/admin/audit">Audit</v-list-item>
        <p>hibla {{ auth.canAdhoc }}</p>
        <v-list-item v-if="auth.canAdhoc" to="/explore">Explore</v-list-item>
      </v-list>
    </v-navigation-drawer>

    <v-main style="overflow: hidden; height: 100vh;">
      <router-view />
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
const auth = useAuthStore()
</script>