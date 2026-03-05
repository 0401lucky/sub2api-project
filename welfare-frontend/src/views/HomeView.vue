<template>
  <section class="card" v-if="auth.state.profile">
    <h2>用户信息</h2>
    <p>LinuxDo：{{ auth.state.profile.linuxdo_name }} ({{ auth.state.profile.linuxdo_subject }})</p>
    <p>sub2api：{{ auth.state.profile.sub2api_email }} / ID {{ auth.state.profile.sub2api_user_id }}</p>
    <button @click="logout">退出登录</button>
  </section>

  <section class="card" v-if="status">
    <h2>今日签到</h2>
    <p>日期：{{ status.date }}（{{ status.campaign.timezone }}）</p>
    <p>奖励区间：{{ status.campaign.reward_min }} ~ {{ status.campaign.reward_max }}</p>
    <p v-if="status.blocked" class="error">当前账号已被风控限制</p>
    <p v-if="status.checked_in" class="success">今天已签到，获得 {{ status.amount }}</p>
    <p v-if="notice" :class="noticeClass">{{ notice }}</p>
    <button :disabled="!status.can_checkin || submitting" @click="checkin">
      {{ submitting ? '处理中...' : '立即签到' }}
    </button>
  </section>

  <section class="card">
    <h2>签到记录</h2>
    <table class="table">
      <thead>
        <tr>
          <th>ID</th>
          <th>日期</th>
          <th>额度</th>
          <th>状态</th>
          <th>时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in history" :key="item.id">
          <td>{{ item.id }}</td>
          <td>{{ item.checkin_date }}</td>
          <td>{{ item.amount }}</td>
          <td>{{ item.status }}</td>
          <td>{{ item.created_at }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiRequest } from '../api/client'
import { useAuthStore } from '../store/auth'

interface CheckinStatus {
  date: string
  can_checkin: boolean
  checked_in: boolean
  grant_status?: string
  amount?: number
  blocked: boolean
  campaign: {
    timezone: string
    reward_min: number
    reward_max: number
  }
}

interface HistoryItem {
  id: number
  checkin_date: string
  amount: number
  status: string
  created_at: string
}

const router = useRouter()
const auth = useAuthStore()
const status = ref<CheckinStatus | null>(null)
const history = ref<HistoryItem[]>([])
const notice = ref('')
const noticeType = ref<'success' | 'error' | 'warn'>('warn')
const submitting = ref(false)

const noticeClass = computed(() => {
  if (noticeType.value === 'success') return 'success'
  if (noticeType.value === 'error') return 'error'
  return 'notice'
})

async function loadAll(): Promise<void> {
  status.value = await apiRequest<CheckinStatus>('/checkin/status')
  const res = await apiRequest<{ items: HistoryItem[] }>('/checkin/history?page=1&page_size=20')
  history.value = res.items
}

async function checkin(): Promise<void> {
  submitting.value = true
  notice.value = ''
  try {
    const res = await apiRequest<{ amount: number; message: string; status: string }>('/checkin/daily', { method: 'POST' })
    notice.value = `${res.message}，发放额度 ${res.amount}`
    noticeType.value = 'success'
  } catch (e) {
    notice.value = (e as Error).message
    noticeType.value = 'error'
  } finally {
    submitting.value = false
    await loadAll()
  }
}

async function logout(): Promise<void> {
  auth.clearToken()
  await router.replace('/login')
}

onMounted(async () => {
  try {
    await loadAll()
  } catch (e) {
    notice.value = (e as Error).message
    noticeType.value = 'error'
  }
})
</script>
