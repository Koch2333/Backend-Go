<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { showConfirmDialog } from '@/shell/confirm'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { acquireBlobImage, blobImageCacheKey, releaseBlobImage } from '@/shell/blobImage'
import {
  createBadgeStyleTemplate,
  deleteBadgeStyleTemplate,
  listBadgeStyles,
  saveBadgeStyleTemplate,
  uploadBadgeStyleTemplateImage,
  type BadgeStyleTemplate,
} from '../api'

const items = ref<BadgeStyleTemplate[]>([])
const loading = ref(false)
const working = ref(false)
const dialog = ref<HTMLDialogElement & { show: () => void; close: () => void } | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const editingKey = ref('')
const payloadText = ref('{}')
const uploadKey = ref('')
const previewSrc = reactive<Record<string, string>>({})
const previewKeys = new Map<string, string>()
let previewRun = 0

const form = reactive<BadgeStyleTemplate>({
  key: '',
  label: '',
  description: '',
  imageUrl: '',
  payload: {},
  enabled: true,
})

async function load() {
  loading.value = true
  try {
    const r = await listBadgeStyles()
    items.value = r.items ?? []
    void hydratePreviews(items.value)
  } catch (e) {
    showFailToast(extractMessage(e, '加载失败'))
  } finally {
    loading.value = false
  }
}

async function hydratePreviews(nextItems: BadgeStyleTemplate[]) {
  const run = ++previewRun
  const nextItemKeys = new Set(nextItems.map((t) => t.key))
  for (const [itemKey, cacheKey] of previewKeys) {
    if (nextItemKeys.has(itemKey)) continue
    releaseBlobImage(cacheKey)
    previewKeys.delete(itemKey)
    delete previewSrc[itemKey]
  }

  for (const t of nextItems) {
    const source = t.imageOriginalUrl || t.imagePreviewUrl
    if (!source) {
      releasePreview(t.key)
      continue
    }
    const cacheKey = blobImageCacheKey('roundnfc', 'style-template', t.key, t.imageUrl || source)
    if (previewKeys.get(t.key) === cacheKey && previewSrc[t.key]) continue
    releasePreview(t.key)
    try {
      const out = await acquireBlobImage(source, { cacheKey })
      if (run !== previewRun) {
        releaseBlobImage(out.key)
        continue
      }
      previewKeys.set(t.key, out.key)
      previewSrc[t.key] = out.src
    } catch {
      if (run === previewRun) delete previewSrc[t.key]
    }
  }
}

function releasePreview(itemKey: string) {
  releaseBlobImage(previewKeys.get(itemKey))
  previewKeys.delete(itemKey)
  delete previewSrc[itemKey]
}

function releaseAllPreviews() {
  previewRun += 1
  for (const cacheKey of previewKeys.values()) releaseBlobImage(cacheKey)
  previewKeys.clear()
  for (const key of Object.keys(previewSrc)) delete previewSrc[key]
}

function openCreate() {
  editingKey.value = ''
  form.key = ''
  form.label = ''
  form.description = ''
  form.imageUrl = ''
  form.payload = {}
  form.enabled = true
  payloadText.value = '{\n  "theme": ""\n}'
  dialog.value?.show()
}

function openEdit(t: BadgeStyleTemplate) {
  editingKey.value = t.key
  form.key = t.key
  form.label = t.label
  form.description = t.description ?? ''
  form.imageUrl = t.imageUrl ?? ''
  form.payload = t.payload ?? {}
  form.enabled = t.enabled ?? true
  payloadText.value = JSON.stringify(form.payload, null, 2)
  dialog.value?.show()
}

function parsePayload() {
  try {
    return JSON.parse(payloadText.value || '{}')
  } catch {
    throw new Error('payload 不是合法 JSON')
  }
}

async function submit() {
  const key = form.key.trim()
  const label = form.label.trim()
  if (!key) {
    showFailToast('请填写 key')
    return
  }
  if (!label) {
    showFailToast('请填写名称')
    return
  }
  let payload: unknown
  try {
    payload = parsePayload()
  } catch (e) {
    showFailToast(extractMessage(e, 'payload 不是合法 JSON'))
    return
  }
  working.value = true
  try {
    const body: BadgeStyleTemplate = {
      key,
      label,
      description: form.description?.trim() ?? '',
      imageUrl: form.imageUrl?.trim() ?? '',
      payload,
      enabled: form.enabled ?? true,
    }
    if (editingKey.value) await saveBadgeStyleTemplate(body)
    else await createBadgeStyleTemplate(body)
    showSuccessToast('已保存')
    dialog.value?.close()
    await load()
  } catch (e) {
    showFailToast(extractMessage(e, '保存失败'))
  } finally {
    working.value = false
  }
}

function pickImage(key: string) {
  uploadKey.value = key
  fileInput.value?.click()
}

async function onPickFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  const key = uploadKey.value
  if (!file || !key) return
  working.value = true
  try {
    const r = await uploadBadgeStyleTemplateImage(key, file)
    if (form.key === key) form.imageUrl = r.key
    showSuccessToast('模板图已上传')
    await load()
  } catch (err) {
    showFailToast(extractMessage(err, '上传失败'))
  } finally {
    working.value = false
    uploadKey.value = ''
    if (fileInput.value) fileInput.value.value = ''
  }
}

async function toggleEnabled(t: BadgeStyleTemplate) {
  working.value = true
  try {
    await saveBadgeStyleTemplate({ ...t, enabled: !(t.enabled ?? true) })
    showSuccessToast((t.enabled ?? true) ? '已停用' : '已启用')
    await load()
  } catch (e) {
    showFailToast(extractMessage(e, '操作失败'))
  } finally {
    working.value = false
  }
}

