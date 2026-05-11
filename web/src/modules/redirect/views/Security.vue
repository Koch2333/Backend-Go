<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { showSuccessToast, showFailToast } from 'vant'
import { toDataURL } from 'qrcode'
import { extractMessage } from '@/shell/http'
import { createCredential } from '@/shell/webauthn'
import {
  getTOTPStatus,
  setupTOTP,
  enableTOTP,
  disableTOTP,
  listPasskeys,
  beginPasskeyRegister,
  finishPasskeyRegister,
  deletePasskey,
  type PasskeyInfo,
} from '../api'

// ----- TOTP -----
const totpEnabled = ref(false)
const totpSetup = ref<{ uri: string; secret: string } | null>(null)
const totpQR = ref('')
const totpCode = ref('')
const totpWorking = ref(false)

onMounted(async () => {
  try {
    const s = await getTOTPStatus()
    totpEnabled.value = s.enabled
  } catch {}
  await loadPasskeys()
})

async function startTOTPSetup() {
  totpWorking.value = true
  try {
    const r = await setupTOTP()
    totpSetup.value = r
    totpQR.value = await toDataURL(r.uri, { width: 200, margin: 2 })
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    totpWorking.value = false
  }
}

async function confirmTOTPEnable() {
  const code = totpCode.value.trim()
  if (!code) return
  totpWorking.value = true
  try {
    await enableTOTP(code)
    totpEnabled.value = true
    totpSetup.value = null
    totpQR.value = ''
    totpCode.value = ''
    showSuccessToast('TOTP 已启用')
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    totpWorking.value = false
  }
}

async function handleDisableTOTP() {
  totpWorking.value = true
  try {
    await disableTOTP()
    totpEnabled.value = false
    showSuccessToast('TOTP 已关闭')
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    totpWorking.value = false
  }
}

// ----- Passkeys -----
const passkeys = ref<PasskeyInfo[]>([])
const passkeyWorking = ref(false)
const showAddDialog = ref(false)
const newKeyName = ref('')

async function loadPasskeys() {
  try {
    const r = await listPasskeys()
    passkeys.value = r.items ?? []
  } catch {}
}

async function addPasskey() {
  const name = newKeyName.value.trim()
  if (!name) return
  passkeyWorking.value = true
  try {
    const begin = await beginPasskeyRegister()
    const credential = await createCredential(begin)
    await finishPasskeyRegister(begin.sessionId, name, credential)
    showAddDialog.value = false
    newKeyName.value = ''
    showSuccessToast('Passkey 已添加')
    await loadPasskeys()
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    passkeyWorking.value = false
  }
}

async function handleDeletePasskey(id: string) {
  passkeyWorking.value = true
  try {
    await deletePasskey(id)
    showSuccessToast('已删除')
    await loadPasskeys()
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    passkeyWorking.value = false
  }
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}
</script>

<template>
  <div class="mx-auto max-w-xl space-y-6 p-4">
    <!-- TOTP -->
    <van-cell-group inset>
      <van-cell
        title="动态验证码 (TOTP)"
        :value="totpEnabled ? '已启用 ✅' : '未启用'"
        :value-class="totpEnabled ? 'text-green-600' : 'text-gray-400'"
      />

      <!-- not enabled, no pending setup -->
      <template v-if="!totpEnabled && !totpSetup">
        <div class="px-4 pb-4">
          <van-button block round type="primary" :loading="totpWorking" @click="startTOTPSetup">
            开始设置
          </van-button>
        </div>
      </template>

      <!-- setup in progress -->
      <template v-if="totpSetup">
        <div class="px-4 py-3 space-y-3">
          <p class="text-sm text-gray-600">
            用 Google Authenticator、Authy 或其他 TOTP 应用扫描二维码
          </p>
          <div class="flex justify-center">
            <img
              v-if="totpQR"
              :src="totpQR"
              alt="TOTP QR"
              class="w-48 h-48 rounded border border-gray-200"
            />
          </div>
          <p class="text-xs text-gray-500">
            或手动输入密锁：
            <code class="bg-gray-100 px-1 rounded break-all select-all">{{ totpSetup.secret }}</code>
          </p>
          <van-field
            v-model="totpCode"
            type="digit"
            label="验证码"
            placeholder="输入 6 位验证码确认"
            maxlength="6"
          />
          <div class="flex gap-2">
            <van-button class="flex-1" round plain @click="totpSetup = null">取消</van-button>
            <van-button
              class="flex-1"
              round
              type="primary"
              :loading="totpWorking"
              @click="confirmTOTPEnable"
            >
              验证并启用
            </van-button>
          </div>
        </div>
      </template>

      <!-- enabled -->
      <template v-if="totpEnabled">
        <div class="px-4 pb-4">
          <van-button block round type="danger" :loading="totpWorking" @click="handleDisableTOTP">
            关闭 TOTP
          </van-button>
        </div>
      </template>
    </van-cell-group>

    <!-- Passkeys -->
    <van-cell-group inset>
      <van-cell title="Passkey / 安全密鑰">
        <template #right-icon>
          <van-button size="small" type="primary" @click="showAddDialog = true">
            + 添加
          </van-button>
        </template>
      </van-cell>

      <van-cell v-if="passkeys.length === 0" title="尚未添加 Passkey" class="text-gray-400" />

      <van-swipe-cell v-for="pk in passkeys" :key="pk.id">
        <van-cell :title="pk.name" :label="'添加于 ' + fmtDate(pk.createdAt)" />
        <template #right>
          <van-button
            square
            type="danger"
            text="删除"
            :loading="passkeyWorking"
            @click="handleDeletePasskey(pk.id)"
          />
        </template>
      </van-swipe-cell>
    </van-cell-group>

    <!-- Add Passkey Dialog -->
    <van-dialog
      v-model:show="showAddDialog"
      title="添加 Passkey"
      :show-confirm-button="false"
      :close-on-click-overlay="!passkeyWorking"
    >
      <div class="p-4 space-y-3">
        <van-field
          v-model="newKeyName"
          label="名称"
          placeholder="如：iPhone Face ID、YubiKey"
        />
        <div class="flex gap-2">
          <van-button
            class="flex-1"
            round
            :disabled="passkeyWorking"
            @click="showAddDialog = false"
          >
            取消
          </van-button>
          <van-button
            class="flex-1"
            round
            type="primary"
            :loading="passkeyWorking"
            @click="addPasskey"
          >
            开始添加
          </van-button>
        </div>
      </div>
    </van-dialog>
  </div>
</template>
