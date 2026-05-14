<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { showConfirmDialog } from '@/shell/confirm'
import { extractMessage } from '@/shell/http'
import { deleteCard, listCards, upsertCard } from '../api'
import type { NFCCard } from '../types'

const items = ref<NFCCard[]>([])
const total = ref(0)
const q = ref('')
const loading = ref(false)
const editDialog = ref<HTMLDialogElement & { show: () => void; close: () => void } | null>(null)
const editing = reactive<{ hwid: string; userId: string; isRegistered: boolean; isNew: boolean }>({
  hwid: '',
  userId: '',
  isRegistered: false,
  isNew: true,
})
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await listCards({ q: q.value, limit: 200 })
    items.value = res.items ?? []
    total.value = res.total ?? 0
  } catch (err) {
    showFailToast(extractMessage(err, '加载失败'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.hwid = ''
  editing.userId = ''
  editing.isRegistered = false
  editing.isNew = true
  editDialog.value?.show()
}

function openEdit(c: NFCCard) {
  editing.hwid = c.hwid
  editing.userId = c.userId
  editing.isRegistered = c.isRegistered
  editing.isNew = false
  editDialog.value?.show()
}

async function onSave() {
  const hwid = editing.hwid.trim()
  if (!hwid) {
    showFailToast('请填写 hwid')
    return
  }
  saving.value = true
  try {
    await upsertCard({
      hwid,
      userId: editing.userId.trim(),
      isRegistered: editing.isRegistered,
    })
    showSuccessToast('已保存')
    editDialog.value?.close()
    await load()
  } catch (err) {
    showFailToast(extractMessage(err, '保存失败'))
  } finally {
    saving.value = false
  }
}

async function onDelete(c: NFCCard) {
  try {
    await showConfirmDialog({ title: '删除卡片', message: `确认删除 ${c.hwid}？` })
  } catch {
    return
  }
  try {
    await deleteCard(c.hwid)
    items.value = items.value.filter((x) => x.hwid !== c.hwid)
    showSuccessToast('已删除')
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
        <h1 class="m3-headline-medium text-on-surface">NFC 卡片</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">共 {{ total }} 条</p>
      </div>
      <md-outlined-text-field
        label="搜索 hwid / userId"
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
      <div class="m3-empty-icon"><md-icon>credit_card_off</md-icon></div>
      <div class="m3-title-medium text-on-surface">还没有 NFC 卡片</div>
      <div class="m3-body-medium text-on-surface-variant">点击右下角的按钮添加一张。</div>
    </div>

    <md-list v-else class="m3-card list-card">
      <template v-for="(c, i) in items" :key="c.hwid">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <md-icon slot="start" class="row-icon">credit_card</md-icon>
          <div slot="headline" class="m3-title-medium">{{ c.hwid }}</div>
          <div slot="supporting-text" class="m3-body-medium">
            {{ c.userId ? `userId: ${c.userId}` : '未绑定' }}
          </div>
          <div slot="end" class="row-actions">
            <md-assist-chip
              :label="c.isRegistered ? '已注册' : '未注册'"
              :class="c.isRegistered ? 'chip-tertiary' : 'chip-muted'"
            />
            <md-icon-button @click="openEdit(c)" aria-label="编辑">
              <md-icon>edit</md-icon>
            </md-icon-button>
            <md-icon-button @click="onDelete(c)" aria-label="删除">
              <md-icon>delete</md-icon>
            </md-icon-button>
          </div>
        </md-list-item>
      </template>
    </md-list>

    <md-fab class="m3-fab" variant="primary" aria-label="新建卡片" @click="openCreate">
      <md-icon slot="icon">add</md-icon>
    </md-fab>

    <md-dialog ref="editDialog">
      <div slot="headline">{{ editing.isNew ? '新建 NFC 卡片' : '编辑 NFC 卡片' }}</div>
      <form slot="content" id="card-form" method="dialog" class="dialog-form">
        <md-outlined-text-field
          label="hwid"
          :value="editing.hwid"
          :readonly="!editing.isNew"
          @input="(e: any) => (editing.hwid = e.target.value)"
        />
        <md-outlined-text-field
          label="userId（可选）"
          :value="editing.userId"
          @input="(e: any) => (editing.userId = e.target.value)"
        />
        <label class="switch-row">
          <span class="m3-body-medium text-on-surface">已注册</span>
          <md-switch
            :selected="editing.isRegistered"
            @change="(e: any) => (editing.isRegistered = e.target.selected)"
          />
        </label>
      </form>
      <div slot="actions">
        <md-text-button :disabled="saving" @click="editDialog?.close()">取消</md-text-button>
        <md-filled-button :disabled="saving" @click="onSave">
          {{ saving ? '保存中…' : '保存' }}
        </md-filled-button>
      </div>
    </md-dialog>
  </div>
</template>

<style scoped>
.search-input { min-width: 240px; }
.list-card { padding: 4px 0; }
.row-icon { color: var(--md-sys-color-primary); }
.row-actions { display: flex; align-items: center; gap: 6px; }
.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 8px;
}
.dialog-form md-outlined-text-field { width: 100%; }
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container-high);
}
</style>
