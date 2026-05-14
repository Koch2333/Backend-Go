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
  <div class="space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">短链规则</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">共 {{ total }} 条</p>
      </div>
      <md-outlined-text-field
        label="搜索 name / URL"
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
      <div class="m3-empty-icon"><md-icon>link_off</md-icon></div>
      <div class="m3-title-medium text-on-surface">还没有短链规则</div>
      <div class="m3-body-medium text-on-surface-variant">点击右下角的按钮创建第一条。</div>
    </div>

    <md-list v-else class="m3-card list-card">
      <template v-for="(r, i) in items" :key="r.name">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <md-icon slot="start" class="row-icon">link</md-icon>
          <div slot="headline" class="m3-title-medium">{{ r.name }}</div>
          <div slot="supporting-text" class="m3-body-medium truncate">{{ r.targetUrl }}</div>
          <div slot="end" class="row-actions">
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

    <md-fab class="m3-fab" variant="primary" aria-label="新建规则" @click="openCreate">
      <md-icon slot="icon">add</md-icon>
    </md-fab>

    <md-dialog ref="editDialog">
      <div slot="headline">{{ editing.isNew ? '新建规则' : '编辑规则' }}</div>
      <form slot="content" id="rule-form" method="dialog" class="dialog-form">
        <md-outlined-text-field
          label="name"
          :value="editing.name"
          :readonly="!editing.isNew"
          @input="(e: any) => (editing.name = e.target.value)"
        />
        <md-outlined-text-field
          label="跳转 URL"
          :value="editing.targetUrl"
          @input="(e: any) => (editing.targetUrl = e.target.value)"
        />
        <label class="switch-row">
          <span class="m3-body-medium text-on-surface">启用</span>
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
.search-input { min-width: 240px; }
.list-card { padding: 4px 0; }
.row-icon { color: var(--md-sys-color-primary); }
.row-actions { display: flex; align-items: center; gap: 4px; }
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
