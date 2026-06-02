<template>
  <div class="min-h-screen bg-gray-100 p-6">
    <div class="max-w-2xl mx-auto card" v-if="inv">
      <div class="flex justify-between items-start mb-4">
        <div>
          <h1 class="text-2xl font-bold text-brand-dark">INVOICE</h1>
          <p class="text-gray-500">{{ inv.store_name || inv.seller_name }}</p>
        </div>
        <a :href="pdfUrl" target="_blank" class="btn-ghost">⬇ PDF</a>
      </div>
      <div class="text-sm text-gray-600 mb-4">
        <p>Invoice: {{ order.id }}</p>
        <p>Status: <span class="badge bg-blue-100 text-blue-700">{{ order.status }}</span></p>
        <p>Bill to: {{ order.full_name }} ({{ order.email }})</p>
      </div>
      <table class="w-full text-sm mb-4">
        <thead class="border-b text-left text-gray-500"><tr><th class="py-1">Item</th><th>Qty</th><th>Unit</th><th>Total</th></tr></thead>
        <tbody>
          <tr v-for="it in order.items" :key="it.id" class="border-b"><td class="py-1">{{ it.product_name || it.sku }}</td><td>{{ it.quantity }}</td><td>RM {{ Number(it.unit_price).toFixed(2) }}</td><td>RM {{ Number(it.line_total).toFixed(2) }}</td></tr>
        </tbody>
      </table>
      <div class="text-right font-bold">Total: RM {{ Number(order.total_price).toFixed(2) }}</div>
    </div>
    <p v-else class="text-center text-gray-400">{{ t('common.loading') }}</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

const { t } = useI18n()
const route = useRoute()
const BASE = import.meta.env.VITE_API_BASE_URL || ''
const id = route.params.id
const inv = ref(null)
const order = computed(() => inv.value?.order || {})
const pdfUrl = `${BASE}/invoice/${id}?format=pdf`

onMounted(async () => {
  const res = await fetch(`${BASE}/invoice/${id}`)
  if (res.ok) inv.value = await res.json()
})
</script>
