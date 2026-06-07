<template>
  <div class="min-h-screen bg-gray-50 p-6">
    <div class="max-w-3xl mx-auto" v-if="product">
      <router-link :to="`/store/${username}`" class="text-brand text-sm">← back</router-link>
      <div class="card mt-3 grid md:grid-cols-2 gap-6">
        <img v-if="product.image_url" :src="product.image_url" class="w-full rounded object-cover" />
        <div v-else class="bg-gray-100 rounded min-h-48 flex items-center justify-center text-gray-300">no image</div>
        <div>
          <h1 class="text-2xl font-bold">{{ product.product_name }}</h1>
          <p class="text-gray-500 text-sm mb-2">{{ product.category }}</p>
          <p class="text-2xl text-brand font-bold mb-3">RM {{ Number(product.unit_price).toFixed(2) }}</p>
          <p class="text-sm text-gray-700 mb-4">{{ product.description }}</p>
          <button class="btn-primary" @click="checkout=true">{{ t('store.checkout') }}</button>
        </div>
      </div>

      <div v-if="checkout" class="card mt-4">
        <h2 class="font-semibold mb-3">{{ t('store.checkout') }}</h2>
        <div class="grid grid-cols-2 gap-3">
          <div><label class="label">Name</label><input v-model="f.full_name" class="input" /></div>
          <div><label class="label">Phone</label><input v-model="f.contact_no" class="input" /></div>
          <div><label class="label">Email</label><input v-model="f.email" class="input" /></div>
          <div><label class="label">Qty</label><input v-model.number="qty" type="number" min="1" class="input" /></div>
          <div class="col-span-2"><label class="label">Address</label><input v-model="f.mailing_addr" class="input" /></div>
          <div><label class="label">City</label><input v-model="f.city" class="input" /></div>
          <div><label class="label">Postcode</label><input v-model="f.postcode" class="input" /></div>
        </div>
        <p v-if="err" class="text-sm text-red-600 mt-2">{{ err }}</p>
        <button class="btn-primary mt-3" @click="placeOrder">{{ t('store.checkout') }}</button>
      </div>

      <div v-if="placed" class="card mt-4">
        <p class="text-green-600 font-medium">Order placed! Total RM {{ Number(placed.total_price).toFixed(2) }}.</p>
        <router-link :to="`/invoice/${placed.order_id}`" class="text-brand underline text-sm">View invoice</router-link>
        <div class="mt-3">
          <label class="label">Upload payment proof</label>
          <input type="file" accept="image/*,application/pdf" @change="uploadProof" />
          <p v-if="proofMsg" class="text-sm text-green-600 mt-1">{{ proofMsg }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

const { t } = useI18n()
const route = useRoute()
const username = route.params.username
const id = route.params.id
const BASE = import.meta.env.VITE_API_BASE_URL || '/api'
const product = ref(null), checkout = ref(false), placed = ref(null), err = ref(''), proofMsg = ref(''), qty = ref(1)
const f = reactive({ full_name:'', contact_no:'', email:'', mailing_addr:'', city:'', postcode:'' })

async function load() {
  const res = await fetch(`${BASE}/store/${username}/products/${id}`)
  if (res.ok) product.value = await res.json()
}
async function placeOrder() {
  err.value = ''
  const body = {
    full_name: f.full_name, email: f.email, contact_no: f.contact_no,
    shipping_address: { mailing_addr: f.mailing_addr, city: f.city, postcode: f.postcode },
    items: [{ sku: product.value.sku, quantity: qty.value, unit_price: product.value.unit_price }],
  }
  const res = await fetch(`${BASE}/store/${username}/checkout`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
  })
  if (!res.ok) { const d = await res.json().catch(()=>({})); err.value = d?.error?.message || 'Failed'; return }
  placed.value = await res.json(); checkout.value = false
}
async function uploadProof(e) {
  const file = e.target.files[0]; if (!file) return
  const fd = new FormData(); fd.append('proof', file)
  const res = await fetch(`${BASE}/orders/${placed.value.order_id}/payment`, { method: 'POST', body: fd })
  proofMsg.value = res.ok ? 'Proof uploaded — seller will verify.' : 'Upload failed'
}
onMounted(load)
</script>
