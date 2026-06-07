// BM-first i18n with English secondary (NFR i18n, parity with legacy BM UI).
import { createI18n } from 'vue-i18n'

const messages = {
  ms: {
    nav: { dashboard: 'Papan Pemuka', products: 'Produk', categories: 'Kategori', orders: 'Pesanan', customers: 'Pelanggan', coupons: 'Kupon', copilot: 'Copilot', logout: 'Log Keluar' },
    login: { title: 'Log Masuk', username: 'Nama Pengguna', password: 'Kata Laluan', submit: 'Log Masuk', register: 'Daftar Akaun' },
    dash: { orders: 'Pesanan', pending: 'Belum Selesai', products: 'Produk', customers: 'Pelanggan', shipped: 'Dihantar' },
    common: { save: 'Simpan', cancel: 'Batal', delete: 'Padam', edit: 'Sunting', create: 'Tambah', search: 'Cari', loading: 'Memuatkan…', empty: 'Tiada rekod', actions: 'Tindakan' },
    store: { addToCart: 'Tambah ke Troli', checkout: 'Buat Pesanan', search: 'Cari produk…', ask: 'Tanya tentang produk' },
  },
  en: {
    nav: { dashboard: 'Dashboard', products: 'Products', categories: 'Categories', orders: 'Orders', customers: 'Customers', coupons: 'Coupons', copilot: 'Copilot', logout: 'Log out' },
    login: { title: 'Sign in', username: 'Username', password: 'Password', submit: 'Sign in', register: 'Create account' },
    dash: { orders: 'Orders', pending: 'Pending', products: 'Products', customers: 'Customers', shipped: 'Shipped' },
    common: { save: 'Save', cancel: 'Cancel', delete: 'Delete', edit: 'Edit', create: 'Create', search: 'Search', loading: 'Loading…', empty: 'No records', actions: 'Actions' },
    store: { addToCart: 'Add to cart', checkout: 'Place order', search: 'Search products…', ask: 'Ask about products' },
  },
}

export const i18n = createI18n({
  legacy: false,
  locale: localStorage.getItem('mh_locale') || 'ms',
  fallbackLocale: 'en',
  messages,
})
