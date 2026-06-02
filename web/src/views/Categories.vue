<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-2xl font-bold text-gray-800">{{ t('nav.categories') }}</h1>
      <button class="btn-primary" @click="openCreate">+ {{ t('common.create') }}</button>
    </div>
    <div class="card">
      <table class="w-full text-sm">
        <thead class="text-left text-gray-500 border-b"><tr><th class="py-2">Name</th><th>Description</th><th>{{ t('common.actions') }}</th></tr></thead>
        <tbody>
          <tr v-for="c in items" :key="c.id" class="border-b last:border-0">
            <td class="py-2">{{ c.name }}</td><td>{{ c.description }}</td>
            <td class="space-x-1">
              <button class="btn-ghost" @click="openEdit(c)">{{ t('common.edit') }}</button>
              <button class="btn-danger" @click="remove(c)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!items.length"><td colspan="3" class="py-6 text-center text-gray-400">{{ t('common.empty') }}</td></tr>
        </tbody>
      </table>
    </div>
    <Modal v-if="editing" @close="editing=null">
      <h2 class="text-lg font-semibold mb-3">{{ form.id ? t('common.edit') : t('common.create') }}</h2>
      <label class="label">Name</label><input v-model="form.name" class="input mb-3" />
      <label class="label">Description</label><textarea v-model="form.description" class="input" rows="2"></textarea>
      <p v-if="err" class="text-sm text-red-600 mt-2">{{ err }}</p>
      <div class="flex justify-end gap-2 mt-4">
        <button class="btn-ghost" @click="editing=null">{{ t('common.cancel') }}</button>
        <button class="btn-primary" @click="save">{{ t('common.save') }}</button>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import Modal from '../components/Modal.vue'

const { t } = useI18n()
const items = ref([]), editing = ref(null), err = ref('')
const form = reactive({})
async function load() { const res = await api.get('/categories'); items.value = res.records || [] }
function openCreate() { Object.assign(form, { id: '', name: '', description: '' }); err.value=''; editing.value = true }
function openEdit(c) { Object.assign(form, c); err.value=''; editing.value = true }
async function save() {
  err.value=''
  try {
    if (form.id) await api.put('/categories/' + form.id, form)
    else await api.post('/categories', form)
    editing.value = null; await load()
  } catch (e) { err.value = e.message }
}
async function remove(c) { if (!confirm('Delete?')) return; await api.del('/categories/' + c.id); await load() }
onMounted(load)
</script>
