import { request } from "./http";

export type Issue = {
  id: string;
  project_id: string;
  fingerprint: string;
  title: string;
  culprit?: string;
  level: string;
  status: string;
  first_seen: string;
  last_seen: string;
  event_count: number;
  user_count: number;
  release?: string;
  environment?: string;
};

export type IssueStatusChange = {
  id: string;
  issue_id: string;
  old_status: string;
  new_status: string;
  reason?: string;
  created_at: string;
};

export type EventNamedContext = {
  name?: string;
  version?: string;
  family?: string;
  data?: Record<string, unknown>;
};

export type EventBreadcrumb = {
  timestamp?: string;
  type?: string;
  category?: string;
  level?: string;
  message?: string;
  data?: Record<string, unknown>;
};

export type EventItem = {
  event_id: string;
  project_id: string;
  issue_id?: string;
  timestamp: string;
  received_at: string;
  platform: string;
  level: string;
  message: string;
  exception_type: string;
  exception_value: string;
  release?: string;
  environment?: string;
  user_id?: string;
  tags: string;
  contexts: string;
  raw_event?: string;
  runtime_name?: string;
  runtime_version?: string;
  sdk_name?: string;
  sdk_version?: string;
  browser?: EventNamedContext;
  os?: EventNamedContext;
  device?: EventNamedContext;
  culture?: Record<string, unknown>;
  trace?: Record<string, unknown>;
  request?: Record<string, unknown>;
  user?: Record<string, unknown>;
  breadcrumbs?: EventBreadcrumb[];
};

export type TrendPoint = {
  bucket: string;
  count: number;
};

export type BreakdownItem = {
  key: string;
  count: number;
};

export type ReleaseSummary = {
  release: string;
  event_count: number;
  issue_count: number;
  user_count: number;
  first_seen: string;
  last_seen: string;
  environment?: string;
};

type TopIssueItem = {
  issue_id: string;
  count: number;
};

export type AlertRule = {
  id: string;
  project_id: string;
  name: string;
  event_type: string;
  channel: string;
  webhook_url?: string;
  min_level: string;
  threshold_count: number;
  window_seconds: number;
  cooldown_seconds: number;
  status: string;
};

export type AlertDelivery = {
  id: string;
  alert_id?: string;
  project_id: string;
  issue_id: string;
  event_id: string;
  event_type: string;
  channel: string;
  status: string;
  error?: string;
  delivered_at?: string;
  created_at: string;
};

export type Project = {
  id: string;
  organization_id: string;
  sentry_project_id: string;
  slug: string;
  name: string;
  platform: string;
  status: string;
  sample_rate: number;
  created_at: string;
  updated_at: string;
};

export type ProjectKey = {
  id: string;
  project_id: string;
  public_key: string;
  name: string;
  status: string;
  rate_limit_per_minute: number;
  created_at: string;
  updated_at: string;
};

type ListResponse<T> = {
  items?: T[];
  page?: PageMeta;
};

export type PageMeta = {
  limit: number;
  offset: number;
  total: number;
};

export type Paginated<T> = {
  items: T[];
  page: PageMeta;
};

const defaultPage: PageMeta = { limit: 20, offset: 0, total: 0 };

export type IssueListParams = {
  status?: string;
  level?: string;
  environment?: string;
  release?: string;
  since?: string;
  until?: string;
  limit?: number;
  offset?: number;
};

export type EventListParams = {
  issue_id?: string;
  level?: string;
  environment?: string;
  release?: string;
  since?: string;
  until?: string;
  limit?: number;
  offset?: number;
};

export type StatsParams = {
  environment?: string;
  release?: string;
  since?: string;
  until?: string;
  limit?: number;
};

function paginated<T>(response: ListResponse<T>): Paginated<T> {
  return { items: response.items ?? [], page: response.page ?? defaultPage };
}

export async function listProjects(params?: { limit?: number; offset?: number }): Promise<Project[]> {
  const result = await listProjectsPage(params);
  return result.items;
}

export async function listProjectsPage(params?: { limit?: number; offset?: number }): Promise<Paginated<Project>> {
  const result = await request<ListResponse<Project>>({ method: "GET", url: "/api/projects", params });
  return paginated(result);
}

