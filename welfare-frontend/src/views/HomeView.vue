<template>
  <div class="view-stack">
    <section class="panel" v-if="auth.state.profile">
      <div class="section-head">
        <div>
          <p class="kicker">Profile</p>
          <h2>你好，{{ auth.state.profile.linuxdo_name }}</h2>
        </div>
        <button class="btn-secondary" @click="logout">退出登录</button>
      </div>

      <div class="info-grid">
        <article class="metric-card">
          <span>LinuxDo Subject</span>
          <strong>{{ auth.state.profile.linuxdo_subject }}</strong>
        </article>
        <article class="metric-card">
          <span>sub2api 邮箱</span>
          <strong>{{ auth.state.profile.sub2api_email }}</strong>
        </article>
        <article class="metric-card">
          <span>sub2api 用户 ID</span>
          <strong>#{{ auth.state.profile.sub2api_user_id }}</strong>
        </article>
      </div>
    </section>

    <section class="panel" v-if="status">
      <div class="section-head">
        <div>
          <p class="kicker">Daily Check-In</p>
          <h2>今日签到</h2>
        </div>
        <span class="inline-chip">{{ status.date }} · {{ status.campaign.timezone }}</span>
      </div>

      <div class="checkin-grid">
        <article class="metric-card">
          <span>奖励区间</span>
          <strong>{{ status.campaign.reward_min }} ~ {{ status.campaign.reward_max }}</strong>
        </article>
        <article class="metric-card">
          <span>今日状态</span>
          <strong>{{ statusTitle }}</strong>
        </article>
        <article class="metric-card">
          <span>发放结果</span>
          <strong>{{ status.amount ?? '--' }}</strong>
        </article>
      </div>

      <div class="action-block">
        <p v-if="status.blocked" class="error">当前账号已被风控限制</p>
        <p v-else-if="status.checked_in" class="success">今天已签到，获得 {{ status.amount ?? '--' }}</p>
        <p v-if="notice" :class="noticeClass">{{ notice }}</p>
        <button :disabled="!status.can_checkin || submitting" @click="checkin">
          {{ submitting ? '处理中...' : status.can_checkin ? '立即签到' : '暂不可签到' }}
        </button>
      </div>
    </section>

    <section class="panel">
      <div class="section-head">
        <div>
          <p class="kicker">History</p>
          <h2>签到记录</h2>
        </div>
        <span class="inline-chip">共 {{ history.length }} 条</span>
      </div>

      <div class="table-wrap">
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
            <tr v-if="history.length === 0">
              <td colspan="5" class="empty-cell">暂无签到记录</td>
            </tr>
            <tr v-for="item in history" :key="item.id">
              <td>#{{ item.id }}</td>
              <td>{{ item.checkin_date }}</td>
              <td>{{ item.amount }}</td>
              <td>
                <span class="table-tag" :class="recordStatusClass(item.status)">
                  {{ formatRecordStatus(item.status) }}
                </span>
              </td>
              <td>{{ item.created_at }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
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

const statusTitle = computed(() => {
  if (!status.value) return '--'
  if (status.value.blocked) return '风控限制'
  if (status.value.checked_in) return '已完成签到'
  if (status.value.can_checkin) return '等待签到'
  return '不可签到'
})

function formatRecordStatus(rawStatus: string): string {
  const statusMap: Record<string, string> = {
    success: '成功',
    failed: '失败',
    blocked: '封禁',
    pending: '处理中'
  }
  return statusMap[rawStatus] || rawStatus || '--'
}

function recordStatusClass(rawStatus: string): 'success' | 'warn' | 'error' {
  if (rawStatus === 'success') return 'success'
  if (rawStatus === 'failed' || rawStatus === 'blocked') return 'error'
  return 'warn'
}

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
  await auth.logout()
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

<style scoped>
.checkin-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.action-block {
  margin-top: 16px;
  display: grid;
  gap: 10px;
  align-items: start;
}

.empty-cell {
  color: #9ab0c8;
  text-align: center;
  padding: 20px 12px;
}
</style>
