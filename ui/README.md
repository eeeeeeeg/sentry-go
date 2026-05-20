# Sentry Lite UI

React + Vite + Tailwind CSS 管理后台。

## 技术栈

```text
React
Vite
Tailwind CSS
Headless UI
React Router
Axios
lucide-react
```

## 路由规划

```text
/overview     监控 / 概览
/issues       错误管理 / Issue
/events       错误管理 / 事件检索
/alerts       通知 / 告警中心
/projects     配置 / 项目管理
/system       系统 / 系统状态
```

## 请求封装

统一 Axios 实例位于：

```text
src/http.ts
```

当前已实现：

- request interceptor：从 `localStorage.sentry-lite-token` 注入 Bearer token。
- response interceptor：统一提取后端 `{ error, message }` 错误信息。
- 401 时派发 `sentry-lite:unauthorized` 浏览器事件，后续可接登录态处理。

## 目录结构

```text
src/
  app/
    layout/
      Sidebar.tsx
      Topbar.tsx
    routes.tsx
  components/
    Charts.tsx
    Empty.tsx
    EventDialog.tsx
    MetricCard.tsx
    Panel.tsx
  pages/
    alerts/
      AlertsPage.tsx
    events/
      EventsPage.tsx
    issues/
      IssuesPage.tsx
    overview/
      OverviewPage.tsx
    projects/
      ProjectsPage.tsx
    system/
      SystemPage.tsx
  services/
    api.ts
    http.ts
  hooks/
    useAlerts.ts
    useAsyncData.ts
    useEvents.ts
    useIssues.ts
    useOverviewData.ts
    useProjects.ts
  utils/
    format.ts
```

`App.tsx` 只负责全局数据装配、过滤、弹窗状态和路由挂载；页面内容放在 `pages/*`。

页面数据加载已下沉到 hooks：

- `useOverviewData`：概览统计、趋势、分布。
- `useIssues`：Issue 列表和状态变更。
- `useEvents`：事件列表和事件详情按需加载。
- `useAlerts`：告警规则、投递记录、创建和启停。
- `useProjects`：项目列表、项目配置、DSN Key 创建和启停。
- `useAsyncData`：统一 loading / error / reload 状态。

`App.tsx` 只保留项目 ID、搜索词、全局 loading/error、刷新触发器和事件详情弹窗状态。

## 本地开发

```bash
cd ui
npm install
npm run dev
```

默认地址：

```text
http://localhost:5173
```

Vite 已将 `/api`、`/healthz`、`/readyz` 代理到：

```text
http://localhost:8080
```

## 构建

```bash
npm run build
```
