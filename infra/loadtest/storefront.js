// k6 load test for storefront read paths (MH-605). Asserts NFR p95 targets.
//   k6 run -e BASE=http://localhost:8080 -e STORE=kedaiali infra/loadtest/storefront.js
import http from 'k6/http'
import { check } from 'k6'

const BASE = __ENV.BASE || 'http://localhost:8080'
const STORE = __ENV.STORE || 'kedaiali'

export const options = {
  vus: 20,
  duration: '30s',
  thresholds: {
    // NFR: storefront/API p95 < 250ms (non-RAG).
    http_req_duration: ['p(95)<250'],
    checks: ['rate>0.99'],
  },
}

export default function () {
  const landing = http.get(`${BASE}/store/${STORE}`)
  check(landing, { 'landing 200': (r) => r.status === 200 })
  const search = http.get(`${BASE}/store/${STORE}/search?q=kuih`)
  check(search, { 'search 200': (r) => r.status === 200 })
}
