# 类 Sentry 错误收集服务端开发规划

本文档用于指导类 Sentry 错误收集系统的服务端开发。服务端目标不是完整复刻 Sentry，而是参考其公开架构思想，先实现稳定的错误采集、分组、查询和告警能力，再逐步扩展性能监控、Source Map、Tracing 等高级能力。

## 1. 总体目标

构建一个支持多项目、多环境、多版本的错误收集后台，能够接收来自 Web、React、Vue、Tauri、鸿蒙、混合移动端等 SDK 的错误事件，并完成以下核心流程：

```text
SDK
  -> Ingestion API
  -> Queue / Stream
  -> Normalize Worker
  -> Grouping Worker
  -> Storage
  -> Query API / Dashboard
  -> Alert Worker
```

第一阶段重点是错误监控，不急于实现完整 APM、Session Replay、Profiling 等复杂能力。

## 2. 推荐技术选型

### 2.1 首选方案

```text
Backend API: Go + chi
Worker: Go
Queue / Stream: NATS JetStream
Cache / Rate Limit: Redis
Metadata DB: PostgreSQL
Event DB: ClickHouse
Object Storage: MinIO / S3
Frontend Dashboard: React / Next.js
Deploy: Docker Compose -> Kubernetes
```


## 3. 服务端模块划分

```text
cmd/
  api/
  worker-normalize/
  worker-grouping/
  worker-alert/

internal/
  config/
  ingest/
  auth/
  quota/
  normalize/
  grouping/
  storage/
  alert/
  project/
  issue/
  sourcemap/
  telemetry/

pkg/
  event/
  envelope/
  dsn/
```

### 3.1 API 服务

职责：

- 接收 SDK 上报事件。
- 校验 DSN、公钥、项目状态。
- 做基础限流和大小限制。
- 将原始事件写入 NATS JetStream。
- 提供项目、Issue、事件查询 API。
- 提供后台管理 API。

### 3.2 Normalize Worker

职责：

- 解析 envelope。
- 标准化事件字段。
- 解析 stacktrace。
- 清洗 PII 敏感信息。
- 补充项目、环境、版本、SDK 信息。
- 写入 normalized event stream。

### 3.3 Grouping Worker

职责：

- 基于 exception type、stacktrace frame、message、platform 生成 fingerprint。
- 判断事件归属到已有 issue，或者创建新 issue。
- 更新 issue 计数、首次出现时间、最后出现时间。
- 将事件明细写入 ClickHouse。
- 将 issue 元数据写入 PostgreSQL。

### 3.4 Alert Worker

职责：

- 新 issue 告警。
- 回归 issue 告警。
- 频率阈值告警。
- Webhook / Email / 飞书 / 企业微信等通知。

## 4. 存储设计

### 4.1 PostgreSQL

存储业务元数据和低频查询数据：

```text
organizations
projects
project_keys
users
teams
issues
issue_status_changes
alerts
releases
source_maps
api_tokens
```

Issue 表核心字段：

```text
id
organization_id
project_id
fingerprint
title
culprit
level
status
first_seen
last_seen
event_count
user_count
release
environment
created_at
updated_at
```

### 4.2 ClickHouse

存储高吞吐事件明细和聚合查询数据：

```text
events
event_exceptions
event_breadcrumbs
event_tags
event_users
```

Events 表核心字段：

```text
event_id
project_id
issue_id
timestamp
received_at
platform
runtime_name
runtime_version
sdk_name
sdk_version
level
message
exception_type
exception_value
release
environment
user_id
tags
contexts
raw_event
```

### 4.3 Redis

用途：

```text
DSN 校验缓存
项目配置缓存
限流计数
短期去重
告警抑制窗口
worker 临时状态
```

### 4.4 MinIO / S3

用途：

```text
Source Map 文件
附件
大体积原始 payload 归档
导出文件
```

## 5. Ingestion API 设计

第一阶段建议提供一个统一上报入口：

```text
POST /api/{project_id}/envelope
```

请求 Header：

```text
Content-Type: application/json
X-SDK-Name
X-SDK-Version
X-Client-Report
```

认证方式：

```text
DSN public key
project_id
```

服务端处理流程：

```text
1. 解析 project_id 和 DSN。
2. 校验项目是否存在、是否启用。
3. 校验 payload 大小。
4. 执行 project / key / IP 维度限流。
5. 写入 NATS raw event stream。
6. 返回 202 Accepted。
```

API 层不要直接写 ClickHouse 和 PostgreSQL，避免存储抖动影响 SDK 上报链路。

## 6. Envelope 协议

