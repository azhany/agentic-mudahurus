<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-100">
    <form class="card w-full max-w-md" @submit.prevent="submit">
      <h1 class="text-xl font-bold text-brand-dark mb-4">{{ t('login.register') }}</h1>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Username</label><input v-model="f.username" class="input" /></div>
        <div><label class="label">Email</label><input v-model="f.email" class="input" /></div>
        <div><label class="label">Store name</label><input v-model="f.store_name" class="input" /></div>
        <div><label class="label">Phone</label><input v-model="f.phone" class="input" /></div>
        <div class="col-span-2"><label class="label">Password</label><input v-model="f.password" type="password" class="input" /></div>
      </div>
      <p v-if="error" class="text-sm text-red-600 my-2">{{ error }}</p>
      <p v-if="done" class="text-sm text-green-600 my-2">Account created — please sign in.</p>
      <button class="btn-primary w-full justify-center mt-3" :disabled="loading">{{ t('login.register') }}</button>
      <router-link to="/login" class="mt-3 block text-center text-sm text-brand">{{ t('login.submit') }}</router-link>
    </form>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'

const { t } = useI18n()
const f = reactive({ username: '', email: '', password: '', store_name: '', phone: '' })
const error = ref(''), done = ref(false), loading = ref(false)

async function submit() {
  loading.value = true; error.value = ''
  try {
    await api.post('/auth/register', f)
    done.value = true
  } catch (e) { error.value = e.message } finally { loading.value = false }
}
</script>
