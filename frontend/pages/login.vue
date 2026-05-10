<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: false })

const auth = useAuthStore()

onMounted(async () => {
  await auth.init()
  if (auth.isAuthenticated) {
    navigateTo('/')
  }
})
</script>

<template>
  <main class="min-h-screen flex items-center justify-center bg-slate-50">
    <div class="text-center space-y-6">
      <div>
        <h1 class="text-5xl font-semibold tracking-tight text-slate-900">Levitate</h1>
        <p class="mt-2 text-slate-600">They float. We levitate.</p>
      </div>
      <button
        class="rounded bg-slate-900 px-5 py-2.5 text-sm font-medium text-white hover:bg-slate-700"
        @click="auth.login()"
      >
        Sign in with Keycloak
      </button>
    </div>
  </main>
</template>
