<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { showConfirmDialog } from '@/shell/confirm'
import { extractMessage } from '@/shell/http'
import { deleteRule, listRules, upsertRule } from '../api'
import type { RedirectRule } from '../types'

const items = ref<RedirectRule[]>([])
const total = ref(0)
const q = ref('')
const loading = ref(false)
const editDialog = ref<HTMLDialogElement & { show: () => void; close: () => void } | null>(null)
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
  editDialog.value?.show()
}

function openEdit(r: RedirectRule) {
  editing.name = r.name
  editing.targetUrl = r.targetUrl
  editing.enabled = r.enabled
  editing.isNew = false
  editDialog.value?.show()
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
    editDialog.value?.close()
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

async function onToggle(r: RedirectRule, on: boolean) {
  try {
    await upsertRule({ name: r.name, targetUrl: r.targetUrl, enabled: on })
    r.enabled = on
  } catch (err) {
    showFailToast(extractMessage(err, '切换失败'))
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-3 p-2">
    <!-- search + new -->
    <div class="flex items-center gap-2">
      <md-outlined-text-field
        label="搜索 name / URL"
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
      还没有短链规则
    </div>

    <md-list v-else class="rounded-2xl bg-white">
      <template v-for="(r, i) in items" :key="r.name">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <div slot="headline">{{ r.name }}</div>
          <div slot="supporting-text" class="truncate">{{ r.targetUrl }}</div>
          <div slot="end" class="flex items-center gap-1">
            <md-switch
              :selected="r.enabled"
              @change="(e: any) => onToggle(r, e.target.selected)"
            />
            <md-icon-button @click="openEdit(r)" aria-label="编辑">
              <md-icon>edit</md-icon>
            </md-icon-button>
            <md-icon-button @click="onDelete(r)" aria-label="删除">
              <md-icon>delete</md-icon>
            </md-icon-button>
          </div>
        </md-list-item>
      </template>
    </md-list>

    <p class="pt-1 text-center text-xs text-gray-400">共 {{ total }} 条</p>

    <md-dialog ref="editDialog">
      <div slot="headline">{{ editing.isNew ? '新建规则' : '编辑规则' }}</div>
      <form slot="content" id="rule-form" method="dialog" class="space-y-3 pt-2">
        <md-outlined-text-field
          label="name"
          :value="editing.name"
          :readonly="!editing.isNew"
          @input="(e: any) => (editing.name = e.target.value)"
          class="w-full"
        />
        <md-outlined-text-field
          label="跳转 URL"
          :value="editing.targetUrl"
          @input="(e: any) => (editing.targetUrl = e.target.value)"
          class="w-full"
        />
        <label class="flex items-center justify-between rounded-xl bg-gray-50 px-3 py-2">
          <span class="text-sm">启用</span>
          <md-switch
            :selected="editing.enabled"
            @change="(e: any) => (editing.enabled = e.target.selected)"
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
</style>
