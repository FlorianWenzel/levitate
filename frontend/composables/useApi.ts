import type { FetchOptions } from 'ofetch'
import { useAuthStore } from '~/stores/auth'

// useApi returns a thin $fetch wrapper that injects the bearer token and the API base URL.
export function useApi() {
  const config = useRuntimeConfig()
  const auth = useAuthStore()

  async function call<T = unknown>(path: string, opts: FetchOptions = {}): Promise<T> {
    const token = auth.accessToken
    const headers: Record<string, string> = {
      ...(opts.headers as Record<string, string> | undefined),
    }
    if (token) headers.Authorization = `Bearer ${token}`
    return await $fetch<T>(`${config.public.apiBase}${path}`, {
      ...opts,
      headers,
    } as FetchOptions<'json'>) as T
  }

  return { call }
}
