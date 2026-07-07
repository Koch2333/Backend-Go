<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { createBadge, getBadge, listBadgeStyles, upsertBadge, uploadBadgeImage, type BadgeStyleTemplate } from '../api'
import type { Badge } from '../types'

const route = useRoute()
const router = useRouter()
const idParam = route.params.id as string | undefined
const isNew = computed(() => !idParam || idParam === 'new')

// 这枚徽章是否已经在后端存在。新建页一旦保存（含上传图片时的自动保存）就置 true，
// 之后的保存走更新而不是再次创建。
const persisted = ref(!isNew.value)

const form = reactive<Partial<Badge> & { id: string }>({
  id: '',
  title: '',
  series: '',
  type: '',
  styleKey: '',
  imageUrl: '',
  description: '',
  serialNo: '',
  releasedAt: '',
})

const loading = ref(false)
const submitting = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const styleOptions = ref<BadgeStyleTemplate[]>([])

async function loadStyles() {
  try {
    const r = await listBadgeStyles()
    styleOptions.value = r.items ?? []
  } catch (err) {
    showFailToast(extractMessage(err, '加载样式失败'))
  }
}

async function load() {
  if (isNew.value) return
  loading.value = true
  try {
    const b = await getBadge(idParam!)
    Object.assign(form, b)
  } catch (err) {
    showFailToast(extractMessage(err, '加载失败'))
  } finally {
    loading.value = false
  }
}

// 确保徽章已在后端存在；新建态会先校验必填项并自动创建一次。
async function ensurePersisted(): Promise<boolean> {
  if (persisted.value) return true
  if (!form.id) {
    showFailToast('请先填写 ID')
    return false
  }
  if (!form.title) {
    showFailToast('请先填写标题')
    return false
  }
  await createBadge({ ...form })
  persisted.value = true
  return true
}

async function onSubmit() {
  if (!form.id) {
    showFailToast('请填写 id')
    return
  }
  submitting.value = true
  try {
    if (!persisted.value) {
      await createBadge({ ...form })
      persisted.value = true
    } else {
      await upsertBadge({ ...form })
    }
    showSuccessToast('已保存')
    router.replace('/m/roundnfc/badges')
  } catch (err) {
    showFailToast(extractMessage(err, '保存失败'))
  } finally {
    submitting.value = false
  }
}

