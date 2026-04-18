import { defineStore } from 'pinia'
import { getLogout, getMe, type Me } from '../api/auth'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    me: null as Me | null,
    loaded: false,
  }),

  getters: {
    isViewer: (s) => !!s.me,
    isEditor: (s) => s.me?.role === 'editor' || s.me?.role === 'admin',
    isAdmin: (s) => s.me?.role === 'admin',
    canAdhoc: (s) => s.me?.can_adhoc === true,
  },

  actions: {
    async load() {
      this.me = await getMe()
      this.loaded = true
    },

    async logout() {
      const { url } = await getLogout()
      window.location.href = url
    },
  },
})