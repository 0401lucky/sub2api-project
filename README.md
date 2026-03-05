# Welfare Station for sub2api

独立福利站实现（不修改 sub2api 源码），包含：
- LinuxDo OAuth 登录
- sub2api 用户识别（synthetic email 映射）
- 每日签到随机发额度
- 后台配置与风控封禁

## 目录
- `welfare-backend`: Go API 服务
- `welfare-frontend`: Vue3 前端

## 快速启动
1. 启动后端
```bash
cd welfare-backend
cp .env.example .env
# 填写 LinuxDo 与 sub2api 管理密钥
go run ./cmd/server
```

2. 启动前端
```bash
cd welfare-frontend
cp .env.example .env
npm install
npm run dev
```

## 注意
- 生产环境请使用 Postgres，并把 `WELFARE_COOKIE_SECURE=true`
- `SUB2API_ADMIN_API_KEY` 仅保留在后端环境变量，不要暴露到前端
- 前端已改为 Cookie 会话，不再在浏览器持久化 access token
