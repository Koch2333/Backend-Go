export function bufToB64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf)
  let bin = ''
  for (const b of bytes) bin += String.fromCharCode(b)
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

export function b64urlToBuf(b64: string): ArrayBuffer {
  const padded = b64
    .replace(/-/g, '+')
    .replace(/_/g, '/')
    .padEnd(b64.length + ((4 - (b64.length % 4)) % 4), '=')
  const bin = atob(padded)
  const buf = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i)
  return buf.buffer
}

export interface RegBeginOptions {
  challenge: string
  rpId: string
  rpName: string
  userId: string
  userName: string
  userDisplayName: string
  timeout: number
  excludeCredentialIds: string[]
  sessionId: string
}

export interface LoginBeginOptions {
  challenge: string
  rpId: string
  timeout: number
  allowCredentials: Array<{ type: string; id: string }>
  sessionId: string
}

export async function createCredential(opts: RegBeginOptions): Promise<object> {
  const cred = (await navigator.credentials.create({
    publicKey: {
      challenge: b64urlToBuf(opts.challenge),
      rp: { id: opts.rpId, name: opts.rpName },
      user: {
        id: b64urlToBuf(opts.userId),
        name: opts.userName,
        displayName: opts.userDisplayName,
      },
      pubKeyCredParams: [
        { alg: -7, type: 'public-key' as const },
        { alg: -257, type: 'public-key' as const },
      ],
      timeout: opts.timeout,
      excludeCredentials: (opts.excludeCredentialIds ?? []).map((id) => ({
        id: b64urlToBuf(id),
        type: 'public-key' as const,
      })),
      authenticatorSelection: { residentKey: 'preferred', userVerification: 'preferred' },
      attestation: 'none',
    },
  })) as PublicKeyCredential
  if (!cred) throw new Error('navigator.credentials.create returned null')
  const r = cred.response as AuthenticatorAttestationResponse
  return {
    id: cred.id,
    rawId: bufToB64url(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: bufToB64url(r.clientDataJSON),
      attestationObject: bufToB64url(r.attestationObject),
    },
  }
}

export async function getCredential(opts: LoginBeginOptions): Promise<object> {
  const cred = (await navigator.credentials.get({
    publicKey: {
      challenge: b64urlToBuf(opts.challenge),
      rpId: opts.rpId,
      timeout: opts.timeout,
      allowCredentials: (opts.allowCredentials ?? []).map((c) => ({
        id: b64urlToBuf(c.id),
        type: 'public-key' as const,
      })),
      userVerification: 'preferred',
    },
  })) as PublicKeyCredential
  if (!cred) throw new Error('navigator.credentials.get returned null')
  const r = cred.response as AuthenticatorAssertionResponse
  return {
    id: cred.id,
    rawId: bufToB64url(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: bufToB64url(r.clientDataJSON),
      authenticatorData: bufToB64url(r.authenticatorData),
      signature: bufToB64url(r.signature),
      userHandle: r.userHandle ? bufToB64url(r.userHandle) : '',
    },
  }
}
