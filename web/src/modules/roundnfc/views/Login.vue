<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from 'vant'
import { extractMessage } from '@/shell/http'
import { login } from '../api'
import { ROUNDNFC } from '../core'

const form = reactive({ username: 'admin', password: '' })
const submitting = ref(false)
const router = useRouter()
const route = useRoute()

async function onSubmit() {
  submitting.value = true
  try {
    const r = await login(form.username, form.password)
    ROUNDNFC.useAuth().set(r.token, r.username, r.expiresAt)
    showSuccessToast('登录成功')
    const from = (route.query.from as string) || '/m/roundnfc/badges'
    router.replace(from)
  } catch (err) {
    showFailToast(extractMessage(err))
  } finally {
    submitting.value = false
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
        </van-cell-group>
        <div class="px-4 pt-4">
          <van-button
            block
            round
            type="primary"
            native-type="submit"
            :loading="submitting"
          >
            登录
          </van-button>
        </div>
      </van-form>
    </div>
  </main>
</template>
