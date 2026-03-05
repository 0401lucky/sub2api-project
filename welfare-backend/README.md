# Welfare Backend

独立福利站后端（Go + Gin + GORM）。

## 功能
- LinuxDo OAuth 登录
- 按 LinuxDo subject 映射 sub2api 用户
- 每日签到随机发放额度
- 调用 sub2api 管理接口加余额（不改 sub2api 源码）
- 管理后台：签到配置、签到记录、风控封禁
- JWT 使用 HttpOnly Cookie 会话（前端不落地 token）

## 启动
```bash
cp .env.example .env
# 填写关键环境变量后
go mod tidy
go run ./cmd/server
```

## 关键接口
- `GET /api/v1/auth/linuxdo/start`
- `GET /api/v1/auth/linuxdo/callback`
- `GET /api/v1/checkin/status`
- `POST /api/v1/checkin/daily`
- `GET /api/v1/checkin/history`
- `GET /api/v1/admin/checkin/config`
- `PUT /api/v1/admin/checkin/config`
- `GET /api/v1/admin/checkin/records`
- `GET /api/v1/admin/risk/blocks`
- `POST /api/v1/admin/risk/blocks`
- `DELETE /api/v1/admin/risk/blocks/:id`

## 安全配置建议
- `WELFARE_CORS_ALLOWED_ORIGINS` 必填，按前端域名精确配置
- `WELFARE_TRUSTED_PROXIES` 只填网关/反代地址，禁止信任公网来源
- 生产环境请开启 `WELFARE_COOKIE_SECURE=true`
- 若前后端不在同一站点（不同主域名），请设置 `WELFARE_COOKIE_SAMESITE=none`，并确保 `WELFARE_COOKIE_SECURE=true`
- `LINUXDO_REDIRECT_URL` 必须配置为后端回调地址：`https://<后端域名>/api/v1/auth/linuxdo/callback`
- `WELFARE_FRONTEND_CALLBACK_URL` 必须配置为前端回调页面：`https://<前端域名>/auth/callback`
