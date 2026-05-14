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
  <div class="space-y-3 p-2">
    <div class="flex items-center gap-2">
      <md-outlined-text-field
        label="按 id / 标题 / 系列搜索"
        :value="q"
        @input="(e: any) => (q = e.target.value)"
        @keyup.enter="load"
        class="flex-1"
      >
        <md-icon slot="leading-icon">search</md-icon>
      </md-outlined-text-field>
      <md-filled-button @click="router.push('/m/roundnfc/badges/new')">
        <md-icon slot="icon">add</md-icon>
        新建
      </md-filled-button>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">
      <md-circular-progress indeterminate aria-label="加载中" />
    </div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-400">
      还没有徽章
    </div>

    <md-list v-else class="m3-card rounded-2xl bg-white">
      <template v-for="(b, i) in items" :key="b.id">
        <md-divider v-if="i > 0" />
        <md-list-item
          type="button"
          @click="router.push(`/m/roundnfc/badges/${encodeURIComponent(b.id)}`)"
        >
          <div slot="headline">{{ b.title || '(未命名)' }}</div>
          <div slot="supporting-text">{{ b.id }}　·　{{ b.series || '—' }}</div>
          <div slot="end">
            <md-icon-button aria-label="删除" @click.stop="onDelete(b)">
              <md-icon>delete</md-icon>
            </md-icon-button>
          </div>
        </md-list-item>
      </template>
    </md-list>

    <p class="pt-1 text-center text-xs text-gray-400">共 {{ total }} 条</p>
  </div>
</template>

<style scoped>
md-outlined-text-field {
  width: 100%;
}
md-list {
  --md-list-container-color: #fff;
}
</style>
