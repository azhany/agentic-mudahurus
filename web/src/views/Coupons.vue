<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-2xl font-bold text-gray-800">{{ t('nav.coupons') }}</h1>
      <button class="btn-primary" @click="openCreate">+ {{ t('common.create') }}</button>
    </div>
    <div class="card overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-left text-gray-500 border-b"><tr><th class="py-2">Campaign</th><th>Description</th><th>Expires</th><th>{{ t('common.actions') }}</th></tr></thead>
        <tbody>
          <tr v-for="c in items" :key="c.id" class="border-b last:border-0">
            <td class="py-2">{{ c.campaign }}</td><td>{{ c.description }}</td>
            <td><span :class="c.expired ? 'text-red-500' : ''">{{ c.expired_date ? new Date(c.expired_date).toLocaleDateString() : '—' }}</span></td>
            <td class="space-x-1">
              <button class="btn-ghost" @click="openEdit(c)">{{ t('common.edit') }}</button>
              <button class="btn-danger" @click="remove(c)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!items.length"><td colspan="4" class="py-6 text-center text-gray-400">{{ t('common.empty') }}</td></tr>
        </tbody>
      </table>
    </div>
    <Modal v-if="editing" @close="editing=null">
      <h2 class="text-lg font-semibold mb-3">{{ form.id ? t('common.edit') : t('common.create') }}</h2>
      <label class="label">Campaign</label><input v-model="form.campaign" class="input mb-3" />
      <label class="label">Description</label><textarea v-model="form.description" class="input mb-3" rows="2"></textarea>
      <label class="label">Expiry date</label><input v-model="form.expired_date" type="date" class="input" />
      <p v-if="err" class="text-sm text-red-600 mt-2">{{ err }}</p>
      <div class="flex justify-end gap-2 mt-4">
        <button class="btn-ghost" @click="editing=null">{{ t('common.cancel') }}</button>
        <button class="btn-primary" @click="save">{{ t('common.save') }}</button>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import Modal from '../components/Modal.vue'

const { t } = useI18n()
const items = ref([]), editing = ref(null), err = ref('')
const form = reactive({})
async function load() { const res = await api.get('/coupons'); items.value = res.records || [] }
function openCreate() { Object.assign(form, { id:'', campaign:'', description:'', expired_date:'' }); err.value=''; editing.value = true }
function openEdit(c) { Object.assign(form, { ...c, expired_date: c.expired_date ? c.expired_date.slice(0,10) : '' }); err.value=''; editing.value = true }
async function save() {
  err.value=''
  try { form.id ? await api.put('/coupons/'+form.id, form) : await api.post('/coupons', form); editing.value=null; await load() }
  catch (e) { err.value = e.message }
}
async function remove(c) { if (!confirm('Delete?')) return; await api.del('/coupons/'+c.id); await load() }
load()
</script>
