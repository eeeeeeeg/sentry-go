# Sentry Lite

类 Sentry 错误收集服务端，当前处于 M0 工程初始化阶段。

## 本地启动

当前机器不要求安装 Go，本地开发可以通过 Docker 完成。

```bash
make dev
make api
make workers
```

API 默认监听：

```text
http://localhost:8080
```

Docker 镜像会同时打包后台 UI。启动 API 容器后，可以直接访问：

```text
http://localhost:8080
```

如果需要从外部挂载或替换前端静态文件，可以设置：

```text
UI_DIST_DIR=/path/to/dist
```

健康检查：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

## 本地依赖

Docker Compose 会启动：

```text
PostgreSQL: localhost:5432
ClickHouse HTTP: localhost:8123
ClickHouse Native: localhost:9000
Redis: localhost:6379
NATS: localhost:4222
MinIO Console: http://localhost:9001
```

## 开发命令

```bash
make deps
make fmt
make test
make down
```

## 一键部署

Windows / Docker Desktop：

```powershell
.\scripts\deploy.ps1
```

常用参数：

```powershell
.\scripts\deploy.ps1 -SkipBuild
.\scripts\deploy.ps1 -UseBuildCache
.\scripts\deploy.ps1 -RecreateVolumes
.\scripts\deploy.ps1 -SkipSmoke
.\scripts\deploy.ps1 -NoForceRecreate
```

脚本会执行 Docker 镜像构建、强制重建 API / worker 容器、`/readyz` 等待和 smoke 验证。默认使用无缓存构建，确保后台 UI 静态资源随源码更新；确认不需要重新打包 UI 时可以加 `-UseBuildCache` 加快构建。

## 后台 UI

```bash
cd ui
npm install
npm run dev
```

访问：

```text
http://localhost:5173
```

UI 使用 React Router 分层菜单，Axios 请求统一封装在 `ui/src/http.ts`。

## 当前状态

已完成：

- Go API 工程骨架。
- Docker Compose 本地依赖。
- PostgreSQL / ClickHouse 首批 schema。
- `/healthz`、`/readyz`、`/metrics` 基础接口。
- NATS JetStream `EVENTS` stream 初始化。
- `POST /api/{project_id}/envelope` 初版 ingestion API。
- project key 校验、Redis 固定窗口限流、raw event 入队。
- `worker-normalize` 消费 `events.raw`，执行基础标准化和 PII 清洗。
- `worker-grouping` 消费 `events.normalized`，生成 fingerprint 并创建 / 更新 issue。
- `worker-event-writer` 消费 `events.grouped`，写入 ClickHouse `sentry.events`。
- `worker-transaction`、`worker-session`、`worker-attachment`、`worker-profile`、`worker-replay`、`worker-outcome` 分别消费 Sentry envelope 的对应 item 类型。
- `worker-alert` 消费 `alerts.triggered`，按 webhook 规则发送告警并使用 Redis 抑制重复通知。
- 提供项目管理和 DSN Key 管理 API，支持后台创建项目、启停项目、生成 / 启停上报 Key。
- Docker API 镜像内置构建后的后台 UI，并由 API 服务托管 SPA 静态资源。

## 上报示例

本地初始化数据包含 demo 项目：

```text
sentry_project_id: 1
public_key: 0123456789abcdef0123456789abcdef
dsn: http://0123456789abcdef0123456789abcdef@localhost:8080/1
```

示例请求：

```bash
curl -i http://localhost:8080/api/1/envelope/ \
  -H "Content-Type: application/json" \
  -H "X-Sentry-Key: 0123456789abcdef0123456789abcdef" \
  -H "X-SDK-Name: your-sdk-js" \
  -H "X-SDK-Version: 0.1.0" \
  -d '{
    "event_id": "018f7b64-0000-7000-9000-000000000001",
    "timestamp": "2026-05-14T10:00:00Z",
    "platform": "javascript",
    "runtime": {"name": "browser", "version": "chrome-124"},
    "sdk": {"name": "your-sdk-js", "version": "0.1.0"},
    "level": "error",
    "message": "Cannot read properties of undefined",
    "exception": {
      "type": "TypeError",
      "value": "Cannot read properties of undefined",
      "stacktrace": []
    },
    "release": "1.0.0",
    "environment": "production"
  }'
```

下一步：

- 增加更多分组测试用例。
- 补齐端到端集成测试。
- 增加 email / 飞书 / 企业微信通知 channel。

## 查询示例

```bash
curl "http://localhost:8080/api/projects"
curl -X POST "http://localhost:8080/api/projects" \
  -H "Content-Type: application/json" \
  -d '{"organization_slug":"demo","slug":"mobile","name":"Mobile App","platform":"javascript","sample_rate":1}'
curl "http://localhost:8080/api/projects/1"
curl -X PATCH "http://localhost:8080/api/projects/1" \
  -H "Content-Type: application/json" \
  -d '{"name":"Web Frontend","platform":"javascript","sample_rate":0.8}'
curl -X PATCH "http://localhost:8080/api/projects/1/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'
curl "http://localhost:8080/api/projects/1/keys"
curl -X POST "http://localhost:8080/api/projects/1/keys" \
  -H "Content-Type: application/json" \
  -d '{"name":"Browser SDK","rate_limit_per_minute":6000}'
curl -X PATCH "http://localhost:8080/api/project-keys/{key_id}" \
  -H "Content-Type: application/json" \
  -d '{"name":"Browser SDK Production","rate_limit_per_minute":12000}'
curl -X PATCH "http://localhost:8080/api/project-keys/{key_id}/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'
curl "http://localhost:8080/api/projects/1/issues?status=unresolved"
curl "http://localhost:8080/api/projects/1/events?limit=20"
curl "http://localhost:8080/api/issues/{issue_id}"
curl "http://localhost:8080/api/events/{event_id}"
curl "http://localhost:8080/api/projects/1/stats/trend"
curl "http://localhost:8080/api/projects/1/stats/levels"
curl "http://localhost:8080/api/projects/1/stats/top-issues"
curl "http://localhost:8080/api/projects/1/stats/top-releases"
curl "http://localhost:8080/api/projects/1/alerts"
curl -X POST "http://localhost:8080/api/projects/1/alerts/webhook" \
  -H "Content-Type: application/json" \
  -d '{"name":"New issue webhook","event_type":"new_issue","webhook_url":"https://example.com/webhook","min_level":"error","cooldown_seconds":300}'
curl -X POST "http://localhost:8080/api/projects/1/alerts/webhook" \
  -H "Content-Type: application/json" \
  -d '{"name":"Frequency webhook","event_type":"frequency","webhook_url":"https://example.com/webhook","min_level":"error","threshold_count":10,"window_seconds":300,"cooldown_seconds":900}'
curl "http://localhost:8080/api/projects/1/alert-deliveries?limit=20"
curl -X PATCH "http://localhost:8080/api/alerts/{alert_id}/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'
curl -X PATCH "http://localhost:8080/api/issues/{issue_id}/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"resolved","reason":"fixed in release 1.0.1"}'
```

## Smoke 验证

Docker 服务启动后可以运行：

```powershell
.\scripts\smoke.ps1
```

如果要验证 webhook 告警投递：

```powershell
.\scripts\smoke.ps1 -WebhookUrl "https://example.com/webhook"
```
