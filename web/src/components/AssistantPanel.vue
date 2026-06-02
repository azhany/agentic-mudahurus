<template>
  <div class="card mb-6">
    <div class="flex items-center gap-2">
      <input v-model="q" class="input" :placeholder="t('store.ask')" @keyup.enter="ask" />
      <button class="btn-primary" :disabled="loading" @click="ask">🔎</button>
    </div>
    <div v-if="answer" class="mt-3 text-sm">
      <p :class="answer.refused ? 'text-amber-600' : 'text-gray-800'">{{ answer.answer }}</p>
      <div v-if="answer.citations?.length" class="mt-2 flex flex-wrap gap-1">
        <span v-for="(c, i) in answer.citations" :key="i" class="badge bg-gray-100 text-gray-600">
          {{ c.source_type }}:{{ c.source_id.slice(0, 8) }} ({{ c.score }})
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'

const { t } = useI18n()
const q = ref('')
const answer = ref(null)
const loading = ref(false)

async function ask() {
  if (!q.value.trim()) return
  loading.value = true
  try {
    answer.value = await api.post('/assistant/search', { question: q.value })
  } catch (e) {
    answer.value = { answer: e.message, refused: true, citations: [] }
  } finally {
    loading.value = false
  }
}
</script>
