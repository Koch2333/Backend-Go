<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { login } from '../api'
import { M } from '../core'

const form = reactive({ username: '', password: '' })
const submitting = ref(false)
const router = useRouter()
const route = useRoute()

async function onSubmit() {
  submitting.value = true
  try {
    const r = await login(form.username, form.password)
    M.useAuth().set(r.token, r.username, r.expiresAt)
    showSuccessToast('登录成功')
    const from = (route.query.from as string) || `/m/${M.name}`
    router.replace(from)
  } catch (err) {
    showFailToast(extractMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-surface-container px-6">
    <form
      class="m3-card w-full max-w-sm rounded-3xl p-6 shadow"
      @submit.prevent="onSubmit"
    >
      <h1 class="text-center text-base font-medium text-on-surface">Template 后台</h1>

      <div class="mt-6 space-y-4">
        <md-outlined-text-field
          label="用户名"
          :value="form.username"
          @input="(e: any) => (form.username = e.target.value)"
          required
          class="w-full"
        />
        <md-outlined-text-field
          label="密码"
          type="password"
          :value="form.password"
          @input="(e: any) => (form.password = e.target.value)"
          required
          class="w-full"
        />
      </div>

      <div class="mt-6">
        <md-filled-button type="submit" :disabled="submitting" class="w-full">
          {{ submitting ? '登录中…' : '登录' }}
        </md-filled-button>
      </div>
    </form>
  </main>
</template>

<style scoped>
.bg-surface-container {
  background: #eef0f3;
}
.m3-card {
  background: #fff;
}
.text-on-surface {
  color: #1a1c1e;
}
md-outlined-text-field,
md-filled-button {
  width: 100%;
}
</style>
