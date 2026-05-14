<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showFailToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { listPhotoRequests, setPhotoStatus } from '../api'
import type { PhotoRequest, RequestStatus } from '../types'

const items = ref<PhotoRequest[]>([])
const total = ref(0)
const status = ref<RequestStatus | ''>('')
const badgeId = ref('')
const loading = ref(false)

const statusOptions: { text: string; value: RequestStatus | '' }[] = [
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

function tagClass(s: RequestStatus) {
  return s === 'handled' ? 'chip-success' : s === 'rejected' ? 'chip-danger' : 'chip-warning'
}
function tagText(s: RequestStatus) {
  return s === 'handled' ? '已处理' : s === 'rejected' ? '已拒绝' : '待处理'
}

onMounted(load)
</script>

<template>
  <div class="space-y-3 p-2">
    <div class="flex items-center gap-2">
      <md-outlined-text-field
        label="按徽章 ID 过滤"
        :value="badgeId"
        @input="(e: any) => (badgeId = e.target.value)"
        @keyup.enter="load"
        @blur="load"
        class="flex-1"
      />
      <select
        v-model="status"
        class="rounded-xl border border-gray-300 bg-white px-3 py-2 text-sm"
        @change="load"
      >
        <option v-for="o in statusOptions" :key="o.value" :value="o.value">{{ o.text }}</option>
      </select>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">
      <md-circular-progress indeterminate aria-label="加载中" />
    </div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-400">
      没有记录
    </div>

    <md-list v-else class="m3-card rounded-2xl bg-white">
      <template v-for="(r, i) in items" :key="r.id">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <div slot="headline" class="flex items-center gap-2">
            <span class="font-medium text-gray-900">{{ r.name }}</span>
            <md-assist-chip :label="tagText(r.status)" :class="tagClass(r.status)" />
          </div>
          <div slot="supporting-text" class="space-y-1">
            <div class="text-xs text-gray-500">{{ r.contact }}　·　{{ r.badgeId }}</div>
            <div v-if="r.message" class="text-xs leading-relaxed text-gray-700">{{ r.message }}</div>
            <div class="text-[11px] text-gray-300">{{ new Date(r.createdAt).toLocaleString() }}</div>
          </div>
          <div slot="end" class="ml-3 flex flex-col gap-1">
            <md-filled-button
              v-if="r.status !== 'handled'"
              @click="update(r, 'handled')"
            >
              已处理
            </md-filled-button>
            <md-outlined-button
              v-if="r.status !== 'rejected'"
              @click="update(r, 'rejected')"
            >
              拒绝
            </md-outlined-button>
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
.chip-success {
  --md-assist-chip-label-text-color: #146c43;
  --md-assist-chip-outline-color: #146c43;
}
.chip-warning {
  --md-assist-chip-label-text-color: #92642a;
  --md-assist-chip-outline-color: #92642a;
}
.chip-danger {
  --md-assist-chip-label-text-color: #b3261e;
  --md-assist-chip-outline-color: #b3261e;
}
</style>
