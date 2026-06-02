import { defineStore } from 'pinia'

const BASE = import.meta.env.VITE_API_BASE_URL || ''

export const useAuthStore = defineStore('auth', {
  state: () => ({
    accessToken: localStorage.getItem('mh_access') || '',
    refreshToken: localStorage.getItem('mh_refresh') || '',
    tenant: JSON.parse(localStorage.getItem('mh_tenant') || 'null'),
  }),
  getters: {
    isAuthenticated: (s) => !!s.accessToken,
  },
  actions: {
    persist() {
      localStorage.setItem('mh_access', this.accessToken)
      localStorage.setItem('mh_refresh', this.refreshToken)
      localStorage.setItem('mh_tenant', JSON.stringify(this.tenant))
    },
    async login(username, password) {
      const res = await fetch(BASE + '/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!res.ok) {
        const d = await res.json().catch(() => ({}))
        throw new Error(d?.error?.message || 'Login failed')
      }
      const data = await res.json()
      this.accessToken = data.access_token
      this.refreshToken = data.refresh_token
      this.tenant = data.tenant
      this.persist()
    },
    async refresh() {
      try {
        const res = await fetch(BASE + '/auth/refresh', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: this.refreshToken }),
        })
        if (!res.ok) return false
        const data = await res.json()
        this.accessToken = data.access_token
        this.refreshToken = data.refresh_token
        this.tenant = data.tenant
        this.persist()
        return true
      } catch {
        return false
      }
    },
    async logout() {
      try {
        await fetch(BASE + '/auth/logout', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: this.refreshToken }),
        })
      } catch {}
      this.logoutLocal()
    },
    logoutLocal() {
      this.accessToken = ''
      this.refreshToken = ''
      this.tenant = null
      localStorage.removeItem('mh_access')
      localStorage.removeItem('mh_refresh')
      localStorage.removeItem('mh_tenant')
    },
  },
})
