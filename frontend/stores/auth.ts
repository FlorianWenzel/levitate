import { defineStore } from 'pinia'
import { UserManager, WebStorageStateStore, type User } from 'oidc-client-ts'

export type Me = {
  sub: string
  email: string
  name: string
  roles: string[]
}

let manager: UserManager | null = null

function getManager(): UserManager {
  if (manager) return manager
  const config = useRuntimeConfig()
  const runtime = (window as any).__APP_CONFIG__ as { oidcIssuer?: string; oidcClientId?: string } | undefined
  const issuer = runtime?.oidcIssuer || (config.public.oidcIssuer as string)
  const clientId = runtime?.oidcClientId || (config.public.oidcClientId as string)
  const origin = window.location.origin
  manager = new UserManager({
    authority: issuer,
    client_id: clientId,
    redirect_uri: `${origin}/callback`,
    post_logout_redirect_uri: `${origin}/`,
    response_type: 'code',
    scope: 'openid profile email',
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
  })
  return manager
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    me: null as Me | null,
    ready: false,
  }),
  getters: {
    isAuthenticated: (s) => !!s.user && !s.user.expired,
    accessToken: (s) => s.user?.access_token ?? null,
    isAdmin: (s) => !!s.me?.roles?.includes('admin'),
  },
  actions: {
    async init() {
      if (this.ready) return
      const m = getManager()
      const existing = await m.getUser()
      if (existing && !existing.expired) {
        this.user = existing
      }
      this.ready = true
    },
    async login() {
      await getManager().signinRedirect()
    },
    async handleCallback() {
      const u = await getManager().signinRedirectCallback()
      this.user = u
    },
    async logout() {
      this.me = null
      try {
        await getManager().signoutRedirect()
      } catch {
        await getManager().removeUser()
        this.user = null
      }
    },
    async fetchMe() {
      if (!this.accessToken) return null
      const config = useRuntimeConfig()
      const me = await $fetch<Me>(`${config.public.apiBase}/api/me`, {
        headers: { Authorization: `Bearer ${this.accessToken}` },
      })
      this.me = me
      return me
    },
  },
})
