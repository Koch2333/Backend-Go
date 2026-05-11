<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showFailToast } from 'vant'
import { extractMessage } from '@/shell/http'
import { listPhotoRequests, setPhotoStatus } from '../api'
import type { PhotoRequest, RequestStatus } from '../types'

const items = ref<PhotoRequest[]>([])
const total = ref(0)
const status = ref<RequestStatus | ''>('')
const badgeId = ref('')
const loading = ref(false)

const statusOptions = [
  { text: '全部', value: '' },
  { text: '待处理', value: 'new' },
  { text: '已处理', value: 'handled' },
  { text: '已拒绝', value: 'rejected' },
]

async function load() {
  loading.value = true
  try {
    const res = await listPhotoRequests({
      status: (status.value || undefined) as RequestStatus | undefined,
      badgeId: badgeId.value || undefined,
      limit: 100,
    })
    items.value = res.items ?? []
    total.value = res.total ?? 0
  } catch (err) {
    showFailToast(extractMessage(err, '加载失败'))
  } finally {
    loading.value = false
  }
}

async function update(r: PhotoRequest, next: RequestStatus) {
  try {
    await setPhotoStatus(r.id, next)
    r.status = next
  } catch (err) {
    showFailToast(extractMessage(err, '更新失败'))
  }
}

function tagType(s: RequestStatus) {
  return s === 'handled' ? 'success' : s === 'rejected' ? 'danger' : 'warning'
}
function tagText(s: RequestStatus) {
  return s === 'handled' ? '已处理' : s === 'rejected' ? '已拒绝' : '待处理'
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center gap-2">
      <van-field
        v-model="badgeId"
        placeholder="按徽章 ID 过滤"
        class="!flex-1"
        @blur="load"
        @keyup.enter="load"
      />
      <van-dropdown-menu>
        <van-dropdown-item v-model="status" :options="statusOptions" @change="load" />
      </van-dropdown-menu>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-400">没有记录</div>

    <van-cell-group v-else inset class="mt-3">
      <van-cell v-for="r in items" :key="r.id">
        <template #title>
          <div class="space-y-1">
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900">{{ r.name }}</span>
              <van-tag size="medium" :type="tagType(r.status)">{{ tagText(r.status) }}</van-tag>
            </div>
            <div class="text-xs text-gray-500">{{ r.contact }}　·　{{ r.badgeId }}</div>
            <div v-if="r.message" class="text-xs leading-relaxed text-gray-700">{{ r.message }}</div>
            <div class="text-[11px] text-gray-300">{{ new Date(r.createdAt).toLocaleString() }}</div>
          </div>
        </template>
        <template #right-icon>
          <div class="ml-3 flex flex-col gap-1">
            <van-button v-if="r.status !== 'handled'" size="mini" type="success" @click="update(r, 'handled')">已处理</van-button>
            <van-button v-if="r.status !== 'rejected'" size="mini" type="danger" plain @click="update(r, 'rejected')">拒绝</van-button>
          </div>
        </template>
      </van-cell>
    </van-cell-group>

    <p class="pt-3 text-center text-xs text-gray-400">共 {{ total }} 条</p>
  </div>
</template>
