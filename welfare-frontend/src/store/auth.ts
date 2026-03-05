import { reactive } from 'vue'
import { apiRequest } from '../api/client'

export interface Profile {
  linuxdo_subject: string
  linuxdo_name: string
  sub2api_user_id: number
  sub2api_email: string
  is_admin: boolean
}

const state = reactive({
  profile: null as Profile | null,
  loaded: false
})

async function loadProfile(): Promise<void> {
  try {
    state.profile = await apiRequest<Profile>('/auth/me')
  } catch {
    state.profile = null
  } finally {
    state.loaded = true
  }
}

async function logout(): Promise<void> {
  try {
    await apiRequest('/auth/logout', { method: 'POST' })
  } catch {
  } finally {
    clearSession()
  }
}

function clearSession(): void {
  state.profile = null
  state.loaded = true
}

export function useAuthStore() {
  return {
    state,
    loadProfile,
    logout,
    clearSession
  }
}
