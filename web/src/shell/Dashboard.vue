<script setup lang="ts">
import { MODULES } from './modules'

const ICONS: Record<string, string> = {
  redirect: 'link',
  roundnfc: 'workspace_premium',
}
function moduleIcon(name: string): string {
  return ICONS[name] ?? 'apps'
}
</script>

<template>
  <div>
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">后台总览</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">
          已装载 {{ MODULES.length }} 个模块，点击卡片进入。
        </p>
      </div>
    </header>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <router-link
        v-for="m in MODULES"
        :key="m.name"
        :to="(m.nav && m.nav[0]?.to) || `/m/${m.name}/login`"
        class="m3-card m3-card-interactive p-5 block no-underline"
      >
        <div class="flex items-start gap-4">
          <div class="tile-icon">
            <md-icon>{{ moduleIcon(m.name) }}</md-icon>
          </div>
          <div class="min-w-0 flex-1">
            <div class="m3-title-large text-on-surface truncate">{{ m.title }}</div>
            <div class="m3-body-small text-on-surface-variant mt-0.5 truncate">
              {{ m.name }} · {{ m.apiPrefix }}
            </div>
            <p v-if="m.description" class="m3-body-medium text-on-surface-variant mt-3">
              {{ m.description }}
            </p>
          </div>
        </div>
      </router-link>
    </div>

    <div v-if="MODULES.length === 0" class="m3-card m3-empty mt-8">
      <div class="m3-empty-icon"><md-icon>extension</md-icon></div>
      <div class="m3-title-medium text-on-surface">还没有模块</div>
      <div class="m3-body-medium text-on-surface-variant">
        复制 <code>src/modules/_template/</code> 过来是最快的起手方式。
      </div>
    </div>
  </div>
</template>

<style scoped>
.tile-icon {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 28px;
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
  display: grid;
  place-items: center;
}
.tile-icon md-icon {
  --md-icon-size: 28px;
  font-size: 28px;
}
</style>
