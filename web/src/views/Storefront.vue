<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-brand text-white p-6">
      <h1 class="text-2xl font-bold">{{ store.store_name || username }}</h1>
      <p class="text-white/80 text-sm">{{ t('store.search') }}</p>
    </header>
    <div class="max-w-5xl mx-auto p-6">
      <div class="flex gap-2 mb-4">
        <input v-model="q" class="input" :placeholder="t('store.search')" @keyup.enter="doSearch" />
        <button class="btn-primary" @click="doSearch">{{ t('common.search') }}</button>
      </div>

      <!-- Product Q&A widget (MH-603 storefront) -->
      <div class="card mb-6">
        <div class="flex gap-2">
          <input v-model="ask" class="input" :placeholder="t('store.ask')" @keyup.enter="doAsk" />
          <button class="btn-ghost" @click="doAsk">💬</button>
        </div>
        <p v-if="answer" class="mt-2 text-sm" :class="answer.refused ? 'text-amber-600' : 'text-gray-700'">{{ answer.answer }}</p>
      </div>

      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <router-link v-for="p in products" :key="p.id" :to="`/store/${username}/product/${p.id}`" class="card hover:shadow-md transition">
          <img v-if="p.image_url" :src="p.image_url" class="h-32 w-full object-cover rounded mb-2" />
          <div v-else class="h-32 w-full bg-gray-100 rounded mb-2 flex items-center justify-center text-gray-300">no image</div>
          <div class="font-medium text-sm">{{ p.product_name }}</div>
          <div class="text-brand font-semibold">RM {{ Number(p.unit_price).toFixed(2) }}</div>
        </router-link>
      </div>
      <p v-if="!products.length" class="text-center text-gray-400 py-10">{{ t('common.empty') }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

const { t } = useI18n()
const route = useRoute()
const username = route.params.username
const BASE = import.meta.env.VITE_API_BASE_URL || ''
const store = ref({}), products = ref([]), q = ref(''), ask = ref(''), answer = ref(null)

async function load() {
  const res = await fetch(`${BASE}/store/${username}`)
  if (res.ok) { const d = await res.json(); store.value = d.store || {}; products.value = d.products || [] }
}
async function doSearch() {
  const res = await fetch(`${BASE}/store/${username}/search?q=` + encodeURIComponent(q.value))
  if (res.ok) products.value = (await res.json()).products || []
}
async function doAsk() {
  if (!ask.value.trim()) return
  const res = await fetch(`${BASE}/store/${username}/ask`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ question: ask.value }),
  })
  answer.value = res.ok ? await res.json() : { answer: 'Unavailable', refused: true }
}
onMounted(load)
</script>
