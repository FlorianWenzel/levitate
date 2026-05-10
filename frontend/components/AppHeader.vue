<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const route = useRoute()

const links = computed(() => [
  { to: '/', label: 'Dashboard' },
  { to: '/people', label: 'People' },
  { to: '/projects', label: 'Projects' },
  { to: '/schedule', label: 'Schedule' },
  { to: '/capacity', label: 'Capacity' },
  { to: '/reports', label: 'Reports' },
])

const isActive = (to: string) => route.path === to || (to !== '/' && route.path.startsWith(to))
</script>

<template>
  <header class="border-b border-slate-200 bg-white">
    <div class="mx-auto flex max-w-6xl items-center justify-between gap-6 px-6 py-3">
      <div class="flex items-center gap-8">
        <NuxtLink to="/" class="flex flex-col">
          <span class="text-base font-semibold text-slate-900">Levitate</span>
          <span class="text-[10px] uppercase tracking-wider text-slate-400">They float. We levitate.</span>
        </NuxtLink>
        <nav class="flex items-center gap-1 text-sm">
          <NuxtLink
            v-for="l in links"
            :key="l.to"
            :to="l.to"
            class="rounded px-2.5 py-1.5"
            :class="isActive(l.to)
              ? 'bg-slate-900 text-white'
              : 'text-slate-700 hover:bg-slate-100'"
          >
            {{ l.label }}
          </NuxtLink>
        </nav>
      </div>
      <div class="flex items-center gap-3 text-sm">
        <span v-if="auth.me" class="text-slate-700">
          {{ auth.me.email }}
          <span v-if="auth.me.roles?.length" class="ml-1 rounded bg-slate-200 px-1.5 py-0.5 text-xs">
            {{ auth.me.roles.join(', ') }}
          </span>
        </span>
        <button
          class="rounded bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700"
          @click="auth.logout()"
        >
          Sign out
        </button>
      </div>
    </div>
  </header>
</template>
