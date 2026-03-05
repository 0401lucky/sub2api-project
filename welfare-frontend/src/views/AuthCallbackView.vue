<template>
  <section class="panel callback-panel">
    <p class="kicker">OAuth Callback</p>
    <h2>{{ error ? '登录失败' : '正在建立登录会话' }}</h2>

    <p v-if="error" class="error">{{ error }}</p>
    <template v-else>
      <p class="text-muted">正在同步你的授权信息并拉取账号资料，请稍候。</p>
      <div class="loader" aria-hidden="true">
        <span></span>
        <span></span>
        <span></span>
      </div>
    </template>

    <button v-if="error" class="btn-secondary" @click="goLogin">返回登录页</button>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../store/auth'
import { sanitizeAuthRedirect } from '../utils/authRedirect'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const error = ref('')

function resolveOAuthErrorMessage(errCode: string): string {
  if (errCode === 'user_not_found') {
    return '请先在 sub2api 完成首次登录或注册'
  }
  return '登录失败，请稍后重试'
}

function goLogin(): void {
  void router.replace('/login')
}

onMounted(async () => {
  const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : window.location.hash
  const params = new URLSearchParams(hash)
  const errCode = params.get('error') || ''
  const oauthCode = typeof route.query.code === 'string' ? route.query.code.trim() : ''
  const oauthState = typeof route.query.state === 'string' ? route.query.state.trim() : ''
  const rawRedirect = params.get('redirect') || (typeof route.query.redirect === 'string' ? route.query.redirect : '/')
  const redirect = sanitizeAuthRedirect(rawRedirect)

  if (errCode) {
    error.value = resolveOAuthErrorMessage(errCode)
    return
  }

  if (oauthCode || oauthState) {
    error.value = '登录回调配置错误：请将 LinuxDo 回调地址配置为后端 /api/v1/auth/linuxdo/callback'
    return
  }

  try {
    await auth.loadProfile()
    if (!auth.state.profile) {
      error.value = '登录状态建立失败，请重新登录'
      return
    }
    await router.replace(redirect)
  } catch {
    error.value = '登录失败，请稍后重试'
  }
})
</script>

<style scoped>
.callback-panel {
  max-width: 640px;
  margin: 0 auto;
  padding: clamp(24px, 4vw, 34px);
  display: grid;
  gap: 14px;
}

.callback-panel h2 {
  margin: 0;
  font-family: 'Noto Serif SC', 'STSong', serif;
  font-size: clamp(25px, 3vw, 32px);
}

.loader {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 32px;
}

.loader span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #78e4d7;
  animation: bounce 1.05s ease-in-out infinite;
}

.loader span:nth-child(2) {
  animation-delay: 0.14s;
}

.loader span:nth-child(3) {
  animation-delay: 0.28s;
}

@keyframes bounce {
  0%,
  80%,
  100% {
    transform: translateY(0);
    opacity: 0.35;
  }
  40% {
    transform: translateY(-8px);
    opacity: 1;
  }
}
</style>
