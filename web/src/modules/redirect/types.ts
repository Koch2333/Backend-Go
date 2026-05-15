export interface RedirectRule {
  name: string
  targetUrl: string
  enabled: boolean
  updatedAt: string
}

export interface NFCCard {
  hwid: string
  isRegistered: boolean
  userId: string
  updatedAt: string
}