服务端内部使用统一 envelope 格式，不要求第一版完全兼容 Sentry 协议。

示例：

```json
{
  "event_id": "uuid",
  "timestamp": "2026-05-14T10:00:00Z",
  "platform": "javascript",
  "runtime": {
    "name": "browser",
    "version": "chrome-124"
  },
  "sdk": {
    "name": "your-sdk-js",
    "version": "0.1.0"
  },
  "level": "error",
  "message": "Cannot read properties of undefined",
  "exception": {
    "type": "TypeError",
    "value": "Cannot read properties of undefined",
    "stacktrace": []
  },
  "release": "1.0.0",
  "environment": "production",
  "tags": {},
  "user": {},
  "breadcrumbs": []
}
```

## 7. 错误分组策略

第一版分组算法应保持简单、可解释：

```text
fingerprint = hash(project_id + platform + exception_type + normalized_stack_top_frames)
```

优先级：

```text
1. 用户自定义 fingerprint。
2. exception type + stacktrace 顶部业务 frame。
3. exception type + message 模板。
4. message 文本。
```

需要避免的问题：

- message 中包含 ID、URL、时间戳导致分组爆炸。
- 第三方库 frame 导致不同业务错误归为一类。
- 压缩 JS 未接入 Source Map 时分组质量较差。

## 8. 安全与限流

必须实现：

```text
payload size limit
project quota
DSN key status
IP rate limit
event sample rate
PII scrubber
blocked IP / user agent
```

默认脱敏字段：

```text
authorization
cookie
set-cookie
password
token
secret
access_token
refresh_token
```

## 9. 分阶段实现

### 阶段 1：最小可用错误采集

目标：

```text
SDK 可以上报错误
服务端可以接收事件
事件可以入队
worker 可以消费并写入数据库
后台可以查看事件列表
```

任务：

- 初始化 Go 项目结构。
- 实现配置加载。
- 实现 PostgreSQL migration。
- 实现 ClickHouse migration。
- 实现 Redis 连接。
- 实现 NATS JetStream。
- 实现 `/api/{project_id}/envelope`。
- 实现 raw event consumer。
- 实现基础事件标准化。
- 实现事件写入 ClickHouse。
- 实现项目和 DSN key 管理。

### 阶段 2：Issue 分组与查询

目标：

```text
同类错误可以归并为 issue
可以按项目、环境、版本、时间范围查询
```

任务：

- 实现 fingerprint 生成。
- 实现 issue 创建和更新。
- 实现事件到 issue 的关联。
- 实现 issue 列表 API。
- 实现 issue 详情 API。
- 实现事件详情 API。
- 实现基础统计接口。

### 阶段 3：Dashboard 和告警

目标：

```text
可以通过 Web UI 查看错误趋势，并收到关键错误通知
```

任务：

- 实现 React / Next.js dashboard。
- 实现项目概览。
- 实现 issue 列表和详情页。
- 实现事件详情页。
- 实现 webhook 告警。
- 实现 email 告警。
- 实现告警抑制窗口。

### 阶段 4：Source Map

目标：

```text
Web / React / Vue 错误可以还原源码位置
```

任务：

- 实现 release 管理。
- 实现 Source Map 上传 API。
- 存储 Source Map 到 MinIO / S3。
- Normalize Worker 中解析 minified stacktrace。
- 支持 source context 展示。

### 阶段 5：高级能力

目标：

```text
逐步扩展为更完整的可观测性平台
```

候选能力：

- Performance transaction。
- Distributed tracing。
- Session replay。
- Profiling。
- Dynamic sampling。
- 多租户计费和配额。
- 多区域部署。

## 10. 开发原则

- API 层只做轻量校验和入队。
- 所有重逻辑放到 worker。
- 事件明细写 ClickHouse，业务元数据写 PostgreSQL。
- 所有上报接口必须可限流、可降级。
- SDK 上报失败不能影响用户业务。
- 第一版优先保证链路稳定，不追求复杂功能。

## 11. 服务端开发工作安排

本节按“先打通链路、再补齐查询、最后增强稳定性”的顺序组织服务端开发。默认按 6 周完成后端 MVP 规划，实际排期可根据团队人数、SDK 进度和前端联调节奏调整。

### 11.1 里程碑拆分

```text
M0：工程骨架和本地环境可运行
M1：SDK 事件可以接收并可靠入队
M2：worker 可以消费、标准化并落库
M3：Issue 分组、事件查询和基础统计可用
M4：告警链路、限流配额和运维可观测性可用
M5：Source Map 和高级能力进入独立迭代
```

### 11.2 第 1 周：工程初始化与基础设施

