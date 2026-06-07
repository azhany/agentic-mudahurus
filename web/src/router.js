import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const routes = [
  { path: '/', redirect: '/admin/dashboard' },
  { path: '/login', component: () => import('./views/Login.vue'), meta: { public: true } },
  { path: '/register', component: () => import('./views/Register.vue'), meta: { public: true } },
  {
    path: '/admin',
    component: () => import('./views/AdminLayout.vue'),
    children: [
      { path: 'dashboard', component: () => import('./views/Dashboard.vue') },
      { path: 'products', component: () => import('./views/Products.vue') },
      { path: 'categories', component: () => import('./views/Categories.vue') },
      { path: 'orders', component: () => import('./views/Orders.vue') },
      { path: 'customers', component: () => import('./views/Customers.vue') },
      { path: 'coupons', component: () => import('./views/Coupons.vue') },
      { path: 'copilot', component: () => import('./views/Copilot.vue') },
    ],
  },
  // Public storefront
  { path: '/store/:username', component: () => import('./views/Storefront.vue'), meta: { public: true } },
  { path: '/store/:username/product/:id', component: () => import('./views/ProductDetail.vue'), meta: { public: true } },
  { path: '/invoice/:id', component: () => import('./views/Invoice.vue'), meta: { public: true } },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  const auth = useAuthStore()
  if (!auth.isAuthenticated) return { path: '/login', query: { redirect: to.fullPath } }
  return true
})
