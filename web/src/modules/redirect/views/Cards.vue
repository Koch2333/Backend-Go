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
const showEdit = ref(false)
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
  showEdit.value = true
}

function openEdit(c: NFCCard) {
  editing.hwid = c.hwid
  editing.userId = c.userId
  editing.isRegistered = c.isRegistered
  editing.isNew = false
  showEdit.value = true
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
    showEdit.value = false
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
  <div>
    <div class="flex items-center gap-2">
      <van-search
        v-model="q"
        class="!flex-1 !p-0"
        placeholder="按 hwid / userId 搜索"
        shape="round"
        @search="load"
      />
      <van-button type="primary" size="small" @click="openCreate">新建</van-button>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-400">
      还没有 NFC 卡片
    </div>

    <van-cell-group v-else inset class="mt-3">
      <van-cell
        v-for="c in items"
        :key="c.hwid"
        :title="c.hwid"
        :label="c.userId ? `userId: ${c.userId}` : '未绑定'"
      >
        <template #value>
          <div class="flex items-center justify-end gap-2">
            <van-tag :type="c.isRegistered ? 'success' : 'warning'" size="medium">
              {{ c.isRegistered ? '已注册' : '未注册' }}
            </van-tag>
            <van-button size="mini" plain @click="openEdit(c)">编辑</van-button>
            <van-button size="mini" type="danger" plain @click="onDelete(c)">删除</van-button>
          </div>
        </template>
      </van-cell>
    </van-cell-group>

    <p class="pt-3 text-center text-xs text-gray-400">共 {{ total }} 条</p>

    <van-dialog
      v-model:show="showEdit"
      :title="editing.isNew ? '新建 NFC 卡片' : '编辑 NFC 卡片'"
      :show-confirm-button="false"
      :close-on-click-overlay="!saving"
    >
      <div class="p-4 space-y-3">
        <van-field
          v-model="editing.hwid"
          label="hwid"
          placeholder="NFC 硬件 ID"
          :readonly="!editing.isNew"
        />
        <van-field v-model="editing.userId" label="userId" placeholder="可选，绑定的用户 ID" />
        <van-cell title="已注册">
          <template #right-icon>
            <van-switch v-model="editing.isRegistered" size="20" />
          </template>
        </van-cell>
        <div class="flex gap-2">
          <van-button class="flex-1" round :disabled="saving" @click="showEdit = false">
            取消
          </van-button>
          <van-button class="flex-1" round type="primary" :loading="saving" @click="onSave">
            保存
          </van-button>
        </div>
      </div>
    </van-dialog>
  </div>
</template>
