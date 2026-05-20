# 类 Sentry 错误收集 SDK 开发规划

本文档用于指导多技术栈、多运行平台的错误收集 SDK 开发。SDK 目标是支持 Web、React、Vue、Tauri、鸿蒙、混合移动端等场景，并通过统一协议向服务端上报错误事件。

## 1. 总体目标

SDK 采用 `Core + Platform Adapter + Framework Integration` 架构，避免每个平台重复实现完整 SDK。

```text
sdk-core
  -> event model
  -> scope/context
  -> breadcrumb
  -> stacktrace normalize
  -> transport interface
  -> sampling
  -> retry/buffer
  -> beforeSend
  -> PII scrubber

platform adapters
  -> browser
  -> tauri
  -> harmony
  -> hybrid mobile

framework integrations
  -> react
  -> vue
```

核心原则：

- 协议、事件格式、发送队列、采样、脱敏、重试逻辑放在 core。
- 平台层只负责捕获错误、补充上下文、提供 transport。
- 框架层只负责接入 React / Vue 等框架生命周期和错误钩子。

## 2. 包结构

TypeScript 侧建议采用 monorepo：

```text
packages/
  core/
  browser/
  react/
  vue/
  tauri/
  harmony/
  shared/
```

后续可扩展：

```text
packages/
  react-native/
  flutter/
  android/
  ios/
```

第一阶段优先实现：

```text
@your/sdk-core
@your/sdk-browser
@your/sdk-react
@your/sdk-vue
```

第二阶段实现：

```text
@your/sdk-tauri
@your/sdk-harmony
```

## 3. SDK Core 设计

### 3.1 Core 职责

```text
初始化配置
事件模型
事件 ID 生成
Scope 管理
用户信息
Tags
Extra
Breadcrumbs
异常标准化
Stacktrace 标准化
事件采样
事件脱敏
beforeSend
发送队列
失败重试
离线缓存接口
Transport 抽象
```

### 3.2 初始化配置

示例：

```ts
import { init } from "@your/sdk-browser";

init({
  dsn: "https://public-key@host/project-id",
  release: "1.0.0",
  environment: "production",
  sampleRate: 1,
  maxBreadcrumbs: 50,
  beforeSend(event) {
    delete event.request?.headers?.authorization;
    return event;
  },
});
```

配置字段：

```ts
type SDKOptions = {
  dsn: string;
  release?: string;
  environment?: string;
  sampleRate?: number;
  maxBreadcrumbs?: number;
  enabled?: boolean;
  debug?: boolean;
  beforeSend?: (event: Event) => Event | null | Promise<Event | null>;
  transport?: Transport;
};
```

### 3.3 核心 API

```ts
init(options)
captureException(error, context?)
captureMessage(message, level?, context?)
setUser(user)
setTag(key, value)
setTags(tags)
setExtra(key, value)
addBreadcrumb(breadcrumb)
withScope(callback)
flush(timeout?)
close(timeout?)
```

## 4. 统一事件协议

SDK 统一生成 envelope，并发送到服务端：

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

## 5. Browser SDK

### 5.1 职责

```text
window.onerror
window.onunhandledrejection
console breadcrumb
fetch breadcrumb
XMLHttpRequest breadcrumb
history route breadcrumb
browser runtime context
device / screen context
```

### 5.2 捕获范围

必须支持：

- JavaScript runtime error。
- Promise unhandled rejection。
- 手动 captureException。
- 手动 captureMessage。

可选支持：

- Fetch / XHR 请求记录。
- Console 记录。
- 页面路由变更记录。

### 5.3 注意事项

- 不要默认采集完整 request body。
- 不要默认采集 cookie、authorization。
- fetch / xhr breadcrumb 需要避免记录敏感 header。
- SDK 自己发送事件产生的请求不能再次被记录为 breadcrumb。

## 6. React SDK

### 6.1 职责

```text
ErrorBoundary
React component stack
React version context
React Router breadcrumb
```

### 6.2 示例

```tsx
import { ErrorBoundary } from "@your/sdk-react";

<ErrorBoundary fallback={<ErrorPage />}>
  <App />
</ErrorBoundary>
```

### 6.3 第一阶段任务

- 实现 ErrorBoundary。
- 捕获 componentDidCatch。
- 将 componentStack 写入 event.extra 或 exception mechanism。
- 提供 withErrorBoundary。

## 7. Vue SDK

### 7.1 职责

```text
app.config.errorHandler
app.config.warnHandler
Vue Router breadcrumb
Vue version context
```

### 7.2 示例

```ts
import { createApp } from "vue";
import { init, vuePlugin } from "@your/sdk-vue";

init({ dsn: "https://public-key@host/project-id" });

createApp(App).use(vuePlugin).mount("#app");
```

### 7.3 第一阶段任务

- 实现 Vue plugin。
- 接入 errorHandler。
- 可选接入 warnHandler。
- 支持 Vue Router afterEach breadcrumb。

## 8. Tauri SDK

Tauri 需要分为前端 JS 层和 Rust 原生层。

```text
sdk-tauri-js
  -> 捕获 WebView JS 错误
  -> 调用 core 生成事件
  -> 通过 HTTP 或 Rust command 上报

tauri plugin / rust bridge
  -> 捕获 Rust panic
  -> 捕获日志
  -> 补充系统信息
  -> 支持离线缓存
```

