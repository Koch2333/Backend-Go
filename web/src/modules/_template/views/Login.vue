<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { login } from '../api'
import { M } from '../core'
import BackendSwitcher from '@/shell/BackendSwitcher.vue'

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
  <main class="login-shell">
    <div class="login-blob blob-a" />
    <div class="login-blob blob-b" />
    <form class="m3-card login-card" @submit.prevent="onSubmit">
      <div class="logo m3-display-medium">Template</div>

      <BackendSwitcher :module-name="M.name" class="backend-row" />

      <div class="fields">
        <md-outlined-text-field
          label="用户名"
          :value="form.username"
          @input="(e: any) => (form.username = e.target.value)"
          required
        />
        <md-outlined-text-field
          label="密码"
          type="password"
          :value="form.password"
          @input="(e: any) => (form.password = e.target.value)"
          required
        />
      </div>

      <div class="actions">
        <md-filled-button type="submit" :disabled="submitting">
          {{ submitting ? '登录中…' : '登录' }}
        </md-filled-button>
      </div>
    </form>
  </main>
</template>

<style scoped>
.login-shell {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: var(--md-sys-color-surface-container-lowest);
  overflow: hidden;
}
.login-blob {
  position: absolute;
  width: 480px;
  height: 480px;
  border-radius: 50%;
  filter: blur(8px);
  pointer-events: none;
  opacity: 0.55;
}
.blob-a {
  top: -160px;
  left: -120px;
  background: radial-gradient(circle, var(--md-sys-color-primary-container) 0%, transparent 70%);
}
.blob-b {
  bottom: -180px;
  right: -140px;
  background: radial-gradient(circle, var(--md-sys-color-tertiary-container) 0%, transparent 70%);
}
.login-card {
  position: relative;
  width: 100%;
  max-width: 400px;
  padding: 32px;
  box-shadow: var(--md-elevation-2);
}
.logo {
  background: linear-gradient(
    135deg,
    var(--md-sys-color-primary) 0%,
    var(--md-sys-color-tertiary) 100%
  );
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  font-weight: 500;
  font-size: 36px;
  line-height: 44px;
  text-align: center;
}
.backend-row { margin-top: 20px; }
.fields { margin-top: 20px; display: flex; flex-direction: column; gap: 16px; }
.actions { margin-top: 24px; display: flex; flex-direction: column; gap: 12px; }
md-outlined-text-field, md-filled-button { width: 100%; }
</style>
