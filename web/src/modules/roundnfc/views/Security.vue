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
  listAppTokens,
  createAppToken,
  setAppTokenEnabled,
  deleteAppToken,
  type PasskeyInfo,
  type AppToken,
  type AppPairingConfig,
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
  await loadAppTokens()
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

const appTokens = ref<AppToken[]>([])
const appTokenWorking = ref(false)
const appTokenDialog = ref<HTMLDialogElement & { show: () => void; close: () => void } | null>(null)
const newAppTokenName = ref('')
const createdAppToken = ref('')
const pairingConfig = ref<AppPairingConfig | null>(null)
const pairingQR = ref('')

async function loadAppTokens() {
  try {
    const r = await listAppTokens()
    appTokens.value = r.items ?? []
  } catch {
    /* */
  }
}

async function addAppToken() {
  const name = newAppTokenName.value.trim()
  if (!name) return
  appTokenWorking.value = true
  try {
    const r = await createAppToken(name)
    createdAppToken.value = r.token
    pairingConfig.value = r.pairing
    pairingQR.value = await toDataURL(JSON.stringify(r.pairing), { width: 240, margin: 2 })
    newAppTokenName.value = ''
    showSuccessToast('Android App 配对码已创建')
    await loadAppTokens()
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    appTokenWorking.value = false
  }
}

async function copyPairingConfig() {
  if (!pairingConfig.value) return
  await navigator.clipboard.writeText(JSON.stringify(pairingConfig.value))
  showSuccessToast('已复制配对信息')
}

async function copyAppToken() {
  if (!createdAppToken.value) return
  await navigator.clipboard.writeText(createdAppToken.value)
  showSuccessToast('已复制令牌')
}

function closeAppTokenDialog() {
  appTokenDialog.value?.close()
  createdAppToken.value = ''
  pairingConfig.value = null
  pairingQR.value = ''
}

async function toggleAppToken(token: AppToken) {
  appTokenWorking.value = true
  try {
    await setAppTokenEnabled(token.id, !token.enabled)
    showSuccessToast(token.enabled ? '令牌已停用' : '令牌已启用')
    await loadAppTokens()
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    appTokenWorking.value = false
  }
}

