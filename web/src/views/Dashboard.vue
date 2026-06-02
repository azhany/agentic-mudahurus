<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-4">{{ t('nav.dashboard') }}</h1>
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
      <div v-for="c in cards" :key="c.key" class="card">
        <div class="text-3xl font-bold text-brand-dark">{{ counts[c.key] ?? '—' }}</div>
        <div class="text-sm text-gray-500">{{ t(`dash.${c.label}`) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'

const { t } = useI18n()
const counts = ref({})
const cards = [
  { key: 'orders', label: 'orders' },
  { key: 'pending_orders', label: 'pending' },
  { key: 'products', label: 'products' },
  { key: 'customers', label: 'customers' },
  { key: 'shipped_orders', label: 'shipped' },
]
onMounted(async () => {
  try { counts.value = await api.get('/dashboard/counts') } catch {}
})
</script>
