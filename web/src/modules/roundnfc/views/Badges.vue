<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { showFailToast } from '@/shell/toast'
import { showConfirmDialog } from '@/shell/confirm'
import { extractMessage } from '@/shell/http'
import { deleteBadge, listBadges } from '../api'
import type { Badge } from '../types'

const router = useRouter()
const items = ref<Badge[]>([])
const total = ref(0)
const q = ref('')
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await listBadges({ q: q.value, limit: 100 })
    items.value = res.items ?? []
    total.value = res.total ?? 0
  } catch (err) {
    showFailToast(extractMessage(err, '加载失败'))
  } finally {
    loading.value = false
  }
}

async function onDelete(b: Badge) {
  try {
    await showConfirmDialog({ title: '删除徽章', message: `确认删除 ${b.id}？` })
  } catch {
    return
  }
  try {
    await deleteBadge(b.id)
    items.value = items.value.filter((x) => x.id !== b.id)
  } catch (err) {
    showFailToast(extractMessage(err, '删除失败'))
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">徽章</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">共 {{ total }} 条</p>
      </div>
      <md-outlined-text-field
        label="按 id / 标题 / 系列搜索"
        :value="q"
        @input="(e: any) => (q = e.target.value)"
        @keyup.enter="load"
        class="search-input"
      >
        <md-icon slot="leading-icon">search</md-icon>
      </md-outlined-text-field>
    </header>

    <div v-if="loading" class="m3-loading">
      <md-circular-progress indeterminate aria-label="加载中" />
      <span class="m3-body-medium">加载中…</span>
    </div>

    <div v-else-if="items.length === 0" class="m3-card m3-empty">
      <div class="m3-empty-icon"><md-icon>workspace_premium</md-icon></div>
      <div class="m3-title-medium text-on-surface">还没有徽章</div>
      <div class="m3-body-medium text-on-surface-variant">
        点击右下角按钮创建第一枚徽章。
      </div>
    </div>

    <md-list v-else class="m3-card list-card">
      <template v-for="(b, i) in items" :key="b.id">
        <md-divider v-if="i > 0" />
        <md-list-item
          type="button"
          @click="router.push(`/m/roundnfc/badges/${encodeURIComponent(b.id)}`)"
        >
          <md-icon slot="start" class="row-icon">workspace_premium</md-icon>
          <div slot="headline" class="m3-title-medium">{{ b.title || '(未命名)' }}</div>
          <div slot="supporting-text" class="m3-body-medium">
            {{ b.id }} · {{ b.series || '—' }}
            <span v-if="b.coserBinding?.cn"> · CN: {{ b.coserBinding.cn }}</span>
          </div>
          <div slot="end">
            <md-icon-button aria-label="删除" @click.stop="onDelete(b)">
              <md-icon>delete</md-icon>
            </md-icon-button>
          </div>
        </md-list-item>
      </template>
    </md-list>

    <md-fab
      class="m3-fab"
      variant="primary"
      aria-label="新建徽章"
      @click="router.push('/m/roundnfc/badges/new')"
    >
      <md-icon slot="icon">add</md-icon>
    </md-fab>
  </div>
</template>

<style scoped>
.search-input { min-width: 240px; }
.list-card { padding: 4px 0; }
.row-icon { color: var(--md-sys-color-primary); }
</style>
