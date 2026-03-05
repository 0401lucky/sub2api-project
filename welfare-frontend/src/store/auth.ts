import { reactive } from 'vue'
import { apiRequest, clearAccessToken, getAccessToken, setAccessToken } from '../api/client'

export interface Profile {
  linuxdo_subject: string
  linuxdo_name: string
  sub2api_user_id: number
  sub2api_email: string
  is_admin: boolean
}

const state = reactive({
  token: getAccessToken(),
  profile: null as Profile | null,
  loaded: false
})

async function loadProfile(): Promise<void> {
  if (!state.token) {
    state.profile = null
    state.loaded = true
    return
  }
  try {
    state.profile = await apiRequest<Profile>('/auth/me')
  } catch {
    clearToken()
  } finally {
    state.loaded = true
  }
}

function saveToken(token: string): void {
  state.token = token
  setAccessToken(token)
}

function clearToken(): void {
  state.token = ''
  state.profile = null
  clearAccessToken()
}

export function useAuthStore() {
  return {
    state,
    loadProfile,
    saveToken,
    clearToken
  }
}
