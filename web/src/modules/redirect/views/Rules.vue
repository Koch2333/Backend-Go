<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showConfirmDialog, showFailToast, showSuccessToast } from 'vant'
import { extractMessage } from '@/shell/http'
import { deleteRule, listRules, upsertRule } from '../api'
import type { RedirectRule } from '../types'

const items = ref<RedirectRule[]>([])
const total = ref(0)
const q = ref('')
const loading = ref(false)
const showEdit = ref(false)
const editing = reactive<{ name: string; targetUrl: string; enabled: boolean; isNew: boolean }>({
  name: '',
  targetUrl: '',
  enabled: true,
  isNew: true,
})
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await listRules({ q: q.value, limit: 200 })
    items.value = res.items ?? []
    total.value = res.total ?? 0
  } catch (err) {
    showFailToast(extractMessage(err, '加载失败'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.name = ''
  editing.targetUrl = ''
  editing.enabled = true
  editing.isNew = true
  showEdit.value = true
}

function openEdit(r: RedirectRule) {
  editing.name = r.name
  editing.targetUrl = r.targetUrl
  editing.enabled = r.enabled
  editing.isNew = false
  showEdit.value = true
}

async function onSave() {
  const name = editing.name.trim()
  const targetUrl = editing.targetUrl.trim()
  if (!name) {
    showFailToast('请填写 name')
    return
  }
  if (!targetUrl) {
    showFailToast('请填写 target URL')
    return
  }
  saving.value = true
  try {
    await upsertRule({ name, targetUrl, enabled: editing.enabled })
    showSuccessToast('已保存')
    showEdit.value = false
    await load()
  } catch (err) {
    showFailToast(extractMessage(err, '保存失败'))
  } finally {
    saving.value = false
  }
}

async function onDelete(r: RedirectRule) {
  try {
    await showConfirmDialog({ title: '删除规则', message: `确认删除 ${r.name}？` })
  } catch {
    return
  }
  try {
    await deleteRule(r.name)
    items.value = items.value.filter((x) => x.name !== r.name)
    showSuccessToast('已删除')
  } catch (err) {
    showFailToast(extractMessage(err, '删除失败'))
  }
}

async function onToggle(r: RedirectRule) {
  try {
    await upsertRule({ name: r.name, targetUrl: r.targetUrl, enabled: !r.enabled })
    r.enabled = !r.enabled
  } catch (err) {
    showFailToast(extractMessage(err, '切换失败'))
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
        placeholder="按 name / URL 搜索"
        shape="round"
        @search="load"
      />
      <van-button type="primary" size="small" @click="openCreate">新建</van-button>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-400">
      还没有短链规则
    </div>

    <van-cell-group v-else inset class="mt-3">
      <van-cell v-for="r in items" :key="r.name" :title="r.name" :label="r.targetUrl">
        <template #value>
          <div class="flex items-center justify-end gap-2">
            <van-switch :model-value="r.enabled" size="18" @update:model-value="onToggle(r)" />
            <van-button size="mini" plain @click="openEdit(r)">编辑</van-button>
            <van-button size="mini" type="danger" plain @click="onDelete(r)">删除</van-button>
          </div>
        </template>
      </van-cell>
    </van-cell-group>

    <p class="pt-3 text-center text-xs text-gray-400">共 {{ total }} 条</p>

    <van-dialog
      v-model:show="showEdit"
      :title="editing.isNew ? '新建规则' : '编辑规则'"
      :show-confirm-button="false"
      :close-on-click-overlay="!saving"
    >
      <div class="p-4 space-y-3">
        <van-field
          v-model="editing.name"
          label="name"
          placeholder="短链 name，例如 home"
          :readonly="!editing.isNew"
        />
        <van-field
          v-model="editing.targetUrl"
          label="跳转 URL"
          placeholder="https://example.com/{name}"
        />
        <van-cell title="启用">
          <template #right-icon>
            <van-switch v-model="editing.enabled" size="20" />
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
