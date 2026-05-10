<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from 'vant'
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
  <div>
    <h2 class="px-1 pb-3 text-base font-semibold text-gray-800">
      {{ isNew ? '新建徽章' : `编辑 ${idParam}` }}
    </h2>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>

    <van-form v-else @submit="onSubmit">
      <van-cell-group inset>
        <van-field v-model="form.id" label="ID" placeholder="例如 DEMO001" :readonly="!isNew" required />
        <van-field v-model="form.title" label="标题" placeholder="徽章标题" required />
        <van-field v-model="form.series" label="系列" placeholder="所属作品" />
        <van-field v-model="form.type" label="类型" placeholder="亚克力 / 金属…" />
        <van-field
          v-model="form.styleKey"
          label="样式 Key"
          placeholder="命中前端内置图，留空走 imageUrl"
        />
        <van-field
          v-model="form.imageUrl"
          label="图片 Key/URL"
          placeholder="后端对象 key 或绝对 URL"
        />
        <van-field v-model="form.serialNo" label="编号" placeholder="12 / 50" />
        <van-field v-model="form.releasedAt" label="发放" placeholder="2026 上海 BW" />
        <van-field v-model="form.description" label="描述" type="textarea" rows="3" autosize />
      </van-cell-group>

      <div v-if="!isNew" class="mt-3 flex items-center gap-2 px-4">
        <input ref="fileInput" type="file" accept="image/*" hidden @change="onPickFile" />
        <van-button size="small" :loading="uploading" @click="fileInput?.click()">
          上传徽章主图
        </van-button>
        <span class="text-xs text-gray-400">jpeg/png/webp/gif，最大 8MB</span>
      </div>

      <div class="px-4 pb-4 pt-4">
        <van-button block round type="primary" native-type="submit" :loading="submitting">
          保存
        </van-button>
      </div>
    </van-form>
  </div>
</template>