export async function listProjectKeys(projectId: string, params?: { limit?: number; offset?: number }): Promise<ProjectKey[]> {
  const result = await listProjectKeysPage(projectId, params);
  return result.items ?? [];
}

export async function listProjectKeysPage(projectId: string, params?: { limit?: number; offset?: number }): Promise<Paginated<ProjectKey>> {
  const result = await request<ListResponse<ProjectKey>>({ method: "GET", url: `/api/projects/${projectId}/keys`, params });
  return paginated(result);
}

export async function createProject(body: {
  organization_slug?: string;
  slug: string;
  name: string;
  platform: string;
  sample_rate: number;
}): Promise<Project> {
  return request<Project>({ method: "POST", url: "/api/projects", data: body });
}

export async function updateProject(projectId: string, body: {
  name?: string;
  platform?: string;
  sample_rate?: number;
}): Promise<Project> {
  return request<Project>({ method: "PATCH", url: `/api/projects/${projectId}`, data: body });
}

export async function updateProjectStatus(projectId: string, status: string): Promise<Project> {
  return request<Project>({ method: "PATCH", url: `/api/projects/${projectId}/status`, data: { status } });
}

export async function createProjectKey(projectId: string, body: {
  name: string;
  rate_limit_per_minute: number;
}): Promise<ProjectKey> {
  return request<ProjectKey>({ method: "POST", url: `/api/projects/${projectId}/keys`, data: body });
}

export async function updateProjectKey(keyId: string, body: {
  name?: string;
  rate_limit_per_minute?: number;
}): Promise<ProjectKey> {
  return request<ProjectKey>({ method: "PATCH", url: `/api/project-keys/${keyId}`, data: body });
}

export async function updateProjectKeyStatus(keyId: string, status: string): Promise<ProjectKey> {
  return request<ProjectKey>({ method: "PATCH", url: `/api/project-keys/${keyId}/status`, data: { status } });
}

export async function listIssues(projectId: string, status: string): Promise<Issue[]> {
  const result = await listIssuesPage(projectId, { status, limit: 100 });
  return result.items;
}

export async function listIssuesPage(projectId: string, params?: IssueListParams): Promise<Paginated<Issue>> {
  const result = await request<ListResponse<Issue>>({
    method: "GET",
    url: `/api/projects/${projectId}/issues`,
    params,
  });
  return paginated(result);
}

export async function getIssue(issueId: string): Promise<Issue> {
  return request<Issue>({ method: "GET", url: `/api/issues/${issueId}` });
}

export async function listIssueStatusChanges(issueId: string): Promise<IssueStatusChange[]> {
  const result = await request<ListResponse<IssueStatusChange>>({ method: "GET", url: `/api/issues/${issueId}/status-changes` });
  return result.items ?? [];
}

export async function listIssueUserReports(issueId: string): Promise<unknown[]> {
  const result = await request<ListResponse<unknown>>({ method: "GET", url: `/api/issues/${issueId}/user-reports` });
  return result.items ?? [];
}

export async function listIssueComments(issueId: string): Promise<unknown[]> {
  const result = await request<ListResponse<unknown>>({ method: "GET", url: `/api/issues/${issueId}/comments` });
  return result.items ?? [];
}

export async function listMergedIssues(issueId: string): Promise<unknown[]> {
  const result = await request<ListResponse<unknown>>({ method: "GET", url: `/api/issues/${issueId}/merged` });
  return result.items ?? [];
}

export async function listEvents(projectId: string, issueId?: string): Promise<EventItem[]> {
  const result = await listEventsPage(projectId, { issue_id: issueId || undefined, limit: 100 });
  return result.items;
}

export async function listEventsPage(projectId: string, params?: EventListParams): Promise<Paginated<EventItem>> {
  const result = await request<ListResponse<EventItem>>({
    method: "GET",
    url: `/api/projects/${projectId}/events`,
    params,
  });
  return paginated(result);
}

export async function getEvent(eventId: string): Promise<EventItem> {
  return request<EventItem>({ method: "GET", url: `/api/events/${eventId}` });
}

