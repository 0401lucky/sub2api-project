# Welfare Backend

独立福利站后端（Go + Gin + GORM）。

## 功能
- LinuxDo OAuth 登录
- 按 LinuxDo subject 映射 sub2api 用户
- 每日签到随机发放额度
- 调用 sub2api 管理接口加余额（不改 sub2api 源码）
- 管理后台：签到配置、签到记录、风控封禁

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
