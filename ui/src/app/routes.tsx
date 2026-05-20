import { Activity, Bell, Bug, FolderKanban, Gauge, GitBranch, ShieldAlert } from "lucide-react";

export const navGroups = [
  {
    title: "监控",
    items: [{ path: "/overview", label: "概览", icon: Gauge }],
  },
  {
    title: "错误管理",
    items: [
      { path: "/issues", label: "Issue", icon: Bug },
      { path: "/releases", label: "Releases", icon: GitBranch },
      { path: "/events", label: "事件检索", icon: Activity },
    ],
  },
  {
    title: "通知",
    items: [{ path: "/alerts", label: "告警中心", icon: Bell }],
  },
  {
    title: "配置",
    items: [{ path: "/projects", label: "项目管理", icon: FolderKanban }],
  },
  {
    title: "系统",
    items: [{ path: "/system", label: "系统状态", icon: ShieldAlert }],
  },
];