async function onPickFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploading.value = true
  try {
    // 新建时先自动保存一次，省得用户被「请先保存再上传」打断。
    if (!(await ensurePersisted())) return
    const res = await uploadBadgeImage(form.id, file)
    form.imageUrl = res.key
    showSuccessToast('已上传，已自动填入图片地址')
  } catch (err) {
    showFailToast(extractMessage(err, '上传失败'))
  } finally {
    uploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

onMounted(async () => {
  await Promise.all([loadStyles(), load()])
})

function fmtDate(s?: string) {
  if (!s) return ''
  return new Date(s).toLocaleString('zh-CN')
}
</script>

<template>
  <div class="space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">
          {{ isNew ? '新建徽章' : `编辑 ${idParam}` }}
        </h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">
          填写徽章信息后保存，再回到列表。
        </p>
      </div>
    </header>

    <div v-if="loading" class="m3-loading">
      <md-circular-progress indeterminate aria-label="加载中" />
      <span class="m3-body-medium">加载中…</span>
    </div>

    <form v-else class="m3-card edit-card" @submit.prevent="onSubmit">
      <md-outlined-text-field
        label="ID" placeholder="例如 DEMO001"
        :value="form.id" :readonly="!isNew"
        @input="(e: any) => (form.id = e.target.value)" required
      />
      <md-outlined-text-field
        label="标题" placeholder="徽章标题"
        :value="form.title"
        @input="(e: any) => (form.title = e.target.value)" required
      />
      <md-outlined-text-field
        label="系列" placeholder="所属作品"
        :value="form.series"
        @input="(e: any) => (form.series = e.target.value)"
      />
      <md-outlined-text-field
        label="类型" placeholder="亚克力 / 金属…"
        :value="form.type"
        @input="(e: any) => (form.type = e.target.value)"
      />
      <md-outlined-select
        label="内置样式" :value="form.styleKey ?? ''"
        @change="(e: any) => (form.styleKey = e.target.value)"
      >
        <md-select-option value="">
          <div slot="headline">无（使用上传的图片）</div>
        </md-select-option>
        <md-select-option v-for="opt in styleOptions" :key="opt.key" :value="opt.key">
          <div slot="headline">{{ opt.label }}</div>
        </md-select-option>
      </md-outlined-select>
      <md-outlined-text-field
        label="编号" placeholder="12 / 50"
        :value="form.serialNo"
        @input="(e: any) => (form.serialNo = e.target.value)"
      />
      <md-outlined-text-field
        label="发放" placeholder="2026 上海 BW"
        :value="form.releasedAt"
        @input="(e: any) => (form.releasedAt = e.target.value)"
      />
      <md-outlined-text-field
        label="描述" type="textarea" rows="3"
        :value="form.description"
        @input="(e: any) => (form.description = e.target.value)"
      />

      <div class="image-section">
        <div class="m3-title-small text-on-surface">徽章主图</div>
        <p class="m3-body-small text-on-surface-variant">
          选了上面的「内置样式」可以不传图；要用自定义图就在这里上传，上传后会自动填好下面的地址。
        </p>
        <div class="upload-row">
          <input ref="fileInput" type="file" accept="image/*" hidden @change="onPickFile" />
          <md-outlined-button type="button" :disabled="uploading" @click="fileInput?.click()">
            <md-icon slot="icon">upload</md-icon>
            {{ uploading ? '上传中…' : (form.imageUrl ? '更换主图' : '上传主图') }}
          </md-outlined-button>
          <span class="m3-body-small text-on-surface-variant">
            jpeg/png/webp/gif，最大 8MB
          </span>
        </div>
        <md-outlined-text-field
          label="图片 Key / URL（高级）" placeholder="上传后自动填写，也可手填后端对象 key 或绝对 URL"
          :value="form.imageUrl"
          @input="(e: any) => (form.imageUrl = e.target.value)"
        />
      </div>

      <section v-if="form.coserBinding" class="binding-section">
        <div class="m3-title-small text-on-surface">CN 绑定</div>
        <div class="binding-grid m3-body-medium">
          <span class="text-on-surface-variant">CN</span>
          <span class="text-on-surface">{{ form.coserBinding.cn }}</span>
          <span class="text-on-surface-variant">图片 Key</span>
          <code class="binding-code">{{ form.coserBinding.photoObjectKey }}</code>
          <span v-if="form.coserBinding.deviceId" class="text-on-surface-variant">设备</span>
          <span v-if="form.coserBinding.deviceId" class="text-on-surface">{{ form.coserBinding.deviceId }}</span>
          <span v-if="form.coserBinding.tagUid" class="text-on-surface-variant">Tag UID</span>
          <span v-if="form.coserBinding.tagUid" class="text-on-surface">{{ form.coserBinding.tagUid }}</span>
          <span v-if="form.coserBinding.writtenAt" class="text-on-surface-variant">写入时间</span>
          <span v-if="form.coserBinding.writtenAt" class="text-on-surface">
            {{ fmtDate(form.coserBinding.writtenAt) }}
          </span>
        </div>
      </section>

      <div class="pt-1">
        <md-filled-button type="submit" :disabled="submitting" class="w-full">
          {{ submitting ? '保存中…' : '保存' }}
        </md-filled-button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.edit-card {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
md-outlined-text-field { width: 100%; }
md-outlined-select { width: 100%; }
.image-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: 12px;
}
.upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-top: 4px;
}
.binding-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: 12px;
}
.binding-grid {
  display: grid;
  grid-template-columns: minmax(72px, max-content) minmax(0, 1fr);
  gap: 8px 12px;
}
.binding-code {
  min-width: 0;
  word-break: break-all;
  color: var(--md-sys-color-on-surface);
}
</style>