async function handleDeleteAppToken(id: string) {
  appTokenWorking.value = true
  try {
    await deleteAppToken(id)
    showSuccessToast('令牌已删除')
    await loadAppTokens()
  } catch (e) {
    showFailToast(extractMessage(e))
  } finally {
    appTokenWorking.value = false
  }
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
          <h2 class="m3-title-large text-on-surface">Android App 配对</h2>
          <p class="m3-body-medium text-on-surface-variant mt-1">
            生成写卡 App 使用的独立密钥，扫码后自动连接当前后端。
          </p>
        </div>
        <md-filled-button @click="appTokenDialog?.show()">
          <md-icon slot="icon">qr_code_2</md-icon>
          配对
        </md-filled-button>
      </div>

      <div v-if="appTokens.length === 0" class="token-empty m3-body-medium text-on-surface-variant">
        还没有 Android App 配对项。
      </div>

      <md-list v-else class="mt-2">
        <template v-for="(token, i) in appTokens" :key="token.id">
          <md-divider v-if="i > 0" />
          <md-list-item>
            <md-icon slot="start" class="row-icon">phone_android</md-icon>
            <div slot="headline" class="m3-title-medium">{{ token.name }}</div>
            <div slot="supporting-text" class="m3-body-medium">
              {{ token.tokenPrefix }}... · 创建于 {{ fmtDate(token.createdAt) }}
              <span v-if="token.lastUsedAt"> · 最近使用 {{ fmtDate(token.lastUsedAt) }}</span>
            </div>
            <md-assist-chip
              slot="end"
              :label="token.enabled ? '启用中' : '已停用'"
              :class="token.enabled ? 'chip-tertiary' : 'chip-muted'"
            />
            <md-icon-button
              slot="end"
              :disabled="appTokenWorking"
              :aria-label="token.enabled ? '停用' : '启用'"
              @click="toggleAppToken(token)"
            >
              <md-icon>{{ token.enabled ? 'pause_circle' : 'play_circle' }}</md-icon>
            </md-icon-button>
            <md-icon-button
              slot="end"
              :disabled="appTokenWorking"
              aria-label="删除"
              @click="handleDeleteAppToken(token.id)"
            >
              <md-icon>delete</md-icon>
            </md-icon-button>
          </md-list-item>
        </template>
      </md-list>
    </section>

    <section class="m3-card p-6">
      <div class="section-head">
        <div>
          <h2 class="m3-title-large text-on-surface">Passkey / 安全密钥</h2>
          <p class="m3-body-medium text-on-surface-variant mt-1">
            指纹、Face ID、YubiKey 等都可作为登录凭证，跨域部署时尤其推荐。
          </p>
        </div>
        <md-filled-button v-if="passkeys.length > 0" @click="addDialog?.show()">
          <md-icon slot="icon">add</md-icon>
          添加
        </md-filled-button>
      </div>

      <div v-if="passkeys.length === 0" class="passkey-empty">
        <div class="passkey-empty-icon"><md-icon>key</md-icon></div>
        <div class="m3-title-medium text-on-surface passkey-empty-title">
          为这个账号创建第一个 Passkey
        </div>
        <div class="m3-body-medium text-on-surface-variant passkey-empty-body">
          无需密码 · 一键登录 · TouchID / FaceID / USB key 都可以。
        </div>
        <md-filled-button class="passkey-empty-btn" @click="addDialog?.show()">
          <md-icon slot="icon">add</md-icon>
          创建 Passkey
        </md-filled-button>
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
          <img v-if="totpQR" :src="totpQR" alt="TOTP QR" class="qr-img" />
        </div>
        <p class="m3-body-small text-on-surface-variant">
          或手动输入密锁：
          <code class="select-all break-all rounded px-1 secret-code">{{ totpSetup.secret }}</code>
        </p>
        <md-outlined-text-field
          label="6 位验证码" type="number" maxlength="6"
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

    <md-dialog ref="appTokenDialog">
      <div slot="headline">创建 Android App 配对码</div>
      <form slot="content" id="add-app-token-form" method="dialog" class="dialog-form">
        <md-outlined-text-field
          label="名称（如 写卡手机、备用机）"
          :value="newAppTokenName"
          @input="(e: any) => (newAppTokenName = e.target.value)"
        />
        <div v-if="pairingQR" class="pairing-created">
          <img :src="pairingQR" alt="Android pairing QR" class="pairing-qr" />
          <p class="m3-body-medium text-on-surface">
            完整令牌只显示这一次，请用 Android 写卡 App 扫码保存。
          </p>
          <code class="token-value">{{ createdAppToken }}</code>
        </div>
      </form>
      <div slot="actions">
        <md-text-button :disabled="appTokenWorking" @click="closeAppTokenDialog">
          关闭
        </md-text-button>
        <md-outlined-button v-if="pairingConfig" @click="copyPairingConfig">
          复制配对信息
        </md-outlined-button>
        <md-outlined-button v-if="createdAppToken" @click="copyAppToken">
          复制令牌
        </md-outlined-button>
        <md-filled-button
          v-if="!pairingConfig"
          :disabled="appTokenWorking || !newAppTokenName.trim()"
          @click="addAppToken"
        >
          创建
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
.passkey-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 24px 12px 12px;
  gap: 12px;
}
.passkey-empty-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
}
.passkey-empty-icon md-icon { font-size: 36px; }
.passkey-empty-title { margin-top: 4px; }
.passkey-empty-body { max-width: 360px; }
.passkey-empty-btn { margin-top: 8px; }
.token-empty {
  padding: 24px 0 4px;
}
.pairing-created {
  margin-top: 16px;
  display: grid;
  justify-items: center;
  gap: 12px;
}
.pairing-qr {
  height: 240px;
  width: 240px;
  border-radius: 16px;
  border: 1px solid var(--md-sys-color-outline-variant);
  background: white;
}
.token-value {
  display: block;
  width: 100%;
  padding: 12px;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container-high);
  color: var(--md-sys-color-on-surface);
  word-break: break-all;
}
.dialog-form { padding-top: 8px; }
.dialog-form md-outlined-text-field { width: 100%; }
md-outlined-text-field { width: 100%; }
</style>
