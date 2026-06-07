<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-1">Seller Copilot</h1>
    <p class="text-sm text-gray-500 mb-4">EH-1 · type a command in plain language; review the proposed action, then run it.</p>

    <div class="card mb-4">
      <div class="flex gap-2">
        <input v-model="message" class="input" placeholder='e.g. "add product Kuih Lapis price RM12.50 category Makanan"' @keyup.enter="interpret" />
        <button class="btn-primary" :disabled="loading" @click="interpret">Interpret</button>
      </div>
      <p class="text-xs text-gray-400 mt-2">
        Examples: <code>set price of KL01 to RM9.90</code> · <code>ship order &lt;id&gt;</code> · <code>generate description for Teh Tarik</code>
      </p>
    </div>

    <div v-if="agent" class="card mb-4">
      <p class="text-sm">Routed to agent: <span class="badge bg-brand text-white">{{ agent }}</span></p>
      <p v-if="note" class="text-sm text-amber-600 mt-2">{{ note }}</p>

      <div v-for="(a, i) in actions" :key="i" class="mt-3 border-t pt-3">
        <p class="font-medium text-sm">Proposed: {{ a.Kind }}</p>
        <pre class="bg-gray-50 rounded p-2 text-xs overflow-x-auto">{{ JSON.stringify(a.Params, null, 2) }}</pre>
        <button class="btn-primary mt-2" @click="execute(a)">Confirm &amp; run</button>
      </div>
      <p v-if="agent==='seller_copilot' && !actions.length" class="text-sm text-gray-500 mt-2">Couldn't parse a command from that.</p>
    </div>

    <div v-if="result" class="card">
      <p class="text-green-600 font-medium mb-2">✓ {{ result.executed }}</p>
      <pre class="bg-gray-50 rounded p-2 text-xs overflow-x-auto">{{ JSON.stringify(result, null, 2) }}</pre>
    </div>
    <p v-if="err" class="text-sm text-red-600 mt-2">{{ err }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { api } from '../api'

const message = ref('')
const agent = ref('')
const note = ref('')
const actions = ref([])
const result = ref(null)
const err = ref('')
const loading = ref(false)

async function interpret() {
  if (!message.value.trim()) return
  loading.value = true; err.value = ''; result.value = null
  try {
    const res = await api.post('/copilot/interpret', { message: message.value })
    agent.value = res.agent; note.value = res.note || ''; actions.value = res.actions || []
  } catch (e) { err.value = e.message } finally { loading.value = false }
}

async function execute(a) {
  err.value = ''
  try {
    result.value = await api.post('/copilot/execute', { kind: a.Kind, target: a.Target, params: a.Params })
  } catch (e) { err.value = e.message }
}
</script>
