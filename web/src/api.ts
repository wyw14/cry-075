const actor = { 'X-Actor-ID': 'demo_operator', 'X-Actor-Role': 'operator' }
const apiBase = String(import.meta.env.VITE_API_BASE || '/api/v1').replace(/\/$/, '')

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const endpoint = path.startsWith('/') ? path : `/${path}`
  const response = await fetch(`${apiBase}${endpoint}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...actor, ...init.headers }
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }))
    throw new Error(error.message || '请求失败')
  }
  return response.json() as Promise<T>
}
