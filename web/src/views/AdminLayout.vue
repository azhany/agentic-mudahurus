<template>
  <div class="min-h-screen bg-gray-50">
    <aside class="fixed inset-y-0 left-0 w-60 bg-brand-dark text-white p-4 flex flex-col">
      <div class="text-xl font-bold mb-6">MUDAHURUS</div>
      <nav class="space-y-1 flex-1">
        <router-link v-for="item in nav" :key="item.to" :to="item.to"
          class="block rounded px-3 py-2 text-sm hover:bg-white/10"
          active-class="bg-white/20 font-semibold">{{ t(`nav.${item.key}`) }}</router-link>
      </nav>
      <div class="space-y-2 pt-4 border-t border-white/20">
        <select v-model="locale" class="w-full rounded bg-white/10 px-2 py-1 text-sm">
          <option value="ms">Bahasa Melayu</option>
          <option value="en">English</option>
        </select>
        <button class="w-full text-left rounded px-3 py-2 text-sm hover:bg-white/10" @click="doLogout">
          {{ t('nav.logout') }}
        </button>
      </div>
    </aside>
    <main class="ml-60 p-6">
      <AssistantPanel />
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AssistantPanel from '../components/AssistantPanel.vue'

const { t, locale } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const nav = [
  { key: 'dashboard', to: '/admin/dashboard' },
  { key: 'products', to: '/admin/products' },
  { key: 'categories', to: '/admin/categories' },
  { key: 'orders', to: '/admin/orders' },
  { key: 'customers', to: '/admin/customers' },
  { key: 'coupons', to: '/admin/coupons' },
]

const _ = computed(() => (localStorage.setItem('mh_locale', locale.value), locale.value))
async function doLogout() {
  await auth.logout()
  router.push('/login')
}
</script>
