<template>
  <section class="panel login-hero">
    <div class="section-head">
      <div>
        <p class="kicker">Secure Sign-In</p>
        <h2>把签到流程变成一键操作</h2>
      </div>
      <span class="inline-chip">OAuth 2.0</span>
    </div>

    <p class="lead">
      使用 LinuxDo 完成授权后，系统会自动映射 sub2api 账号，
      登录成功即回到你刚才访问的页面。
    </p>

    <ul class="feature-list">
      <li>自动识别账号身份，无需手动绑定</li>
      <li>登录态过期自动回收，减少异常操作</li>
      <li>支持管理员与普通用户统一入口</li>
    </ul>

    <button class="btn-login" @click="login">使用 LinuxDo 登录</button>
    <p class="text-muted footnote">授权仅用于签到业务所需的身份校验。</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { linuxdoLoginURL } from '../api/client'
import { sanitizeAuthRedirect } from '../utils/authRedirect'

const route = useRoute()
const redirectPath = computed(() => {
  const raw = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
  return sanitizeAuthRedirect(raw)
})

function login(): void {
  window.location.href = linuxdoLoginURL(redirectPath.value)
}
</script>

<style scoped>
.login-hero {
  max-width: 760px;
  margin: 0 auto;
  padding: clamp(24px, 4vw, 38px);
}

.login-hero::after {
  content: '';
  position: absolute;
  right: -22%;
  bottom: -34%;
  width: 320px;
  height: 320px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(71, 214, 195, 0.28), rgba(71, 214, 195, 0));
  pointer-events: none;
}

.lead {
  margin: 0;
  color: #c4d8ee;
  line-height: 1.75;
  max-width: 58ch;
}

.feature-list {
  margin: 18px 0 24px;
  padding-left: 18px;
  display: grid;
  gap: 9px;
  color: #dce9f8;
}

.feature-list li::marker {
  color: #6be0d3;
}

.btn-login {
  min-width: 220px;
  padding: 12px 20px;
  font-size: 15px;
}

.footnote {
  margin: 12px 0 0;
  font-size: 13px;
}

@media (max-width: 640px) {
  .login-hero {
    padding: 20px;
  }

  .btn-login {
    width: 100%;
  }
}
</style>
