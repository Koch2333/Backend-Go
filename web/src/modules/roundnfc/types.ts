export interface Badge {
  id: string
  title: string
  series?: string
  type?: string
  styleKey?: string
  imageUrl?: string
  description?: string
  serialNo?: string
  releasedAt?: string
  coserBinding?: BadgeCoserBinding
}

export interface BadgeCoserBinding {
  badgeId: string
  cn: string
  photoObjectKey: string
  deviceId?: string
  tagUid?: string
  writtenAt?: string
  createdAt?: string
  updatedAt?: string
}

export type RequestStatus = 'new' | 'handled' | 'rejected'

export interface PhotoRequest {
  id: string
  badgeId: string
  name: string
  contact: string
  message?: string
  status: RequestStatus
  createdAt: string
  updatedAt: string
}

export interface AutographRequest {
  id: string
  badgeId: string
  name: string
  contact: string
  target: string
  content: string
  status: RequestStatus
  createdAt: string
  updatedAt: string
}