async function remove(t: BadgeStyleTemplate) {
  try {
    await showConfirmDialog({ title: '删除样式模板', message: `确认删除 ${t.label} (${t.key})？` })
  } catch {
    return
  }
  working.value = true
  try {
    await deleteBadgeStyleTemplate(t.key)
    showSuccessToast('已删除')
    await load()
  } catch (e) {
    showFailToast(extractMessage(e, '删除失败'))
  } finally {
    working.value = false
  }
}

onMounted(load)
onBeforeUnmount(releaseAllPreviews)
</script>

<template>
  <div class="space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">样式模板</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">
          Android App 会从这里读取可选模板。
        </p>
      </div>
      <md-filled-button @click="openCreate">
        <md-icon slot="icon">add</md-icon>
        新建
      </md-filled-button>
    </header>

    <div v-if="loading" class="m3-loading">
      <md-circular-progress indeterminate aria-label="加载中" />
      <span class="m3-body-medium">加载中…</span>
    </div>

    <div v-else-if="items.length === 0" class="m3-card m3-empty">
      <div class="m3-empty-icon"><md-icon>format_paint</md-icon></div>
      <div class="m3-title-medium text-on-surface">还没有样式模板</div>
      <div class="m3-body-medium text-on-surface-variant">点击右上角创建一个模板。</div>
    </div>

    <md-list v-else class="m3-card list-card">
      <input ref="fileInput" type="file" accept="image/*" hidden @change="onPickFile" />
      <template v-for="(t, i) in items" :key="t.key">
        <md-divider v-if="i > 0" />
        <md-list-item>
          <div slot="start" class="template-thumb">
            <img v-if="previewSrc[t.key]" :src="previewSrc[t.key]" alt="" />
            <md-icon v-else>format_paint</md-icon>
          </div>
          <div slot="headline" class="m3-title-medium">{{ t.label }}</div>
          <div slot="supporting-text" class="m3-body-medium">
            {{ t.key }}
            <span v-if="t.description"> · {{ t.description }}</span>
            <span v-if="t.imageUrl"> · {{ t.imageUrl }}</span>
          </div>
          <md-assist-chip
            slot="end"
            :label="(t.enabled ?? true) ? '启用中' : '已停用'"
            :class="(t.enabled ?? true) ? 'chip-tertiary' : 'chip-muted'"
          />
          <md-icon-button
            slot="end"
            :disabled="working"
            :aria-label="(t.enabled ?? true) ? '停用' : '启用'"
            @click="toggleEnabled(t)"
          >
            <md-icon>{{ (t.enabled ?? true) ? 'pause_circle' : 'play_circle' }}</md-icon>
          </md-icon-button>
          <md-icon-button slot="end" :disabled="working" aria-label="编辑" @click="openEdit(t)">
            <md-icon>edit</md-icon>
          </md-icon-button>
          <md-icon-button slot="end" :disabled="working" aria-label="上传图片" @click="pickImage(t.key)">
            <md-icon>image</md-icon>
          </md-icon-button>
          <md-icon-button slot="end" :disabled="working" aria-label="删除" @click="remove(t)">
            <md-icon>delete</md-icon>
          </md-icon-button>
        </md-list-item>
      </template>
    </md-list>

    <md-dialog ref="dialog">
      <div slot="headline">{{ editingKey ? '编辑样式模板' : '新建样式模板' }}</div>
      <form slot="content" id="style-template-form" method="dialog" class="dialog-form">
        <md-outlined-text-field
          label="Key"
          placeholder="例如 sakura"
          :readonly="!!editingKey"
          :value="form.key"
          @input="(e: any) => (form.key = e.target.value)"
        />
        <md-outlined-text-field
          label="名称"
          placeholder="例如 樱花粉"
          :value="form.label"
          @input="(e: any) => (form.label = e.target.value)"
        />
        <md-outlined-text-field
          label="描述"
          :value="form.description"
          @input="(e: any) => (form.description = e.target.value)"
        />
        <md-outlined-text-field
          label="图片 Key / URL"
          placeholder="上传后自动填写，也可手填对象 key 或 URL"
          :value="form.imageUrl"
          @input="(e: any) => (form.imageUrl = e.target.value)"
        />
        <label class="switch-row">
          <span class="m3-body-medium text-on-surface">启用</span>
          <md-switch
            :selected="form.enabled"
            @change="(e: any) => (form.enabled = e.target.selected)"
          />
        </label>
        <md-outlined-text-field
          label="Payload JSON"
          type="textarea"
          rows="8"
          :value="payloadText"
          @input="(e: any) => (payloadText = e.target.value)"
        />
      </form>
      <div slot="actions">
        <md-text-button :disabled="working" @click="dialog?.close()">取消</md-text-button>
        <md-filled-button :disabled="working" @click="submit">
          {{ working ? '保存中…' : '保存' }}
        </md-filled-button>
      </div>
    </md-dialog>
  </div>
</template>

<style scoped>
.list-card { padding: 4px 0; }
.template-thumb {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: var(--md-sys-color-surface-container-high);
  color: var(--md-sys-color-primary);
}
.template-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.dialog-form {
  display: grid;
  gap: 14px;
  padding-top: 8px;
  min-width: min(560px, calc(100vw - 48px));
}
.dialog-form md-outlined-text-field { width: 100%; }
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 48px;
}
</style>
