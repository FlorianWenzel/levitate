import { defineStore } from 'pinia'
import type { Role, RoleInput } from '~/types/api'

export const useRolesStore = defineStore('roles', {
  state: () => ({
    roles: [] as Role[],
    loading: false,
    error: null as string | null,
  }),
  getters: {
    byId: (s) => (id: string) => s.roles.find((r) => r.id === id) ?? null,
  },
  actions: {
    async fetch() {
      const { call } = useApi()
      this.loading = true
      this.error = null
      try {
        this.roles = await call<Role[]>('/api/roles')
      } catch (e: any) {
        this.error = e?.data?.detail ?? e?.message ?? 'Failed to load roles'
        throw e
      } finally {
        this.loading = false
      }
    },
    async create(input: RoleInput): Promise<Role> {
      const { call } = useApi()
      const role = await call<Role>('/api/roles', { method: 'POST', body: input })
      this.roles = [...this.roles, role].sort((a, b) => a.name.localeCompare(b.name))
      return role
    },
    async update(id: string, input: RoleInput): Promise<Role> {
      const { call } = useApi()
      const role = await call<Role>(`/api/roles/${id}`, { method: 'PATCH', body: input })
      this.roles = this.roles
        .map((r) => (r.id === id ? role : r))
        .sort((a, b) => a.name.localeCompare(b.name))
      return role
    },
    async remove(id: string): Promise<void> {
      const { call } = useApi()
      await call(`/api/roles/${id}`, { method: 'DELETE' })
      this.roles = this.roles.filter((r) => r.id !== id)
    },
  },
})
