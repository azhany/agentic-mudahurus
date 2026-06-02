<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-2xl font-bold text-gray-800">{{ t('nav.customers') }}</h1>
      <button class="btn-primary" @click="openCreate">+ {{ t('common.create') }}</button>
    </div>
    <input v-model="search" class="input max-w-xs mb-4" :placeholder="t('common.search')" @input="debouncedLoad" />
    <div class="card overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-left text-gray-500 border-b"><tr><th class="py-2">Name</th><th>Loyalty</th><th>Email</th><th>Phone</th><th>{{ t('common.actions') }}</th></tr></thead>
        <tbody>
          <tr v-for="c in items" :key="c.id" class="border-b last:border-0">
            <td class="py-2">{{ c.full_name }}</td><td class="font-mono">{{ c.customer_loyalty_code }}</td>
            <td>{{ c.email }}</td><td>{{ c.contact_no }}</td>
            <td class="space-x-1">
              <button class="btn-ghost" @click="openEdit(c)">{{ t('common.edit') }}</button>
              <button class="btn-danger" @click="remove(c)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!items.length"><td colspan="5" class="py-6 text-center text-gray-400">{{ t('common.empty') }}</td></tr>
        </tbody>
      </table>
    </div>
    <Modal v-if="editing" @close="editing=null">
      <h2 class="text-lg font-semibold mb-3">{{ form.id ? t('common.edit') : t('common.create') }}</h2>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">Full name</label><input v-model="form.full_name" class="input" /></div>
        <div><label class="label">Loyalty code</label><input v-model="form.customer_loyalty_code" class="input" /></div>
        <div><label class="label">Email</label><input v-model="form.email" class="input" /></div>
        <div><label class="label">Phone</label><input v-model="form.contact_no" class="input" /></div>
        <div><label class="label">IC No (PII)</label><input v-model="form.ic_no" class="input" /></div>
        <div><label class="label">City</label><input v-model="form.city" class="input" /></div>
      </div>
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
const items = ref([]), editing = ref(null), err = ref(''), search = ref('')
const form = reactive({})
let timer
function debouncedLoad() { clearTimeout(timer); timer = setTimeout(load, 300) }
async function load() { const res = await api.get('/customers?search=' + encodeURIComponent(search.value)); items.value = res.records || [] }
function openCreate() { Object.assign(form, { id:'', full_name:'', customer_loyalty_code:'', email:'', contact_no:'', ic_no:'', city:'' }); err.value=''; editing.value = true }
function openEdit(c) { Object.assign(form, c); err.value=''; editing.value = true }
async function save() {
  err.value=''
  try { form.id ? await api.put('/customers/'+form.id, form) : await api.post('/customers', form); editing.value=null; await load() }
  catch (e) { err.value = e.message }
}
async function remove(c) { if (!confirm('Delete?')) return; await api.del('/customers/'+c.id); await load() }
load()
</script>
