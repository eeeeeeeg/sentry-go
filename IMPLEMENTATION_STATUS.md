# 服务端实现状态

本文档记录当前实现进度，便于按 `SERVICE_DEVELOPMENT_PLAN.md` 的第 11 节继续推进。

## 已完成

- M0 工程骨架：
  - `cmd/api` API 服务入口。
  - `internal/config` 环境变量配置加载。
  - `internal/platform` PostgreSQL、ClickHouse、Redis、NATS 连接封装。
  - `internal/api` HTTP router、请求日志、`/healthz`、`/readyz`、`/metrics`。
- 本地开发环境：
  - `docker-compose.yml` 编排 PostgreSQL、ClickHouse、Redis、NATS JetStream、MinIO、API 和 worker。
  - `Dockerfile` 支持容器内构建 API、Normalize Worker、Event Writer。
  - `Makefile` 提供 `dev`、`api`、`workers`、`fmt`、`test` 等入口。
  - `.env.example` 提供本地默认配置。
- 首批 schema：
  - PostgreSQL：organizations、teams、users、projects、project_keys、api_tokens、issues。
  - ClickHouse：events、event_exceptions、event_breadcrumbs、event_tags、event_users。
- M1 ingestion API 初稿：
  - 启动时创建 / 更新 NATS JetStream `EVENTS` stream。
  - 支持 `events.raw`、`events.normalized`、`events.dlq` subject。
  - `POST /api/{project_id}/envelope` 支持 JSON 上报。
  - 支持 `X-Sentry-Key`、`sentry_key` query、`X-DSN`、`Authorization: DSN ...`、`Authorization: Sentry sentry_key=...`。
  - 从 PostgreSQL 校验 project key、项目状态和 key 状态。
  - 使用 Redis 按 project / key / IP 做固定窗口限流。
  - 将 raw event 元数据和原始 payload 发布到 NATS。
- M2 normalize / event writer 初稿：
  - `worker-normalize` 使用 durable consumer 消费 `events.raw`。
  - 基础解析 envelope 字段，补齐默认 timestamp、level、platform、environment。
  - 递归清洗 authorization、cookie、password、token、secret 等敏感字段。
  - 将 normalized event 发布到 `events.normalized`。
  - 无法解析的 raw event 发布到 `events.dlq`。
- M3 issue grouping / query API 初稿：
  - `worker-grouping` 消费 `events.normalized`，生成 fingerprint。
  - 根据 project + fingerprint 创建或更新 PostgreSQL `issues`。
  - grouped event 写入 `events.grouped`，包含 `issue_id` 和 `fingerprint`。
  - `worker-event-writer` 消费 `events.grouped` 并写入 ClickHouse `sentry.events`。
  - 提供 issue 列表 / 详情 API。
  - 提供 event 列表 / 详情 API。
  - 提供 issue 状态变更 API，并记录 `issue_status_changes`。
  - 提供基础统计 API：趋势、level 分布、top issue、top release。
- M4 alert 初稿：
  - PostgreSQL 增加 `alerts` 和 `alert_deliveries`。
  - `worker-grouping` 在新 issue 和回归 issue 时发布 `alerts.triggered`。
  - `worker-alert` 消费告警事件，匹配 active webhook 规则并发送 HTTP POST。
  - 使用 Redis 根据 alert / issue / event_type 做告警抑制窗口。
  - 提供 webhook 告警规则创建和列表 API。
  - 支持频率阈值告警：同一 issue 在指定窗口内达到 `threshold_count` 后触发。
  - 支持告警规则启停和 alert delivery 查询。
- 后台管理 API 初稿：
  - 提供项目列表、详情、创建、基础配置更新和启停 API。
  - 提供项目 DSN Key 列表、创建、基础配置更新和启停 API。
  - 新建项目默认挂到 `demo` organization，可通过 `organization_id` 或 `organization_slug` 指定组织。
- 后台 UI 部署：
  - Dockerfile 增加 Node 构建阶段，执行 `ui` 的 `npm ci` 和 `npm run build`。
  - API 镜像复制 `ui/dist` 到 `/ui`。
  - API 服务托管 SPA 静态资源，`/api/*`、`/healthz`、`/readyz`、`/metrics` 保持后端接口语义。
- 联调辅助：
  - 新增 `scripts/smoke.ps1`，用于本地端到端 smoke 验证。

## 已验证

- `docker compose config` 可以正常解析 Compose 配置。

## 暂未验证

当前机器未安装 Go，且 Docker daemon 没有运行，因此以下命令尚未执行成功：

```bash
make deps
make fmt
make test
make api
make workers
```

启动 Docker Desktop 或安装 Go 后，需要先执行：

```bash
docker run --rm -v "${PWD}:/app" -w /app golang:1.23-alpine go mod tidy
docker run --rm -v "${PWD}:/app" -w /app golang:1.23-alpine gofmt -w cmd internal pkg
docker run --rm -v "${PWD}:/app" -w /app golang:1.23-alpine go test ./...
```

## 下一步

继续完善后台管理能力和 M4，并准备联调。

- 在 UI 中接入项目管理和 DSN Key 管理页面。
- 增加 alert 规则编辑、删除 / 测试发送 API。
- 增加 DLQ 查询、重放和丢弃 API。
- 优化 user_count 统计，避免同一用户重复累加。
- 增加 grouping worker 和查询 API 的集成测试。
- 增加 email / 飞书 / 企业微信通知 channel。
- 给 Alert Worker 增加失败重试和更细的投递状态。
