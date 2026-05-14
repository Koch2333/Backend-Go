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
  <main class="flex min-h-screen items-center justify-center bg-surface-container px-6">
    <form
      class="m3-card w-full max-w-sm rounded-3xl p-6 shadow"
      @submit.prevent="onSubmit"
    >
      <h1 class="m3-title text-center text-base font-medium text-on-surface">RoundNFC 后台</h1>
      <p class="mt-1 text-center text-xs text-on-surface-variant">
        账号在后端 config/roundnfc/.env 里配置
      </p>

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
        <md-outlined-text-field
          v-if="showTOTP"
          label="动态验证码"
          type="number"
          maxlength="6"
          :value="form.totpCode"
          @input="(e: any) => (form.totpCode = e.target.value)"
          required
          class="w-full"
        />
      </div>

      <div class="mt-6 space-y-3">
        <md-filled-button
          type="submit"
          :disabled="submitting"
          class="w-full"
        >
          {{ submitting ? '登录中…' : showTOTP ? '验证登录' : '登录' }}
        </md-filled-button>
        <md-outlined-button
          type="button"
          :disabled="passkeyWorking"
          class="w-full"
          @click="loginWithPasskey"
        >
          <md-icon slot="icon">fingerprint</md-icon>
          {{ passkeyWorking ? '请在系统弹窗中确认…' : '使用 Passkey 登录' }}
        </md-outlined-button>
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
.m3-title {
  font-family: Roboto, system-ui, sans-serif;
}
.text-on-surface {
  color: #1a1c1e;
}
.text-on-surface-variant {
  color: #44474e;
}
md-outlined-text-field,
md-filled-button,
md-outlined-button {
  width: 100%;
}
</style>
