import { useAuthStore } from '~/stores/auth'

export default defineNuxtRouteMiddleware(async (to) => {
  // Public routes — no auth required.
  const publicPaths = ['/login', '/callback', '/silent-renew']
  if (publicPaths.includes(to.path)) return

  const auth = useAuthStore()
  await auth.init()

  if (!auth.isAuthenticated) {
    return navigateTo('/login')
  }
  if (!auth.me) {
    try {
      await auth.fetchMe()
    } catch {
      // /api/me failure shouldn't block the page — admin gates fail closed.
    }
  }
})
