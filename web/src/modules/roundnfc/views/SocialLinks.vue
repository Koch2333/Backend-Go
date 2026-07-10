<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { listSocialLinks, saveSocialLinks, type SocialLink } from '../api'

const items = ref<SocialLink[]>([])
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const result = await listSocialLinks()
    items.value = result.items ?? []
  } catch (error) {
    showFailToast(extractMessage(error, '加载扩列方式失败'))
  } finally {
    loading.value = false
  }
}

async function save() {
  const invalid = items.value.find((item) => !item.key.trim() || !item.label.trim())
  if (invalid) {
    showFailToast('平台标识和显示名称不能为空')
    return
  }
  saving.value = true
  try {
    const result = await saveSocialLinks(items.value)
    items.value = result.items ?? items.value
    showSuccessToast('扩列方式已保存')
  } catch (error) {
    showFailToast(extractMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">扩列方式</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">
          配置公开页展示的社交账号；填写链接后，访客点击该项会直接跳转。
        </p>
      </div>
      <md-filled-button :disabled="loading || saving" @click="save">
        <md-icon slot="icon">save</md-icon>
        {{ saving ? '保存中' : '保存' }}
      </md-filled-button>
    </header>

    <div v-if="loading" class="m3-loading">
      <md-circular-progress indeterminate aria-label="加载中" />
      <span class="m3-body-medium">加载中...</span>
    </div>

    <div v-else class="social-list">
      <section v-for="item in items" :key="item.key" class="social-row">
        <div class="social-heading">
          <md-icon>link</md-icon>
          <div>
            <div class="m3-title-medium text-on-surface">{{ item.label }}</div>
            <div class="m3-body-small text-on-surface-variant">{{ item.key }}</div>
          </div>
          <label class="visibility-switch">
            <span class="m3-label-large">显示</span>
            <md-switch
              :selected="item.enabled"
              @change="(event: any) => (item.enabled = event.target.selected)"
            />
          </label>
        </div>

        <div class="field-grid">
          <md-outlined-text-field
            label="显示名称"
            :value="item.label"
            @input="(event: any) => (item.label = event.target.value)"
          />
          <md-outlined-text-field
            label="账号 / 显示内容"
            :value="item.value"
            @input="(event: any) => (item.value = event.target.value)"
          />
          <md-outlined-text-field
            label="跳转链接"
            type="url"
            placeholder="https://..."
            :value="item.url"
            @input="(event: any) => (item.url = event.target.value)"
          />
          <md-outlined-text-field
            label="Vant 图标"
            :value="item.icon"
            @input="(event: any) => (item.icon = event.target.value)"
          />
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.social-list {
  border-top: 1px solid var(--md-sys-color-outline-variant);
}
.social-row {
  padding: 20px 0 24px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}
.social-heading {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.social-heading > md-icon {
  color: var(--md-sys-color-primary);
}
.visibility-switch {
  display: flex;
  align-items: center;
  gap: 10px;
}
.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.field-grid md-outlined-text-field {
  width: 100%;
}
@media (max-width: 720px) {
  .field-grid {
    grid-template-columns: 1fr;
  }
}
</style>
