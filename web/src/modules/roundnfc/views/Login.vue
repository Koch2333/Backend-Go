<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from '@/shell/toast'
import { extractMessage } from '@/shell/http'
import { getCredential } from '@/shell/webauthn'
import { login, beginPasskeyLogin, finishPasskeyLogin } from '../api'
import { ROUNDNFC } from '../core'

const form = reactive({ username: 'admin', password: '', totpCode: '' })
const submitting = ref(false)
const passkeyWorking = ref(false)
const showTOTP = ref(false)
const router = useRouter()
const route = useRoute()

const target = () => (route.query.from as string) || '/m/roundnfc/badges'

async function onSubmit() {
  submitting.value = true
  try {
    const r = await login(form.username, form.password, showTOTP.value ? form.totpCode : undefined)
    if (r.needsTOTP) {
      showTOTP.value = true
      return
    }
    ROUNDNFC.useAuth().set(r.token!, r.username!, r.expiresAt!)
    showSuccessToast('登录成功')
    router.replace(target())
  } catch (err) {
    showFailToast(extractMessage(err))
  } finally {
    submitting.value = false
  }
}

async function loginWithPasskey() {
  passkeyWorking.value = true
  try {
    const begin = await beginPasskeyLogin(form.username)
    const credential = await getCredential(begin)
    const r = await finishPasskeyLogin(begin.sessionId, credential)
    ROUNDNFC.useAuth().set(r.token!, r.username!, r.expiresAt!)
    showSuccessToast('登录成功')
    router.replace(target())
  } catch (err) {
    showFailToast(extractMessage(err))
  } finally {
    passkeyWorking.value = false
  }
}
</script>

<template>
  <main class="login-shell">
    <div class="login-blob blob-a" />
    <div class="login-blob blob-b" />
    <form class="m3-card login-card" @submit.prevent="onSubmit">
      <div class="logo m3-display-medium">RoundNFC</div>
      <p class="m3-body-medium text-on-surface-variant logo-sub">
        后台账号在 config/roundnfc/.env 里配置
      </p>

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
        <md-outlined-text-field
          v-if="showTOTP"
          label="动态验证码"
          type="number"
          maxlength="6"
          :value="form.totpCode"
          @input="(e: any) => (form.totpCode = e.target.value)"
          required
        />
      </div>

      <div class="actions">
        <md-filled-button type="submit" :disabled="submitting">
          {{ submitting ? '登录中…' : showTOTP ? '验证登录' : '登录' }}
        </md-filled-button>
        <md-outlined-button type="button" :disabled="passkeyWorking" @click="loginWithPasskey">
          <md-icon slot="icon">fingerprint</md-icon>
          {{ passkeyWorking ? '请在系统弹窗中确认…' : '使用 Passkey 登录' }}
        </md-outlined-button>
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
.logo-sub { text-align: center; margin-top: 4px; }
.fields { margin-top: 28px; display: flex; flex-direction: column; gap: 16px; }
.actions { margin-top: 24px; display: flex; flex-direction: column; gap: 12px; }
md-outlined-text-field { width: 100%; }
md-filled-button, md-outlined-button { width: 100%; }
</style>
