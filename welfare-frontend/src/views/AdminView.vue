<template>
  <section class="card" v-if="!auth.state.profile?.is_admin">
    <h2>管理后台</h2>
    <p class="error">你不是管理员账号。</p>
  </section>

  <template v-else>
    <section class="card">
      <h2>签到配置</h2>
      <div class="form-grid" v-if="config">
        <label>开关 <input type="checkbox" v-model="config.enabled" /></label>
        <label>最小额度 <input type="number" step="0.01" v-model.number="config.reward_min" /></label>
        <label>最大额度 <input type="number" step="0.01" v-model.number="config.reward_max" /></label>
        <label>小数位数 <input type="number" min="0" max="4" v-model.number="config.reward_scale" /></label>
        <label>时区 <input v-model="config.timezone" /></label>
      </div>
      <p v-if="msg" :class="msgType">{{ msg }}</p>
      <button @click="saveConfig">保存配置</button>
    </section>

    <section class="card">
      <h2>风控封禁</h2>
      <div class="form-grid">
        <label>类型
          <select v-model="newBlock.block_type">
            <option value="user">user</option>
            <option value="subject">subject</option>
            <option value="ip">ip</option>
          </select>
        </label>
        <label>值 <input v-model="newBlock.block_value" placeholder="如 123 或 1.2.3.4" /></label>
        <label>原因 <input v-model="newBlock.reason" /></label>
      </div>
      <button @click="createBlock">新增封禁</button>

      <table class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>类型</th>
            <th>值</th>
            <th>原因</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in blocks" :key="item.id">
            <td>{{ item.id }}</td>
            <td>{{ item.block_type }}</td>
            <td>{{ item.block_value }}</td>
            <td>{{ item.reason }}</td>
            <td><button @click="removeBlock(item.id)">删除</button></td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="card">
      <h2>签到记录</h2>
      <table class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>用户ID</th>
            <th>日期</th>
            <th>额度</th>
            <th>状态</th>
            <th>错误</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in records" :key="item.id">
            <td>{{ item.id }}</td>
            <td>{{ item.sub2api_user_id }}</td>
            <td>{{ item.checkin_date }}</td>
            <td>{{ item.amount }}</td>
            <td>{{ item.status }}</td>
            <td>{{ item.last_error }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </template>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { apiRequest } from '../api/client'
import { useAuthStore } from '../store/auth'

interface ConfigItem {
  enabled: boolean
  reward_min: number
  reward_max: number
  reward_scale: number
  timezone: string
}

interface CheckinRecord {
  id: number
  sub2api_user_id: number
  checkin_date: string
  amount: number
  status: string
  last_error: string
}

interface RiskBlock {
  id: number
  block_type: string
  block_value: string
  reason: string
}

const auth = useAuthStore()
const config = ref<ConfigItem | null>(null)
const records = ref<CheckinRecord[]>([])
const blocks = ref<RiskBlock[]>([])
const msg = ref('')
const msgType = ref('notice')
const newBlock = reactive({
  block_type: 'ip',
  block_value: '',
  reason: ''
})

async function loadAll(): Promise<void> {
  config.value = await apiRequest<ConfigItem>('/admin/checkin/config')
  const recordResp = await apiRequest<{ items: CheckinRecord[] }>('/admin/checkin/records?page=1&page_size=50')
  records.value = recordResp.items
  blocks.value = await apiRequest<RiskBlock[]>('/admin/risk/blocks')
}

async function saveConfig(): Promise<void> {
  if (!config.value) return
  try {
    await apiRequest('/admin/checkin/config', {
      method: 'PUT',
      body: JSON.stringify(config.value)
    })
    msg.value = '配置已保存'
    msgType.value = 'success'
  } catch (e) {
    msg.value = (e as Error).message
    msgType.value = 'error'
  }
}

async function createBlock(): Promise<void> {
  try {
    await apiRequest('/admin/risk/blocks', {
      method: 'POST',
      body: JSON.stringify(newBlock)
    })
    newBlock.block_value = ''
    newBlock.reason = ''
    await loadAll()
  } catch (e) {
    msg.value = (e as Error).message
    msgType.value = 'error'
  }
}

async function removeBlock(id: number): Promise<void> {
  await apiRequest(`/admin/risk/blocks/${id}`, { method: 'DELETE' })
  await loadAll()
}

onMounted(async () => {
  if (!auth.state.profile?.is_admin) return
  await loadAll()
})
</script>