第一阶段可以先只做前端 JS 层，复用 browser SDK。

第二阶段再加入：

- Rust panic hook。
- Tauri command bridge。
- 本地文件缓存。
- 应用版本、系统版本、窗口信息。

## 9. 鸿蒙 SDK

鸿蒙建议使用 ArkTS adapter。

职责：

```text
ArkTS 异常捕获
Promise rejection
应用生命周期
页面路由
设备信息
网络状态
日志 breadcrumb
离线缓存
```

第一阶段目标：

- 提供 ArkTS 初始化 API。
- 支持手动 captureException。
- 支持全局错误钩子。
- 支持 HTTP transport。
- 支持基础设备和应用信息。

## 10. 混合移动端 SDK

混合移动端分两类：

```text
WebView JS 错误
Native 容器错误
```

第一阶段：

- 复用 browser SDK 捕获 WebView JS 错误。
- 通过 bridge 注入 app、device、release 信息。

第二阶段：

- Android native crash。
- iOS native crash。
- React Native / Flutter adapter。

## 11. Source Map 支持

Source Map 是 Web / React / Vue 错误定位的关键能力。

SDK 侧职责：

```text
上报 release
上报 dist
上报 abs_path / filename
保留 minified stacktrace
```

CLI 或构建插件职责：

```text
上传 source map
关联 release
注入 release 版本
删除生产产物中的 sourceMappingURL 可选
```

后续可提供：

```text
@your/sdk-cli
vite-plugin-your-sdk
webpack-plugin-your-sdk
```

## 12. 脱敏策略

默认过滤字段：

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

SDK 支持：

```ts
beforeSend(event) {
  return event;
}
```

如果 `beforeSend` 返回 `null`，事件不上报。

## 13. 发送策略

SDK 发送要求：

- 上报失败不能影响业务代码。
- 默认使用异步发送。
- 支持 flush。
- 支持 sampleRate。
- 支持最大队列长度。
- 支持失败重试，但必须限制次数。
- 浏览器环境优先使用 `sendBeacon` 处理页面卸载场景。
- 普通场景使用 `fetch`。

Transport 抽象：

```ts
interface Transport {
  send(envelope: Envelope): Promise<TransportResult>;
  flush?(timeout?: number): Promise<boolean>;
}
```

## 14. 分阶段实现

### 阶段 1：TypeScript Core + Browser

目标：

```text
Web 应用可以初始化 SDK
可以捕获 JS 错误和 Promise rejection
可以手动上报错误
可以发送到服务端 ingestion API
```

任务：

- 初始化 monorepo。
- 实现 `@your/sdk-core`。
- 实现 DSN parser。
- 实现 event model。
- 实现 scope。
- 实现 breadcrumb。
- 实现 stacktrace parser。
- 实现 fetch transport。
- 实现 beforeSend。
- 实现 sampleRate。
- 实现 `@your/sdk-browser`。
- 接入 `window.onerror`。
- 接入 `unhandledrejection`。

### 阶段 2：React 和 Vue

目标：

```text
React / Vue 项目可以以框架原生方式接入 SDK
```

任务：

- 实现 `@your/sdk-react`。
- 实现 ErrorBoundary。
- 实现 component stack 采集。
- 实现 `@your/sdk-vue`。
- 实现 Vue plugin。
- 接入 Vue errorHandler。
- 接入 Vue Router breadcrumb。

### 阶段 3：可靠性增强

目标：

```text
SDK 在弱网、页面关闭、移动端 WebView 中更可靠
```

任务：

- 实现离线缓存接口。
- 浏览器使用 localStorage / IndexedDB 缓存失败事件。
- 支持重试退避。
- 支持 sendBeacon。
- 支持最大事件大小限制。
- 支持 SDK client report。

### 阶段 4：Tauri 和鸿蒙

目标：

```text
桌面端和鸿蒙应用可以接入错误收集
```

任务：

- 实现 `@your/sdk-tauri`。
- 复用 browser 错误捕获。
- 增加 Tauri app context。
- 设计 Rust bridge。
- 实现 `@your/sdk-harmony`。
- 实现 ArkTS 初始化 API。
- 实现鸿蒙平台 context。

### 阶段 5：构建工具和 Source Map

目标：

```text
生产环境错误可以还原到源码位置
```

任务：

- 实现 release 注入。
- 实现 CLI 登录和上传。
- 实现 Source Map 上传。
- 实现 Vite 插件。
- 实现 Webpack 插件。

### 阶段 6：更多平台

候选平台：

- React Native。
- Flutter。
- Android。
- iOS。
- 小程序。

原则：

- 先实现手动 captureException。
- 再实现全局错误捕获。
- 最后实现 native crash 和性能数据。

## 15. 开发原则

- Core 不依赖浏览器全局对象。
- 平台 API 只出现在对应 adapter。
- SDK 不能抛出未捕获异常影响业务。
- 默认不开启高风险数据采集。
- 上报链路必须有大小限制和频率保护。
- 所有平台共享同一事件协议。
- 新平台优先通过 adapter 扩展，而不是复制 core。

