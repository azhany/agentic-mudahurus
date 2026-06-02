# Web — Admin + Storefront SPA (Vue 3)

Vite + Vue 3 (Composition API) + Pinia + Vue Router + Tailwind + vue-i18n (BM/EN).

```bash
npm install
npm run dev      # http://localhost:5173 (proxies API to :8080)
npm run build    # production bundle -> dist/
```

- `src/views` admin screens (Dashboard, Products, Categories, Orders, Customers,
  Coupons) + public storefront (Storefront, ProductDetail, Invoice) + auth.
- `src/stores/auth.js` token storage + refresh; `src/api.js` refresh interceptor.
- `src/components/AssistantPanel.vue` grounded assistant UI (admin + storefront).
