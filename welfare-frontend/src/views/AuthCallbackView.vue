<template>
  <section class="card">
    <h2>登录回调处理中...</h2>
    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../store/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const error = ref('')

onMounted(async () => {
  const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : window.location.hash
  const params = new URLSearchParams(hash)
  const token = params.get('access_token') || ''
  const errCode = params.get('error')
  const errMsg = params.get('error_description')
  const redirect = params.get('redirect') || (route.query.redirect as string) || '/'

  if (errCode) {
    error.value = errMsg || errCode
    return
  }
  if (!token) {
    error.value = '回调未返回 access_token'
    return
  }

  auth.saveToken(token)
  await auth.loadProfile()
  await router.replace(redirect)
})
</script>
