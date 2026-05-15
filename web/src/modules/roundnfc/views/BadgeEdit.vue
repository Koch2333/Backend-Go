<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { createBadge, getBadge, upsertBadge, uploadBadgeImage } from '../api'
import type { Badge } from '../types'

const route = useRoute()
const router = useRouter()
const idParam = route.params.id as string | undefined
const isNew = computed(() => !idParam || idParam === 'new')

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

async function onSubmit() {
  if (!form.id) {
    showFailToast('请填写 id')
    return
  }
  submitting.value = true
  try {
    if (isNew.value) {
      await createBadge({ ...form })
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
  if (isNew.value) {
    showFailToast('请先保存再上传图片')
    return
  }
  uploading.value = true
  try {
    const res = await uploadBadgeImage(form.id, file)
    form.imageUrl = res.key
    showSuccessToast('已上传')
  } catch (err) {
    showFailToast(extractMessage(err, '上传失败'))
  } finally {
    uploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

onMounted(load)
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
      <md-outlined-text-field
        label="样式 Key" placeholder="命中前端内置图，留空走 imageUrl"
        :value="form.styleKey"
        @input="(e: any) => (form.styleKey = e.target.value)"
      />
      <md-outlined-text-field
        label="图片 Key/URL" placeholder="后端对象 key 或绝对 URL"
        :value="form.imageUrl"
        @input="(e: any) => (form.imageUrl = e.target.value)"
      />
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

      <div v-if="!isNew" class="upload-row">
        <input ref="fileInput" type="file" accept="image/*" hidden @change="onPickFile" />
        <md-outlined-button type="button" :disabled="uploading" @click="fileInput?.click()">
          <md-icon slot="icon">upload</md-icon>
          {{ uploading ? '上传中…' : '上传徽章主图' }}
        </md-outlined-button>
        <span class="m3-body-small text-on-surface-variant">
          jpeg/png/webp/gif，最大 8MB
        </span>
      </div>

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
.upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-top: 4px;
}
</style>
