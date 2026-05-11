// https://nuxt.com/docs/api/configuration/nuxt-config
declare const process: { env: Record<string, string | undefined> }

export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  // SPA: OIDC client lives in the browser only, and we don't need SSR for an internal tool.
  ssr: false,
  devtools: { enabled: true },
  modules: ['@nuxtjs/tailwindcss', '@pinia/nuxt'],
  runtimeConfig: {
    public: {
      // In dev, the frontend runs on :3000 and the backend on :8080 — set
      // NUXT_PUBLIC_API_BASE explicitly. In production we ship a single image
      // that serves both API and SPA, so an empty base resolves to same-origin.
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '',
      // Fallback OIDC config baked at build time (used in dev). In production
      // the SPA fetches /api/public/config at startup and overrides these.
      oidcIssuer: process.env.NUXT_PUBLIC_OIDC_ISSUER || '',
      oidcClientId: process.env.NUXT_PUBLIC_OIDC_CLIENT_ID || '',
    },
  },
})
