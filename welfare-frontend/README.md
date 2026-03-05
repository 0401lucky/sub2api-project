# Welfare Frontend

独立福利站前端（Vue3 + Vite）。

## 功能页面
- `/login` LinuxDo 登录入口
- `/auth/callback` OAuth 回调
- `/` 用户签到页
- `/admin` 管理后台（管理员可见）
- 使用 HttpOnly Cookie 会话，不在前端持久化 token

## 启动
```bash
cp .env.example .env
npm install
npm run dev
```
