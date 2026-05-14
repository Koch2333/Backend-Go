<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MODULES } from './modules'

const route = useRoute()
const router = useRouter()

const activeModule = computed(() => {
  const m = route.matched.find((r) => r.meta?.moduleName)?.meta?.moduleName as
    | string
    | undefined
  return m ? MODULES.find((x) => x.name === m) : null
})

// Map legacy Vant icon names to Material Symbols.
const ICON_MAP: Record<string, string> = {
  'link-o': 'link',
  link: 'link',
  'credit-pay': 'credit_card',
  'shield-o': 'shield',
  'medal-o': 'workspace_premium',
  'photo-o': 'photo',
  edit: 'edit',
  'user-o': 'person',
  'setting-o': 'settings',
  description: 'description',
  'description-o': 'description',
  'delete-o': 'delete',
}
function mdIcon(name?: string): string {
  if (!name) return ''
  return ICON_MAP[name] ?? name
}

function logoutCurrent() {
  if (!activeModule.value) return
  try {
    localStorage.removeItem(`roast.admin.${activeModule.value.name}.auth`)
  } catch {
    /* */
  }
  router.replace(`/m/${activeModule.value.name}/login`)
}
</script>

<template>
  <div class="flex min-h-screen">
    <aside class="w-52 shrink-0 border-r bg-white">
      <div class="px-4 py-3 text-base font-semibold text-gray-800">Backend Admin</div>
      <nav class="py-1">
        <router-link
          to="/"
          class="block px-4 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
          active-class="bg-brand-50 text-brand-700"
          :exact="true"
        >
          总览
        </router-link>

        <div v-for="m in MODULES" :key="m.name" class="mt-2">
          <p class="px-4 pb-1 text-[11px] uppercase tracking-wide text-gray-400">
            {{ m.title }}
          </p>
          <router-link
            v-for="n in (m.nav ?? [])"
            :key="n.to"
            :to="n.to"
            class="flex items-center gap-2 px-4 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
            active-class="bg-brand-50 text-brand-700"
          >
            <md-icon v-if="n.icon" class="nav-icon">{{ mdIcon(n.icon) }}</md-icon>
            <span>{{ n.label }}</span>
          </router-link>
        </div>
      </nav>
    </aside>

    <div class="flex flex-1 flex-col">
      <header class="flex items-center justify-between border-b bg-white px-6 py-3">
        <div class="text-sm text-gray-600">
          <span v-if="activeModule" class="font-medium text-gray-800">{{ activeModule.title }}</span>
          <span v-else class="font-medium text-gray-800">总览</span>
        </div>
        <md-outlined-button v-if="activeModule" @click="logoutCurrent">
          退出 {{ activeModule.title }}
        </md-outlined-button>
      </header>
      <main class="flex-1 overflow-y-auto px-6 py-5">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.nav-icon {
  --md-icon-size: 18px;
  font-size: 18px;
}
</style>
