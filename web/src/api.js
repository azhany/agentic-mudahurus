// Central API client with a token-refresh interceptor (MH-205).
// On 401 it transparently refreshes the access token using the refresh token
// and retries the request once.
import { useAuthStore } from './stores/auth'

const BASE = import.meta.env.VITE_API_BASE_URL || '/api'

async function request(method, path, { body, isForm, retry = true } = {}) {
  const auth = useAuthStore()
  const headers = {}
  if (!isForm) headers['Content-Type'] = 'application/json'
  if (auth.accessToken) headers['Authorization'] = `Bearer ${auth.accessToken}`

  const res = await fetch(BASE + path, {
    method,
    headers,
    body: isForm ? body : body != null ? JSON.stringify(body) : undefined,
  })

  if (res.status === 401 && retry && auth.refreshToken) {
    const ok = await auth.refresh()
    if (ok) return request(method, path, { body, isForm, retry: false })
    auth.logoutLocal()
  }

  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    const err = new Error(data?.error?.message || res.statusText)
    err.status = res.status
    err.fields = data?.error?.fields
    throw err
  }
  return data
}

export const api = {
  get: (p) => request('GET', p),
  post: (p, body) => request('POST', p, { body }),
  put: (p, body) => request('PUT', p, { body }),
  patch: (p, body) => request('PATCH', p, { body }),
  del: (p) => request('DELETE', p),
  upload: (p, formData) => request('POST', p, { body: formData, isForm: true }),
}