目标：

```text
服务端工程可以本地启动
PostgreSQL / ClickHouse / Redis / NATS JetStream 依赖可一键拉起
基础配置、日志、健康检查和 migration 框架完成
```

任务：

- 初始化 Go module、cmd 和 internal 目录结构。
- 建立统一配置加载，支持本地 `.env`、环境变量和默认值。
- 建立结构化日志、中间件、请求 ID、panic recover。
- 编写 Docker Compose，包含 PostgreSQL、ClickHouse、Redis、NATS、MinIO。
- 实现 PostgreSQL migration 基础表：organizations、projects、project_keys、issues、api_tokens。
- 实现 ClickHouse migration 基础表：events。
- 封装 PostgreSQL、ClickHouse、Redis、NATS 连接和健康检查。
- 提供 `/healthz`、`/readyz`、`/metrics` 基础接口。

交付物：

- `make dev` 或等价脚本可以启动完整本地依赖。
- API 服务启动后可以完成健康检查。
- migration 可以重复执行且具备幂等保护。

验收标准：

- 新机器按 README 操作可以在 15 分钟内启动本地开发环境。
- 依赖未就绪时 `/readyz` 能正确返回非就绪状态。
- 基础单元测试和 migration 测试通过。

### 11.3 第 2 周：Ingestion API 与入队链路

目标：

```text
SDK 可以调用统一上报接口
API 层完成轻量校验、限流和 raw event 入队
上报链路不直接依赖 ClickHouse / PostgreSQL 写入成功
```

任务：

- 实现 DSN 解析和 project key 校验。
- 实现 `POST /api/{project_id}/envelope`。
- 实现 payload size limit、Content-Type 校验和基础 schema 校验。
- 实现 project / key / IP 维度限流，限流状态写入 Redis。
- 定义 raw event stream subject、consumer name、重试策略和 dead letter subject。
- 将原始 envelope 写入 NATS JetStream。
- 实现上报接口 202、400、401、403、413、429 的稳定响应格式。
- 补充 ingestion API 的集成测试和压测脚本。

交付物：

- 使用 curl 或 JS SDK demo 可以完成错误事件上报。
- NATS 中可以看到 raw event 消息。
- API 层在 ClickHouse 不可用时仍可接收并入队。

验收标准：

- 单实例 API 在本地压测下能稳定处理目标吞吐，例如 500-1000 events/s。
- 无效 DSN、禁用 key、超大 payload、超限流请求都有明确响应。
- 入队失败时 API 返回可观测错误，并记录结构化日志。

### 11.4 第 3 周：Normalize Worker 与事件落库

目标：

```text
raw event 可以被 worker 消费
事件完成标准化、PII 清洗、stacktrace 解析并写入 ClickHouse
```

任务：

- 实现 raw event consumer，支持 ack、nak、重试和 dead letter。
- 定义内部 envelope 结构体和版本字段。
- 实现 event_id、timestamp、level、platform、runtime、sdk、release、environment 标准化。
- 实现 exception、stacktrace、breadcrumbs、tags、user、contexts 基础解析。
- 实现 PII scrubber，覆盖 authorization、cookie、password、token 等默认字段。
- 实现 normalized event stream，供后续 grouping worker 消费。
- 实现事件明细写入 ClickHouse，保留 raw_event 便于排查。
- 增加 worker 消费延迟、失败次数、DLQ 数量等 metrics。

交付物：

- 上报事件可以在 ClickHouse `events` 表查询到。
- 脏数据不会阻塞整个 consumer。
- PII 字段在落库前已被清洗。

验收标准：

- 同一批测试 envelope 中，合法事件全部落库，非法事件进入 DLQ 或被明确丢弃。
- worker 重启后不会重复消费已 ack 消息。
- ClickHouse 短暂不可用时消息不会静默丢失。

### 11.5 第 4 周：Issue 分组与查询 API

目标：

```text
同类错误可以归并为 issue
后台可以按项目、环境、版本、时间范围查询 issue 和事件
```

任务：

- 实现 fingerprint 生成，优先支持用户自定义 fingerprint。
- 实现 message 归一化，去除 URL、UUID、数字 ID、时间戳等高噪声片段。
- 实现业务 frame 识别和顶部 stack frame 提取。
- 实现 issue upsert，更新 first_seen、last_seen、event_count、user_count。
- 建立事件与 issue 的关联，写入 ClickHouse issue_id。
- 实现 issue 列表 API，支持项目、环境、版本、状态、时间范围过滤。
- 实现 issue 详情 API，返回 issue 元数据、最近事件、聚合统计。
- 实现事件列表和事件详情 API。
- 实现基础统计接口：错误趋势、level 分布、top issue、top release。

