<template>
  <div class="shell">
    <header class="topbar">
      <h1>签到福利站</h1>
      <nav>
        <RouterLink to="/">首页</RouterLink>
        <RouterLink v-if="isAdmin" to="/admin">管理后台</RouterLink>
        <RouterLink to="/login">登录</RouterLink>
      </nav>
    </header>
    <main>
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import { useAuthStore } from './store/auth'

const auth = useAuthStore()
const isAdmin = computed(() => Boolean(auth.state.profile?.is_admin))
</script>

<style>
:root {
  --bg: #f4f7fb;
  --card: #ffffff;
  --text: #17202a;
  --accent: #0e7490;
  --warn: #b45309;
  --danger: #b91c1c;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: radial-gradient(circle at 20% 10%, #d6effa 0%, #f4f7fb 42%, #ebf3ff 100%);
  color: var(--text);
  font-family: "Microsoft YaHei", "PingFang SC", sans-serif;
}
.shell { min-height: 100vh; }
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 22px;
  background: rgba(255,255,255,.85);
  border-bottom: 1px solid #dbe6f0;
  backdrop-filter: blur(8px);
}
.topbar h1 { margin: 0; font-size: 20px; }
.topbar nav { display: flex; gap: 14px; }
.topbar a { color: var(--accent); text-decoration: none; font-weight: 600; }
.topbar a.router-link-exact-active { text-decoration: underline; }
main { max-width: 980px; margin: 24px auto; padding: 0 16px; }
.card {
  background: var(--card);
  border: 1px solid #d6e2ef;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 16px;
}
button {
  background: var(--accent);
  border: none;
  color: white;
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
}
button:disabled {
  opacity: .55;
  cursor: not-allowed;
}
input, select {
  border: 1px solid #c7d4e4;
  border-radius: 8px;
  padding: 8px;
}
.table {
  width: 100%;
  border-collapse: collapse;
}
.table th, .table td {
  border-bottom: 1px solid #ebf0f7;
  padding: 8px;
  text-align: left;
}
.notice { color: var(--warn); }
.error { color: var(--danger); }
.success { color: #166534; }
.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
}
</style>
