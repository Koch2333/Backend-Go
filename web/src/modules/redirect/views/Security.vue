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
  } catch {
    /* */
  }
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
const addDialog = ref<HTMLDialogElement & { show: () => void; close: () => void } | null>(null)
const newKeyName = ref('')

async function loadPasskeys() {
  try {
    const r = await listPasskeys()
    passkeys.value = r.items ?? []
  } catch {
    /* */
  }
}

async function addPasskey() {
  const name = newKeyName.value.trim()
  if (!name) return
  passkeyWorking.value = true
  try {
    const begin = await beginPasskeyRegister()
    const credential = await createCredential(begin)
    await finishPasskeyRegister(begin.sessionId, name, credential)
    addDialog.value?.close()
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
  <div class="mx-auto max-w-xl space-y-4 p-3">
    <!-- TOTP section -->
    <section class="m3-card rounded-3xl bg-white p-4">
      <header class="flex items-center justify-between pb-2">
        <h2 class="text-base font-medium">动态验证码 (TOTP)</h2>
        <md-assist-chip
          :label="totpEnabled ? '已启用' : '未启用'"
          :class="totpEnabled ? 'chip-success' : 'chip-muted'"
        />
      </header>

      <div v-if="!totpEnabled && !totpSetup">
        <md-filled-button :disabled="totpWorking" class="w-full" @click="startTOTPSetup">
          开始设置
        </md-filled-button>
      </div>

      <div v-if="totpSetup" class="space-y-3">
        <p class="text-sm text-gray-600">
          用 Google Authenticator、Authy 或其他 TOTP 应用扫描二维码
        </p>
        <div class="flex justify-center">
          <img
            v-if="totpQR"
            :src="totpQR"
            alt="TOTP QR"
            class="h-48 w-48 rounded-xl border border-gray-200"
          />
        </div>
        <p class="text-xs text-gray-500">
          或手动输入密锁：
          <code class="select-all break-all rounded bg-gray-100 px-1">{{ totpSetup.secret }}</code>
        </p>
        <md-outlined-text-field
          label="6 位验证码"
          type="number"
          maxlength="6"
          :value="totpCode"
          @input="(e: any) => (totpCode = e.target.value)"
          class="w-full"
        />
        <div class="flex gap-2">
          <md-text-button class="flex-1" @click="totpSetup = null">取消</md-text-button>
          <md-filled-button
            class="flex-1"
            :disabled="totpWorking"
            @click="confirmTOTPEnable"
          >
            验证并启用
          </md-filled-button>
        </div>
      </div>

      <div v-if="totpEnabled">
        <md-outlined-button
          :disabled="totpWorking"
          class="w-full"
          @click="handleDisableTOTP"
        >
          <md-icon slot="icon">lock_open</md-icon>
          关闭 TOTP
        </md-outlined-button>
      </div>
    </section>

    <!-- Passkeys section -->
    <section class="m3-card rounded-3xl bg-white p-4">
      <header class="flex items-center justify-between pb-2">
        <h2 class="text-base font-medium">Passkey / 安全密钥</h2>
        <md-filled-button @click="addDialog?.show()">
          <md-icon slot="icon">add</md-icon>
          添加
        </md-filled-button>
      </header>

      <p v-if="passkeys.length === 0" class="py-4 text-center text-sm text-gray-400">
        尚未添加 Passkey
      </p>

      <md-list v-else>
        <template v-for="(pk, i) in passkeys" :key="pk.id">
          <md-divider v-if="i > 0" />
          <md-list-item>
            <md-icon slot="start">key</md-icon>
            <div slot="headline">{{ pk.name }}</div>
            <div slot="supporting-text">添加于 {{ fmtDate(pk.createdAt) }}</div>
            <md-icon-button
              slot="end"
              :disabled="passkeyWorking"
              aria-label="删除"
              @click="handleDeletePasskey(pk.id)"
            >
              <md-icon>delete</md-icon>
            </md-icon-button>
          </md-list-item>
        </template>
      </md-list>
    </section>

    <md-dialog ref="addDialog">
      <div slot="headline">添加 Passkey</div>
      <form slot="content" id="add-passkey-form" method="dialog" class="pt-2">
        <md-outlined-text-field
          label="名称（如 iPhone Face ID、YubiKey）"
          :value="newKeyName"
          @input="(e: any) => (newKeyName = e.target.value)"
          class="w-full"
        />
      </form>
      <div slot="actions">
        <md-text-button :disabled="passkeyWorking" @click="addDialog?.close()">
          取消
        </md-text-button>
        <md-filled-button :disabled="passkeyWorking" @click="addPasskey">
          {{ passkeyWorking ? '请在系统弹窗中确认…' : '开始添加' }}
        </md-filled-button>
      </div>
    </md-dialog>
  </div>
</template>

<style scoped>
md-outlined-text-field {
  width: 100%;
}
.chip-success {
  --md-assist-chip-label-text-color: #146c43;
  --md-assist-chip-outline-color: #146c43;
}
.chip-muted {
  --md-assist-chip-label-text-color: #6b7280;
  --md-assist-chip-outline-color: #d1d5db;
}
</style>
