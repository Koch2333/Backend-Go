<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { showSuccessToast, showFailToast } from '@/shell/toast'
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
  <div class="mx-auto max-w-2xl space-y-5">
    <header class="m3-page-header">
      <div>
        <h1 class="m3-headline-medium text-on-surface">安全设置</h1>
        <p class="m3-body-medium text-on-surface-variant mt-1">
          管理 TOTP 与 Passkey 二步验证。
        </p>
      </div>
    </header>

    <section class="m3-card p-6">
      <div class="section-head">
        <div>
          <h2 class="m3-title-large text-on-surface">动态验证码 (TOTP)</h2>
          <p class="m3-body-medium text-on-surface-variant mt-1">
            登录时除密码外再输入 6 位动态码。
          </p>
        </div>
        <md-assist-chip
          :label="totpEnabled ? '已启用' : '未启用'"
          :class="totpEnabled ? 'chip-tertiary' : 'chip-muted'"
        />
      </div>

      <div v-if="!totpEnabled && !totpSetup" class="mt-4">
        <md-filled-button :disabled="totpWorking" @click="startTOTPSetup">
          开始设置
        </md-filled-button>
      </div>

      <div v-if="totpSetup" class="space-y-3 mt-4">
        <p class="m3-body-medium text-on-surface-variant">
          用 Google Authenticator、Authy 或其他 TOTP 应用扫描二维码
        </p>
        <div class="flex justify-center">
          <img
            v-if="totpQR"
            :src="totpQR"
            alt="TOTP QR"
            class="qr-img"
          />
        </div>
        <p class="m3-body-small text-on-surface-variant">
          或手动输入密锁：
          <code class="select-all break-all rounded px-1 secret-code">{{ totpSetup.secret }}</code>
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
          <md-filled-button class="flex-1" :disabled="totpWorking" @click="confirmTOTPEnable">
            验证并启用
          </md-filled-button>
        </div>
      </div>

      <div v-if="totpEnabled" class="mt-4">
        <md-outlined-button :disabled="totpWorking" @click="handleDisableTOTP">
          <md-icon slot="icon">lock_open</md-icon>
          关闭 TOTP
        </md-outlined-button>
      </div>
    </section>

    <section class="m3-card p-6">
      <div class="section-head">
        <div>
          <h2 class="m3-title-large text-on-surface">Passkey / 安全密钥</h2>
          <p class="m3-body-medium text-on-surface-variant mt-1">
            指纹、Face ID、YubiKey 等都可作为登录凭证。
          </p>
        </div>
        <md-filled-button @click="addDialog?.show()">
          <md-icon slot="icon">add</md-icon>
          添加
        </md-filled-button>
      </div>

      <div v-if="passkeys.length === 0" class="m3-empty pt-4 pb-2">
        <div class="m3-empty-icon"><md-icon>key</md-icon></div>
        <div class="m3-title-medium text-on-surface">尚未添加 Passkey</div>
        <div class="m3-body-medium text-on-surface-variant">
          点击右上角「添加」开始注册。
        </div>
      </div>

      <md-list v-else class="mt-2">
        <template v-for="(pk, i) in passkeys" :key="pk.id">
          <md-divider v-if="i > 0" />
          <md-list-item>
            <md-icon slot="start" class="row-icon">key</md-icon>
            <div slot="headline" class="m3-title-medium">{{ pk.name }}</div>
            <div slot="supporting-text" class="m3-body-medium">
              添加于 {{ fmtDate(pk.createdAt) }}
            </div>
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
      <form slot="content" id="add-passkey-form" method="dialog" class="dialog-form">
        <md-outlined-text-field
          label="名称（如 iPhone Face ID、YubiKey）"
          :value="newKeyName"
          @input="(e: any) => (newKeyName = e.target.value)"
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
.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.qr-img {
  height: 192px;
  width: 192px;
  border-radius: 16px;
  border: 1px solid var(--md-sys-color-outline-variant);
  background: white;
}
.secret-code {
  background: var(--md-sys-color-surface-container-high);
  color: var(--md-sys-color-on-surface);
}
.row-icon { color: var(--md-sys-color-primary); }
.dialog-form {
  padding-top: 8px;
}
.dialog-form md-outlined-text-field { width: 100%; }
md-outlined-text-field { width: 100%; }
</style>
