<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from 'vant'
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
  <main class="flex min-h-screen items-center justify-center bg-gray-100 px-6">
    <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow">
      <h1 class="text-center text-base font-semibold text-gray-800">RoundNFC 后台</h1>
      <p class="mt-1 text-center text-xs text-gray-400">
        账号在后端 config/roundnfc/.env 里配置
      </p>
      <van-form class="mt-4" @submit="onSubmit">
        <van-cell-group inset>
          <van-field
            v-model="form.username"
            label="用户名"
            placeholder="admin"
            :rules="[{ required: true, message: '请输入用户名' }]"
            required
          />
          <van-field
            v-model="form.password"
            type="password"
            label="密码"
            placeholder="对应 PASSWORD_HASH 的明文"
            :rules="[{ required: true, message: '请输入密码' }]"
            required
          />
          <van-field
            v-if="showTOTP"
            v-model="form.totpCode"
            type="digit"
            label="动态验证码"
            placeholder="6 位 TOTP 验证码"
            maxlength="6"
            :rules="[{ required: true, message: '请输入验证码' }]"
            required
          />
        </van-cell-group>
        <div class="px-4 pt-4 space-y-3">
          <van-button
            block
            round
            type="primary"
            native-type="submit"
            :loading="submitting"
          >
            {{ showTOTP ? '验证登录' : '登录' }}
          </van-button>
          <van-button
            block
            round
            plain
            type="primary"
            :loading="passkeyWorking"
            @click.prevent="loginWithPasskey"
          >
            使用 Passkey 登录
          </van-button>
        </div>
      </van-form>
    </div>
  </main>
</template>
