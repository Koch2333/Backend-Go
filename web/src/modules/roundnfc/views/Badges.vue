<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { showConfirmDialog, showFailToast } from 'vant'
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
  <div>
    <div class="flex items-center gap-2">
      <van-search
        v-model="q"
        class="!flex-1 !p-0"
        placeholder="按 id / 标题 / 系列搜索"
        shape="round"
        @search="load"
      />
      <van-button type="primary" size="small" @click="router.push('/m/roundnfc/badges/new')">
        新建
      </van-button>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-400">
      还没有徽章
    </div>

    <van-cell-group v-else inset class="mt-3">
      <van-cell
        v-for="b in items"
        :key="b.id"
        :title="b.title || '(未命名)'"
        :label="`${b.id}　·　${b.series || '—'}`"
        is-link
        @click="router.push(`/m/roundnfc/badges/${encodeURIComponent(b.id)}`)"
      >
        <template #right-icon>
          <van-icon
            name="delete-o"
            class="ml-2 text-base text-red-400"
            @click.stop="onDelete(b)"
          />
        </template>
      </van-cell>
    </van-cell-group>

    <p class="pt-3 text-center text-xs text-gray-400">共 {{ total }} 条</p>
  </div>
</template>
