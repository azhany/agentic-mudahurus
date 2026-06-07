<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-4">{{ t('nav.orders') }}</h1>
    <div class="flex gap-2 mb-4">
      <select v-model="status" class="input max-w-xs" @change="load">
        <option value="">All</option>
        <option v-for="s in statuses" :key="s" :value="s">{{ s }}</option>
      </select>
      <input v-model="search" class="input max-w-xs" :placeholder="t('common.search')" @input="debouncedLoad" />
    </div>
    <div class="card overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-left text-gray-500 border-b"><tr><th class="py-2">Customer</th><th>Total</th><th>Status</th><th>Expires</th><th>{{ t('common.actions') }}</th></tr></thead>
        <tbody>
          <tr v-for="o in items" :key="o.id" class="border-b last:border-0">
            <td class="py-2">{{ o.full_name }}</td>
            <td>RM {{ Number(o.total_price).toFixed(2) }}</td>
            <td><span class="badge bg-blue-100 text-blue-700">{{ o.status }}</span></td>
            <td>{{ new Date(o.expired_date).toLocaleDateString() }}</td>
            <td><button class="btn-ghost" @click="open(o)">{{ t('common.edit') }}</button></td>
          </tr>
          <tr v-if="!items.length"><td colspan="5" class="py-6 text-center text-gray-400">{{ t('common.empty') }}</td></tr>
        </tbody>
      </table>
    </div>

    <Modal v-if="detail" @close="detail=null">
      <h2 class="text-lg font-semibold mb-1">Order {{ detail.id.slice(0,8) }}</h2>
      <p class="text-sm text-gray-500 mb-3">{{ detail.full_name }} · {{ detail.email }} · {{ detail.contact_no }}</p>
      <div class="text-sm mb-3">
        <div v-for="it in detail.items" :key="it.id" class="flex justify-between border-b py-1">
          <span>{{ it.product_name || it.sku }} × {{ it.quantity }}</span>
          <span>RM {{ Number(it.line_total).toFixed(2) }}</span>
        </div>
        <div class="flex justify-between font-semibold pt-1"><span>Total</span><span>RM {{ Number(detail.total_price).toFixed(2) }}</span></div>
      </div>

      <div class="mb-3">
        <label class="label">Status</label>
        <div class="flex gap-2">
          <select v-model="newStatus" class="input"><option v-for="s in statuses" :key="s" :value="s">{{ s }}</option></select>
          <button class="btn-primary" @click="changeStatus">{{ t('common.save') }}</button>
        </div>
      </div>

      <div v-if="detail.payments?.length" class="mb-3">
        <label class="label">Payment proof</label>
        <div v-for="p in detail.payments" :key="p.id" class="flex items-center gap-2 mb-1">
          <a :href="p.proof_url" target="_blank" class="text-brand text-sm underline">view proof</a>
          <span class="badge bg-gray-100 text-gray-600">{{ p.status }}</span>
          <button class="btn-ghost" @click="verify(p, true)">✓</button>
          <button class="btn-danger" @click="verify(p, false)">✗</button>
        </div>
      </div>

      <a :href="invoiceUrl" target="_blank" class="btn-ghost">⬇ Invoice PDF</a>
      <p v-if="err" class="text-sm text-red-600 mt-2">{{ err }}</p>
    </Modal>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import Modal from '../components/Modal.vue'

const { t } = useI18n()
const items = ref([]), detail = ref(null), err = ref(''), status = ref(''), search = ref(''), newStatus = ref('')
const statuses = ['pending','payment_received','payment_accepted','shipped','expired','cancelled','rejected']
const invoiceUrl = computed(() => detail.value ? `${import.meta.env.VITE_API_BASE_URL || '/api'}/orders/${detail.value.id}/invoice.pdf` : '#')

let timer
function debouncedLoad() { clearTimeout(timer); timer = setTimeout(load, 300) }
async function load() {
  const qs = new URLSearchParams()
  if (status.value) qs.set('status', status.value)
  if (search.value) qs.set('search', search.value)
  const res = await api.get('/orders?' + qs.toString())
  items.value = res.records || []
}
async function open(o) { detail.value = await api.get('/orders/' + o.id); newStatus.value = detail.value.status; err.value='' }
async function changeStatus() {
  err.value=''
  try { detail.value = await api.patch('/orders/' + detail.value.id + '/status', { status: newStatus.value }); await load() }
  catch (e) { err.value = e.message }
}
async function verify(p, accept) {
  await api.patch('/payments/' + p.id + '/verify', { accept })
  detail.value = await api.get('/orders/' + detail.value.id)
}
load()
</script>
