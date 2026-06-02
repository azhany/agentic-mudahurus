<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-2xl font-bold text-gray-800">{{ t('nav.products') }}</h1>
      <button class="btn-primary" @click="openCreate">+ {{ t('common.create') }}</button>
    </div>
    <input v-model="search" class="input max-w-xs mb-4" :placeholder="t('common.search')" @input="debouncedLoad" />

    <div class="card overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-left text-gray-500 border-b">
          <tr><th class="py-2">SKU</th><th>{{ t('nav.products') }}</th><th>{{ t('nav.categories') }}</th><th>Price</th><th>Status</th><th>{{ t('common.actions') }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="p in items" :key="p.id" class="border-b last:border-0">
            <td class="py-2 font-mono">{{ p.sku }}</td>
            <td>{{ p.product_name }}</td>
            <td>{{ p.category }}</td>
            <td>RM {{ Number(p.unit_price).toFixed(2) }}</td>
            <td><span class="badge" :class="p.status==='active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">{{ p.status }}</span></td>
            <td class="space-x-1">
              <button class="btn-ghost" @click="openEdit(p)">{{ t('common.edit') }}</button>
              <button class="btn-danger" @click="remove(p)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!items.length"><td colspan="6" class="py-6 text-center text-gray-400">{{ t('common.empty') }}</td></tr>
        </tbody>
      </table>
    </div>

    <Modal v-if="editing" @close="editing=null">
      <h2 class="text-lg font-semibold mb-3">{{ form.id ? t('common.edit') : t('common.create') }}</h2>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="label">SKU</label><input v-model="form.sku" class="input" /></div>
        <div><label class="label">Name</label><input v-model="form.product_name" class="input" /></div>
        <div><label class="label">Price</label><input v-model.number="form.unit_price" type="number" step="0.01" class="input" /></div>
        <div>
          <label class="label">Category</label>
          <select v-model="form.category_id" class="input">
            <option value="">—</option>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>
        <div class="col-span-2"><label class="label">Description</label><textarea v-model="form.description" class="input" rows="2"></textarea></div>
        <div>
          <label class="label">Status</label>
          <select v-model="form.status" class="input"><option value="active">active</option><option value="inactive">inactive</option></select>
        </div>
        <div v-if="form.id"><label class="label">Image</label><input type="file" accept="image/*" @change="onFile" /></div>
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
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import Modal from '../components/Modal.vue'

const { t } = useI18n()
const items = ref([]), categories = ref([]), editing = ref(null), err = ref(''), search = ref('')
const form = reactive({})

let timer
function debouncedLoad() { clearTimeout(timer); timer = setTimeout(load, 300) }

async function load() {
  const res = await api.get('/products?search=' + encodeURIComponent(search.value))
  items.value = res.records || []
}
async function loadCategories() {
  const res = await api.get('/categories')
  categories.value = res.records || []
}
function openCreate() { Object.assign(form, { id: '', sku: '', product_name: '', unit_price: 0, category_id: '', description: '', status: 'active' }); err.value=''; editing.value = true }
function openEdit(p) { Object.assign(form, { ...p, category_id: p.category_id || '' }); err.value=''; editing.value = true }
async function save() {
  err.value = ''
  try {
    if (form.id) await api.put('/products/' + form.id, form)
    else await api.post('/products', form)
    editing.value = null
    await load()
  } catch (e) { err.value = e.message }
}
async function remove(p) {
  if (!confirm('Delete ' + p.product_name + '?')) return
  await api.del('/products/' + p.id); await load()
}
async function onFile(e) {
  const file = e.target.files[0]; if (!file || !form.id) return
  const fd = new FormData(); fd.append('image', file)
  try { await api.upload('/products/' + form.id + '/image', fd); await load() } catch (e) { err.value = e.message }
}
onMounted(() => { load(); loadCategories() })
</script>