export async function getTrend(projectId: string, params?: StatsParams): Promise<TrendPoint[]> {
  const result = await request<ListResponse<TrendPoint>>({ method: "GET", url: `/api/projects/${projectId}/stats/trend`, params });
  return result.items ?? [];
}

export async function getLevels(projectId: string, params?: StatsParams): Promise<BreakdownItem[]> {
  const result = await request<ListResponse<BreakdownItem>>({ method: "GET", url: `/api/projects/${projectId}/stats/levels`, params });
  return result.items ?? [];
}

export async function getTopIssues(projectId: string, params?: StatsParams): Promise<BreakdownItem[]> {
  const result = await request<ListResponse<TopIssueItem>>({ method: "GET", url: `/api/projects/${projectId}/stats/top-issues`, params });
  return (result.items ?? []).map((item) => ({ key: item.issue_id, count: item.count }));
}

export async function getTopReleases(projectId: string, params?: StatsParams): Promise<BreakdownItem[]> {
  const result = await request<ListResponse<BreakdownItem>>({ method: "GET", url: `/api/projects/${projectId}/stats/top-releases`, params });
  return result.items ?? [];
}

export async function listReleases(projectId: string, params?: StatsParams): Promise<ReleaseSummary[]> {
  const result = await request<ListResponse<ReleaseSummary>>({ method: "GET", url: `/api/projects/${projectId}/releases`, params });
  return result.items ?? [];
}

export async function listAlerts(projectId: string): Promise<AlertRule[]> {
  const result = await request<ListResponse<AlertRule>>({ method: "GET", url: `/api/projects/${projectId}/alerts` });
  return result.items ?? [];
}

export async function listAlertDeliveries(projectId: string): Promise<AlertDelivery[]> {
  const result = await listAlertDeliveriesPage(projectId, { limit: 100 });
  return result.items;
}

export async function listAlertDeliveriesPage(projectId: string, params?: { status?: string; limit?: number; offset?: number }): Promise<Paginated<AlertDelivery>> {
  const result = await request<ListResponse<AlertDelivery>>({
    method: "GET",
    url: `/api/projects/${projectId}/alert-deliveries`,
    params,
  });
  return paginated(result);
}

export async function updateIssueStatus(issueId: string, status: string): Promise<Issue> {
  return request<Issue>({ method: "PATCH", url: `/api/issues/${issueId}/status`, data: { status } });
}

export async function updateAlertStatus(alertId: string, status: string): Promise<AlertRule> {
  return request<AlertRule>({ method: "PATCH", url: `/api/alerts/${alertId}/status`, data: { status } });
}

export async function testAlert(alertId: string): Promise<{ status: string }> {
  return request<{ status: string }>({ method: "POST", url: `/api/alerts/${alertId}/test` });
}

export async function createWebhookAlert(projectId: string, body: {
  name: string;
  event_type: string;
  webhook_url: string;
  min_level: string;
  threshold_count?: number;
  window_seconds?: number;
  cooldown_seconds?: number;
}): Promise<AlertRule> {
  return request<AlertRule>({ method: "POST", url: `/api/projects/${projectId}/alerts/webhook`, data: body });
}

export async function sendTestEvent(projectId: string, publicKey: string): Promise<{ id?: string; status: string }> {
  return request<{ id?: string; status: string }>({
    method: "POST",
    url: `/api/${projectId}/envelope/`,
    headers: { "X-Sentry-Key": publicKey },
    data: {
      event_id: crypto.randomUUID?.(),
      timestamp: new Date().toISOString(),
      platform: "javascript",
      level: "error",
      message: "Sentry Lite test event",
      exception: {
        type: "TestError",
        value: "This event was sent from the project onboarding panel.",
        stacktrace: [],
      },
      release: "onboarding-demo",
      environment: "production",
      tags: { source: "dashboard-test" },
    },
  });
}

export async function getHealth(): Promise<Record<string, unknown>> {
  return request<Record<string, unknown>>({ method: "GET", url: "/healthz" });
}

export async function getReady(): Promise<Record<string, unknown>> {
  return request<Record<string, unknown>>({ method: "GET", url: "/readyz" });
}

export async function getMetrics(): Promise<string> {
  return request<string>({ method: "GET", url: "/metrics", responseType: "text" });
}
