<template>
  <section class="card">
    <h2>登录</h2>
    <p>使用 LinuxDo 登录后，系统会自动匹配 sub2api 用户身份。</p>
    <button @click="login">使用 LinuxDo 登录</button>
  </section>
</template>

<script setup lang="ts">
import { linuxdoLoginURL } from '../api/client'

function login(): void {
  window.location.href = linuxdoLoginURL('/')
}
</script>
