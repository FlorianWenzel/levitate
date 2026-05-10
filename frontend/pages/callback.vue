<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: false })

const auth = useAuthStore()
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    await auth.handleCallback()
    await navigateTo('/')
  } catch (e: any) {
    error.value = e?.message ?? String(e)
  }
})
</script>

<template>
  <main class="min-h-screen flex items-center justify-center bg-slate-50">
    <div class="text-center">
      <p v-if="!error" class="text-slate-600">Signing you in…</p>
      <div v-else class="space-y-3">
        <p class="text-red-600">Sign-in failed.</p>
        <p class="text-xs text-slate-500 font-mono">{{ error }}</p>
        <NuxtLink to="/login" class="text-sm text-slate-700 underline">Try again</NuxtLink>
      </div>
    </div>
  </main>
</template>