交付物：

- 多个相同错误会上报到同一个 issue。
- 后台可以通过 API 拉取 issue 列表、事件详情和趋势数据。
- issue 状态支持 unresolved、resolved、ignored。

验收标准：

- 分组用例覆盖 stacktrace、message-only、自定义 fingerprint 三类事件。
- API 查询有分页、排序和稳定响应结构。
- 查询接口具备合理索引和超时控制，不允许无边界全表扫描。

### 11.6 第 5 周：告警、配额与稳定性治理

目标：

```text
关键错误可以触发通知
系统具备基础限流、配额、降级和运维可观测性
```

任务：

- 实现 Alert Worker，消费 issue 变更或事件聚合信号。
- 实现新 issue、回归 issue、频率阈值告警。
- 实现 webhook 告警，email / 飞书 / 企业微信作为可插拔 channel。
- 实现告警抑制窗口和重复告警合并。
- 实现项目级 quota、sample rate、blocked IP / user agent。
- 增加 API、worker、queue、DB 的 Prometheus metrics。
- 增加关键链路 trace id 和结构化审计日志。
- 补充 DLQ 重放工具和手动修复脚本。

交付物：

- 创建新 issue 时可以收到 webhook 通知。
- 高频错误不会无限刷屏。
- 超出配额的项目会被限流或采样。

验收标准：

- 告警规则变更后无需重启服务即可生效，或具备明确缓存刷新机制。
- 依赖异常时系统有可观测指标和明确日志。
- DLQ 消息可以按项目、时间范围重放。

### 11.7 第 6 周：联调、验收与发布准备

目标：

```text
完成 SDK、前端 dashboard、服务端和部署脚本联调
形成可发布的后端 MVP
```

任务：

- 与 Web / React / Vue SDK 联调 envelope 字段和错误上报。
- 与 dashboard 联调项目、issue、事件、统计 API。
- 补齐 OpenAPI 文档和接口示例。
- 完成 Docker Compose 部署文档和生产环境配置模板。
- 执行 ingestion API 压测、worker 消费压测、查询 API 压测。
- 执行异常演练：NATS 不可用、ClickHouse 不可用、Redis 不可用、worker 重启。
- 梳理数据保留策略、ClickHouse 分区策略和清理任务。
- 准备 MVP 发布检查清单。

交付物：

- 后端 MVP 可以在测试环境持续运行。
- SDK demo 到 dashboard 展示的完整链路可演示。
- 核心接口、队列、数据库和告警链路都有基础测试覆盖。

验收标准：

- 端到端链路成功率、延迟和吞吐达到第一阶段目标。
- 关键异常场景不会造成消息静默丢失。
- 发布文档包含环境变量、依赖服务、migration、回滚和排障步骤。

### 11.8 并行工作与依赖关系

推荐并行拆分：

- API 开发：负责 ingestion、认证、限流、查询 API。
- Worker 开发：负责 normalize、grouping、alert、DLQ。
- 存储与运维：负责 migration、ClickHouse schema、NATS、Redis、metrics、部署脚本。
- SDK / Dashboard 联调：在第 2 周后即可基于 mock API 或测试环境提前接入。

关键依赖：

- Ingestion API 依赖 project、project_keys、NATS stream 先完成。
- Normalize Worker 依赖 raw event stream 和 envelope 结构定义。
- Grouping Worker 依赖 normalized event stream、issues 表和 ClickHouse events 表。
- Alert Worker 依赖 issue 状态变更、告警规则和 Redis 抑制窗口。
- Dashboard 查询依赖 issue API、event API 和统计 API。

### 11.9 第一阶段暂缓事项

以下内容不进入后端 MVP 主路径，避免影响错误采集链路上线：

- 完整兼容 Sentry envelope 协议。
- 完整 APM transaction、trace span 和 profiling。
- Session replay 存储与回放。
- 多租户计费、账单和复杂配额。
- 多区域主动主动部署。
- 复杂 Source Map 还原和源码上下文高亮，可作为阶段 4 独立迭代。

### 11.10 MVP 发布检查清单

```text
工程可以一键启动
migration 可重复执行
SDK 上报返回 202
raw event 可以入队
worker 可以稳定消费
事件可以写入 ClickHouse
issue 可以创建和更新
查询 API 支持分页、过滤和超时
限流、配额、PII 清洗生效
告警可以触发且可抑制
metrics、日志、DLQ 可用于排障
部署文档和回滚步骤完整
```
