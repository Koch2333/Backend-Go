<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MODULES } from './modules'
import { theme, toggleTheme } from './themeToggle'

const route = useRoute()
const router = useRouter()

const activeModule = computed(() => {
  const m = route.matched.find((r) => r.meta?.moduleName)?.meta?.moduleName as
    | string
    | undefined
  return m ? MODULES.find((x) => x.name === m) : null
})

const currentNav = computed(() => {
  if (!activeModule.value?.nav) return null
  return activeModule.value.nav.find((n) => n.to === route.path) ?? null
})

const pageTitle = computed(() => {
  if (!activeModule.value) return '总览'
  return currentNav.value?.label
    ? `${activeModule.value.title} · ${currentNav.value.label}`
    : activeModule.value.title
})

// Legacy Vant icon names → Material Symbols.
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
  if (!name) return 'apps'
  return ICON_MAP[name] ?? name
}

// Flat rail entries: dashboard + every module's nav items.
const railItems = computed(() => {
  const items: { to: string; label: string; icon: string }[] = [
    { to: '/', label: '总览', icon: 'space_dashboard' },
  ]
  for (const m of MODULES) {
    for (const n of m.nav ?? []) {
      items.push({ to: n.to, label: n.label, icon: mdIcon(n.icon) })
    }
  }
  return items
})

function isActive(to: string) {
  if (to === '/') return route.path === '/'
  return route.path === to || route.path.startsWith(to + '/')
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

// Responsive: rail on desktop, top bar + drawer on narrow.
const isNarrow = ref(false)
const drawerOpen = ref(false)
function updateNarrow() { isNarrow.value = window.matchMedia('(max-width: 767px)').matches }
onMounted(() => {
  updateNarrow()
  window.addEventListener('resize', updateNarrow)
})
onBeforeUnmount(() => window.removeEventListener('resize', updateNarrow))

function go(to: string) {
  drawerOpen.value = false
  if (route.path !== to) router.push(to)
}
</script>

<template>
  <div class="layout">
    <!-- Desktop navigation rail -->
    <aside v-if="!isNarrow" class="rail">
      <div class="rail-brand">
        <md-icon class="rail-brand-icon">grid_view</md-icon>
      </div>
      <nav class="rail-nav">
        <button
          v-for="item in railItems"
          :key="item.to"
          type="button"
          class="rail-item"
          :class="{ active: isActive(item.to) }"
          @click="go(item.to)"
        >
          <span class="rail-pill">
            <md-icon>{{ item.icon }}</md-icon>
            <md-ripple />
          </span>
          <span class="rail-label">{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <!-- Mobile drawer -->
    <div v-if="isNarrow && drawerOpen" class="drawer-scrim" @click="drawerOpen = false" />
    <aside v-if="isNarrow" class="drawer" :class="{ open: drawerOpen }">
      <div class="drawer-head m3-title-large">Backend Admin</div>
      <nav class="drawer-nav">
        <button
          v-for="item in railItems"
          :key="item.to"
          type="button"
          class="drawer-item"
          :class="{ active: isActive(item.to) }"
          @click="go(item.to)"
        >
          <md-icon>{{ item.icon }}</md-icon>
          <span class="m3-label-large">{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <div class="main">
      <header class="topbar">
        <div class="topbar-left">
          <md-icon-button v-if="isNarrow" aria-label="菜单" @click="drawerOpen = !drawerOpen">
            <md-icon>menu</md-icon>
          </md-icon-button>
          <span class="m3-title-large topbar-title">{{ pageTitle }}</span>
        </div>
        <div class="topbar-actions">
          <md-icon-button
            :aria-label="theme === 'dark' ? '切换浅色主题' : '切换深色主题'"
            @click="toggleTheme"
          >
            <md-icon>{{ theme === 'dark' ? 'light_mode' : 'dark_mode' }}</md-icon>
          </md-icon-button>
          <md-outlined-button v-if="activeModule" @click="logoutCurrent">
            <md-icon slot="icon">logout</md-icon>
            退出
          </md-outlined-button>
        </div>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
  background: var(--md-sys-color-background);
  color: var(--md-sys-color-on-background);
}

/* ===== Navigation rail ===== */
.rail {
  width: 88px;
  flex-shrink: 0;
  background: var(--md-sys-color-surface);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 12px 0;
  border-right: 1px solid var(--md-sys-color-outline-variant);
  position: sticky;
  top: 0;
  height: 100vh;
}
.rail-brand {
  display: grid;
  place-items: center;
  height: 56px;
  margin-bottom: 8px;
}
.rail-brand-icon {
  color: var(--md-sys-color-primary);
  --md-icon-size: 28px;
  font-size: 28px;
}
.rail-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 0;
  overflow-y: auto;
}
.rail-item {
  background: transparent;
  border: none;
  padding: 6px 0 10px;
  cursor: pointer;
  color: var(--md-sys-color-on-surface-variant);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  transition: color 200ms var(--md-easing-standard);
}
.rail-pill {
  position: relative;
  width: 56px;
  height: 32px;
  border-radius: 16px;
  display: grid;
  place-items: center;
  overflow: hidden;
  transition: background-color 200ms var(--md-easing-standard);
}
.rail-item:hover .rail-pill {
  background: color-mix(in srgb, var(--md-sys-color-on-surface) 8%, transparent);
}
.rail-item.active {
  color: var(--md-sys-color-on-secondary-container);
}
.rail-item.active .rail-pill {
  background: var(--md-sys-color-secondary-container);
}
.rail-label {
  font: 500 11px/14px 'Roboto', system-ui, sans-serif;
  letter-spacing: 0.5px;
  max-width: 80px;
  text-align: center;
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
  padding: 0 4px;
}

/* ===== Mobile drawer ===== */
.drawer {
  position: fixed;
  inset: 0 auto 0 0;
  width: 280px;
  background: var(--md-sys-color-surface-container-low);
  color: var(--md-sys-color-on-surface);
  z-index: 40;
  transform: translateX(-100%);
  transition: transform 240ms var(--md-easing-standard);
  display: flex;
  flex-direction: column;
  padding: 16px 12px;
  gap: 8px;
}
.drawer.open { transform: translateX(0); }
.drawer-scrim {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 35;
}
.drawer-head { padding: 8px 16px 12px; }
.drawer-nav { display: flex; flex-direction: column; gap: 4px; }
.drawer-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  text-align: left;
  border-radius: 28px;
  color: var(--md-sys-color-on-surface-variant);
  cursor: pointer;
  transition: background-color 200ms var(--md-easing-standard);
}
.drawer-item:hover {
  background: color-mix(in srgb, var(--md-sys-color-on-surface) 8%, transparent);
}
.drawer-item.active {
  background: var(--md-sys-color-secondary-container);
  color: var(--md-sys-color-on-secondary-container);
}

/* ===== Main column ===== */
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 24px;
  background: var(--md-sys-color-surface);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  min-height: 64px;
}
.topbar-left { display: flex; align-items: center; gap: 8px; min-width: 0; }
.topbar-title {
  color: var(--md-sys-color-on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.topbar-actions { display: flex; align-items: center; gap: 8px; }
.content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: var(--md-sys-color-background);
}

@media (max-width: 767px) {
  .content { padding: 16px; }
  .topbar { padding: 10px 12px; }
}
</style>
