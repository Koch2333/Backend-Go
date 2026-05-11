<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showConfirmDialog, showFailToast, showSuccessToast } from 'vant'
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
  <div class="space-y-3 p-2">
    <div class="flex items-center gap-2">
      <md-outlined-text-field
        label="搜索 hwid / userId"
        :value="q"
        @input="(e: any) => (q = e.target.value)"
        @keyup.enter="load"
        class="flex-1"
      >
        <md-icon slot="leading-icon">search</md-icon>
      </md-outlined-text-field>
      <md-filled-button @click="openCreate">
        <md-icon slot="icon">add</md-icon>
        新建
      </md-filled-button>
    </div>

    <div
      v-if="loading"
      class="py-8 text-center text-sm text-gray-400"
    >
      <md-circular-progress indeterminate aria-label="加载中" />
    </div>
    <div
      v-else-if="items.length === 0"
      class="py-8 text-center text-sm text-gray-400"
    >
      还没有 NFC 卡片
    </div>

    <md-list v-else class="rounded-2xl bg-white">
      <template v-for="(c, i) in items" :key="c.hwid">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <div slot="headline">{{ c.hwid }}</div>
          <div slot="supporting-text">
            {{ c.userId ? `userId: ${c.userId}` : '未绑定' }}
          </div>
          <div slot="end" class="flex items-center gap-1">
            <md-assist-chip
              :label="c.isRegistered ? '已注册' : '未注册'"
              :class="c.isRegistered ? 'chip-success' : 'chip-warning'"
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

    <p class="pt-1 text-center text-xs text-gray-400">共 {{ total }} 条</p>

    <md-dialog ref="editDialog">
      <div slot="headline">{{ editing.isNew ? '新建 NFC 卡片' : '编辑 NFC 卡片' }}</div>
      <form slot="content" id="card-form" method="dialog" class="space-y-3 pt-2">
        <md-outlined-text-field
          label="hwid"
          :value="editing.hwid"
          :readonly="!editing.isNew"
          @input="(e: any) => (editing.hwid = e.target.value)"
          class="w-full"
        />
        <md-outlined-text-field
          label="userId（可选）"
          :value="editing.userId"
          @input="(e: any) => (editing.userId = e.target.value)"
          class="w-full"
        />
        <label class="flex items-center justify-between rounded-xl bg-gray-50 px-3 py-2">
          <span class="text-sm">已注册</span>
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
</style>
