<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const config = useRuntimeConfig()
const health = ref<string | null>(null)

onMounted(async () => {
  if (!auth.me) await auth.fetchMe()
})

async function pingBackend() {
  try {
    const res = await $fetch<{ status: string }>(`${config.public.apiBase}/healthz`)
    health.value = res.status
  } catch {
    health.value = 'unreachable'
  }
}
</script>

<template>
  <main class="mx-auto max-w-6xl px-6 py-12">
    <h1 class="text-3xl font-semibold tracking-tight text-slate-900">
      Hello, {{ auth.me?.name || auth.me?.email || 'there' }}
    </h1>
    <p class="mt-2 text-slate-600">
      Welcome to Levitate. Use the nav to manage People, Projects, and the Schedule.
    </p>
    <div class="mt-6 flex items-center gap-3">
      <button
        class="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
        @click="pingBackend"
      >
        Ping backend
      </button>
      <p v-if="health" class="text-sm text-slate-500">
        /healthz → <span class="font-mono">{{ health }}</span>
      </p>
    </div>
  </main>
</template>
