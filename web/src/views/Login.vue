<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-100">
    <form class="card w-full max-w-sm" @submit.prevent="submit">
      <h1 class="text-2xl font-bold text-brand-dark mb-1">MUDAHURUS</h1>
      <p class="text-sm text-gray-500 mb-4">{{ t('login.title') }}</p>
      <div class="mb-3">
        <label class="label">{{ t('login.username') }}</label>
        <input v-model="username" class="input" autocomplete="username" />
      </div>
      <div class="mb-4">
        <label class="label">{{ t('login.password') }}</label>
        <input v-model="password" type="password" class="input" autocomplete="current-password" />
      </div>
      <p v-if="error" class="text-sm text-red-600 mb-3">{{ error }}</p>
      <button class="btn-primary w-full justify-center" :disabled="loading">
        {{ loading ? t('common.loading') : t('login.submit') }}
      </button>
      <router-link to="/register" class="mt-3 block text-center text-sm text-brand">{{ t('login.register') }}</router-link>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value, password.value)
    router.push(route.query.redirect || '/admin/dashboard')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
