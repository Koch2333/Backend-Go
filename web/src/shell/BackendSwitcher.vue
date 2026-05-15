<script setup lang="ts">
import { ref, computed } from 'vue'
import { useApiBaseRef, setApiBase, getRuntimeApiBase } from './backend'
import { showSuccessToast, showFailToast } from './toast'

const props = defineProps<{ moduleName: string }>()

const current = useApiBaseRef(props.moduleName)
const editing = ref(false)
const draft = ref('')
const runtimeDefault = getRuntimeApiBase()

const effective = computed(() => current.value || runtimeDefault)
const display = computed(() => effective.value || '（同源）')
const sourceHint = computed(() => {
  if (current.value) return '自定义'
  if (runtimeDefault) return '默认'
  return '同源'
})

function startEdit() {
  draft.value = current.value
  editing.value = true
}

function cancel() {
  editing.value = false
  draft.value = ''
}

function save() {
  try {
    setApiBase(props.moduleName, draft.value)
    editing.value = false
    if (draft.value.trim()) {
      showSuccessToast('已切换后端')
    } else {
      showSuccessToast(runtimeDefault ? '已恢复默认后端' : '已切回同源')
    }
  } catch (e) {
    showFailToast((e as Error).message || '保存失败')
  }
}
</script>

<template>
  <div class="backend-switcher">
    <div v-if="!editing" class="row">
      <span class="m3-body-small text-on-surface-variant label">连接到：</span>
      <span class="m3-body-small value" :title="display">{{ display }}</span>
      <span class="m3-label-small chip" :data-source="sourceHint">{{ sourceHint }}</span>
      <md-icon-button aria-label="编辑后端地址" @click="startEdit">
        <md-icon>edit</md-icon>
      </md-icon-button>
    </div>
    <div v-else class="edit">
      <md-outlined-text-field
        label="后端 API 地址"
        :placeholder="runtimeDefault || 'https://api.example.com'"
        :value="draft"
        @input="(e: any) => (draft = e.target.value)"
        @keydown.enter.prevent="save"
      />
      <div class="edit-actions">
        <md-text-button @click="cancel">取消</md-text-button>
        <md-filled-button @click="save">保存</md-filled-button>
      </div>
      <p class="m3-body-small text-on-surface-variant hint">
        {{ runtimeDefault ? `留空将使用默认后端 ${runtimeDefault}` : '留空表示使用与本页同源的后端。' }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.backend-switcher {
  width: 100%;
  padding: 8px 12px;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
}
.row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.label { flex: 0 0 auto; }
.chip {
  flex: 0 0 auto;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--md-sys-color-secondary-container);
  color: var(--md-sys-color-on-secondary-container);
}
.chip[data-source="自定义"] {
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
}
.value {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--md-sys-color-on-surface);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.edit { display: flex; flex-direction: column; gap: 8px; padding: 4px 0; }
.edit md-outlined-text-field { width: 100%; }
.edit-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.hint { margin: 0; }
</style>
