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
  return s === 'handled' ? 'chip-tertiary' : s === 'rejected' ? 'chip-danger' : 'chip-muted'
}
function tagText(s: RequestStatus) {
  return s === 'handled' ? '已处理' : s === 'rejected' ? '已拒绝' : '待处理'
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">返图申请</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">共 {{ total }} 条</p>
      </div>
      <div class="filter-bar">
        <md-outlined-text-field
          label="按徽章 ID 过滤"
          :value="badgeId"
          @input="(e: any) => (badgeId = e.target.value)"
          @keyup.enter="load"
          @blur="load"
          class="search-input"
        />
        <select v-model="status" class="status-select" @change="load">
          <option v-for="o in statusOptions" :key="o.value" :value="o.value">{{ o.text }}</option>
        </select>
      </div>
    </header>

    <div v-if="loading" class="m3-loading">
      <md-circular-progress indeterminate aria-label="加载中" />
      <span class="m3-body-medium">加载中…</span>
    </div>

    <div v-else-if="items.length === 0" class="m3-card m3-empty">
      <div class="m3-empty-icon"><md-icon>photo_library</md-icon></div>
      <div class="m3-title-medium text-on-surface">没有记录</div>
      <div class="m3-body-medium text-on-surface-variant">
        换个筛选条件再试试，或等待新的申请到达。
      </div>
    </div>

    <md-list v-else class="m3-card list-card">
      <template v-for="(r, i) in items" :key="r.id">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <div slot="headline" class="row-headline">
            <span class="m3-title-medium">{{ r.name }}</span>
            <md-assist-chip :label="tagText(r.status)" :class="tagClass(r.status)" />
          </div>
          <div slot="supporting-text" class="row-supporting">
            <div class="m3-body-small text-on-surface-variant">
              {{ r.contact }} · {{ r.badgeId }}
            </div>
            <div v-if="r.message" class="m3-body-medium text-on-surface">{{ r.message }}</div>
            <div class="m3-body-small text-on-surface-variant">
              {{ new Date(r.createdAt).toLocaleString() }}
            </div>
          </div>
          <div slot="end" class="row-actions">
            <md-filled-button v-if="r.status !== 'handled'" @click="update(r, 'handled')">
              已处理
            </md-filled-button>
            <md-outlined-button v-if="r.status !== 'rejected'" @click="update(r, 'rejected')">
              拒绝
            </md-outlined-button>
          </div>
        </md-list-item>
      </template>
    </md-list>
  </div>
</template>

<style scoped>
.search-input { min-width: 220px; }
.filter-bar {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.status-select {
  border-radius: 12px;
  border: 1px solid var(--md-sys-color-outline-variant);
  background: var(--md-sys-color-surface-container-low);
  color: var(--md-sys-color-on-surface);
  padding: 10px 12px;
  font: inherit;
}
.list-card { padding: 4px 0; }
.row-headline { display: flex; align-items: center; gap: 8px; }
.row-supporting { display: flex; flex-direction: column; gap: 4px; padding-top: 2px; }
.row-actions {
  margin-left: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
</style>
