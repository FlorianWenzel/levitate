// Fetches the backend's runtime config (OIDC issuer + client id) before the
// app finishes booting, so a single production binary can be reconfigured by
// changing env vars on the container — no SPA rebuild required.
//
// In dev, the values are also baked at build time via NUXT_PUBLIC_OIDC_*.
// Anything returned here takes precedence.

export default defineNuxtPlugin(async () => {
  const config = useRuntimeConfig()
  const apiBase = (config.public.apiBase as string) || ''
  try {
    const cfg = await $fetch<{ oidcIssuer: string; oidcClientId: string }>(
      `${apiBase}/api/public/config`,
    )
    ;(window as any).__APP_CONFIG__ = {
      oidcIssuer: cfg.oidcIssuer || (config.public.oidcIssuer as string),
      oidcClientId: cfg.oidcClientId || (config.public.oidcClientId as string),
    }
  } catch {
    // Backend not reachable — fall back to the build-time runtime config.
    ;(window as any).__APP_CONFIG__ = {
      oidcIssuer: config.public.oidcIssuer as string,
      oidcClientId: config.public.oidcClientId as string,
    }
  }
})
