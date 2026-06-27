import { toast } from 'sonner'

const BASE = ''

export class ApiError extends Error {
  constructor(message, status, data) {
    super(message)
    this.status = status
    this.data = data
  }
}

function authHeaders(includeJson = false) {
  const t = localStorage.getItem('token')
  const h = { Authorization: `Bearer ${t}` }
  if (includeJson) h['Content-Type'] = 'application/json'
  return h
}

async function request(method, path, body) {
  const opts = { method, headers: authHeaders(!!body) }
  if (body) opts.body = JSON.stringify(body)
  const res = await fetch(BASE + path, opts)
  const json = await res.json()
  if (!res.ok) {
    throw new ApiError(json.error || res.statusText, res.status, json)
  }
  return json.data ?? json
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  patch: (path, body) => request('PATCH', path, body),
  put: (path, body) => request('PUT', path, body),
  del: (path) => request('DELETE', path),
  upload: async (path, file) => {
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(BASE + path, {
      method: 'POST',
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
      body: form,
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new ApiError(data.error || 'Upload failed', res.status, data)
    }
    return res.json()
  },
  download: async (path) => {
    const res = await fetch(BASE + path, {
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new ApiError(data.error || 'Download failed', res.status, data)
    }
    return res.blob()
  },
}

export function showToast(message, type = 'success', duration = 3000) {
  const m = { success: toast.success, error: toast.error, warning: toast.warning, info: toast.info }
  ;(m[type] || toast.info)(message, { duration })
}

export function cn(...classes) {
  return classes.filter(Boolean).join(' ')
}
