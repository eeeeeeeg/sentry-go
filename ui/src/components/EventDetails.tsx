import { Activity, AlertTriangle, Code2, Cpu, ExternalLink, Globe2, Monitor, MousePointerClick, Route, Send, Terminal, UserRound } from "lucide-react";
import { EventBreadcrumb, EventItem, EventNamedContext } from "../services/api";
import { formatDate, levelClass, prettyJSON } from "../utils/format";
import { Empty } from "./Empty";

type AnyRecord = Record<string, unknown>;

export function EventDetails({ event }: { event: EventItem }) {
  const raw = parseJSON(event.raw_event);
  const exception = rawObject(raw?.exception) ?? { type: event.exception_type, value: event.exception_value };
  const exceptionValue = latestException(exception);
  const stacktrace = latestStacktrace(exception);
  const mechanism = rawObject(exceptionValue?.mechanism);
  const handled = valueAt(mechanism, "handled");
  const browser = event.browser ?? contextFromRaw(raw, "browser");
  const os = event.os ?? contextFromRaw(raw, "os");
  const device = event.device ?? contextFromRaw(raw, "device");
  const culture = event.culture ?? rawObject(rawObject(raw?.contexts)?.culture);
  const request = event.request ?? rawObject(raw?.request);
  const user = event.user ?? rawObject(raw?.user);
  const breadcrumbs = event.breadcrumbs?.length ? event.breadcrumbs : breadcrumbsFromRaw(raw);
  const tags = parseJSON(event.tags);
  const sdk = rawObject(raw?.sdk) ?? { name: event.sdk_name, version: event.sdk_version };
  const trace = event.trace ?? rawObject(rawObject(raw?.contexts)?.trace);

  return (
    <div className="grid gap-4">
      <section className="card p-4">
        <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="text-xs font-semibold uppercase text-slate-500">事件</div>
            <h2 className="mt-1 break-all text-lg font-semibold text-slate-950">{event.event_id}</h2>
            <p className="mt-1 text-sm text-slate-500">{event.message || event.exception_value || "-"}</p>
          </div>
          <span className={levelClass(event.level)}>{event.level || "error"}</span>
        </div>
        <div className="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
          <SummaryItem label="环境" value={event.environment || "-"} />
          <SummaryItem label="平台" value={event.platform || "-"} />
          <SummaryItem label="发生时间" value={formatDate(event.timestamp)} />
          <SummaryItem label="接收时间" value={formatDate(event.received_at)} />
          <SummaryItem label="SDK" value={compactNameVersion(event.sdk_name || valueAt(sdk, "name"), event.sdk_version || valueAt(sdk, "version")) || "-"} />
          <SummaryItem label="处理状态" value={handled === "false" ? "未处理" : handled === "true" ? "已处理" : "-"} tone={handled === "false" ? "danger" : undefined} />
        </div>
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="grid gap-4">
          <section className="card p-4">
            <SectionTitle icon={AlertTriangle} title="异常" />
            <div className="mt-3 rounded-md border border-red-100 bg-red-50 p-3">
              <div className="text-sm font-semibold text-red-800">{event.exception_type || valueAt(exceptionValue, "type") || "Exception"}</div>
              <div className="mt-1 break-words text-sm text-red-700">{event.exception_value || valueAt(exceptionValue, "value") || event.message || "-"}</div>
            </div>
            {mechanism && (
              <div className="mt-3 grid gap-2 rounded-md border border-slate-200 p-3 text-sm md:grid-cols-3">
                <KeyValue label="机制" value={valueAt(mechanism, "type") || "-"} />
                <KeyValue label="是否处理" value={handled === "false" ? "未处理" : handled === "true" ? "已处理" : "-"} />
                <KeyValue label="来源" value={valueAt(mechanism, "source") || "-"} />
              </div>
            )}
            {stacktrace.length > 0 && (
              <div className="mt-4 grid gap-2">
                {stacktrace.map((frame, index) => (
                  <div key={`${valueAt(frame, "filename")}-${index}`} className="rounded-md border border-slate-200 px-3 py-2 text-sm">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="break-all font-medium text-slate-800">{valueAt(frame, "function") || "<anonymous>"}</div>
                      {valueAt(frame, "in_app") && <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600">in_app: {valueAt(frame, "in_app")}</span>}
                    </div>
                    <div className="mt-1 break-all text-xs text-slate-500">
                      {valueAt(frame, "filename") || valueAt(frame, "abs_path") || "-"}
                      {valueAt(frame, "lineno") ? `:${valueAt(frame, "lineno")}` : ""}
                      {valueAt(frame, "colno") ? `:${valueAt(frame, "colno")}` : ""}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section className="card overflow-hidden">
            <div className="border-b border-slate-100 px-4 py-3">
              <SectionTitle icon={Activity} title="面包屑" />
            </div>
            <div className="grid">
              {breadcrumbs.map((breadcrumb: EventBreadcrumb, index: number) => (
                <BreadcrumbRow key={`${breadcrumb.category}-${index}`} breadcrumb={breadcrumb} />
              ))}
              {breadcrumbs.length === 0 && <Empty label="暂无面包屑" />}
            </div>
          </section>
        </div>

        <aside className="grid content-start gap-4">
          <section className="card p-4">
            <SectionTitle icon={Monitor} title="运行环境" />
            <div className="mt-3 grid gap-3">
              <ContextCard title="浏览器" context={browser} fallback="未知浏览器" />
              <ContextCard title="操作系统" context={os} fallback="未知操作系统" />
              <ContextCard title="设备" context={device} fallback="未知设备" />
              <KeyValue label="语言" value={valueAt(culture, "locale") || "-"} />
              <KeyValue label="时区" value={valueAt(culture, "timezone") || "-"} />
            </div>
          </section>

          <section className="card p-4">
            <SectionTitle icon={Send} title="请求" />
            <div className="mt-3 grid gap-2">
              <KeyValue label="URL" value={valueAt(request, "url") || "-"} />
              <KeyValue label="Method" value={valueAt(request, "method") || "GET"} />
              <KeyValue label="User-Agent" value={requestUserAgent(request) || "-"} />
              <CollapsibleJSON title="Headers" value={rawObject(request?.headers)} />
            </div>
          </section>

          <InfoPanel title="Tags" icon={Globe2} value={tags ?? { environment: event.environment }} />
          <InfoPanel title="用户" icon={UserRound} value={user ?? { id: event.user_id }} />
          <InfoPanel title="Trace" icon={Route} value={trace} />
          <InfoPanel title="SDK" icon={Code2} value={sdk} />
        </aside>
      </section>

      <section className="card overflow-hidden">
        <div className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-slate-800">原始事件</div>
        <pre className="max-h-[520px] overflow-auto bg-slate-950 p-4 text-xs leading-6 text-slate-100">{prettyJSON(event.raw_event)}</pre>
      </section>
    </div>
  );
}

function BreadcrumbRow({ breadcrumb }: { breadcrumb: EventBreadcrumb }) {
  const category = breadcrumb.category || breadcrumb.type || "-";
  const data = breadcrumb.data ?? {};
  const icon = breadcrumbIcon(category);
  const primary = breadcrumbPrimary(category, breadcrumb, data);
  const secondary = breadcrumbSecondary(category, data);
  const status = valueAt(data, "status_code");

  return (
    <div className="grid gap-3 border-b border-slate-100 px-4 py-3 last:border-b-0 md:grid-cols-[32px_130px_minmax(0,1fr)_120px]">
      <div className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-100 text-slate-600">{icon}</div>
      <div className="min-w-0">
        <div className="truncate text-xs font-semibold text-slate-600">{category}</div>
        {breadcrumb.level && <div className="mt-1 text-xs text-slate-400">{breadcrumb.level}</div>}
      </div>
      <div className="min-w-0">
        <div className="break-words text-sm font-medium text-slate-850">{primary || "-"}</div>
        {secondary && <div className="mt-1 break-words text-xs text-slate-500">{secondary}</div>}
      </div>
      <div className="flex items-start justify-end gap-2 text-xs text-slate-500">
        {status && <span className={Number(status) >= 400 ? "rounded-full bg-amber-100 px-2 py-0.5 font-semibold text-amber-800" : "rounded-full bg-emerald-100 px-2 py-0.5 font-semibold text-emerald-700"}>{status}</span>}
        <span>{formatDate(breadcrumb.timestamp)}</span>
      </div>
    </div>
  );
}

function breadcrumbIcon(category: string) {
  if (category.includes("click") || category.includes("input")) return <MousePointerClick className="h-4 w-4" />;
  if (category === "xhr" || category === "fetch") return <Send className="h-4 w-4" />;
  if (category === "navigation") return <Route className="h-4 w-4" />;
  if (category === "console") return <Terminal className="h-4 w-4" />;
  if (category === "sentry.event") return <AlertTriangle className="h-4 w-4" />;
  return <Activity className="h-4 w-4" />;
}

function breadcrumbPrimary(category: string, breadcrumb: EventBreadcrumb, data: AnyRecord) {
  if (category === "navigation") return `${valueAt(data, "from") || "-"} -> ${valueAt(data, "to") || "-"}`;
  if (category === "xhr" || category === "fetch") return `${valueAt(data, "method") || "GET"} ${valueAt(data, "url") || "-"}`;
  if (category === "console") return breadcrumb.message || formatUnknown(data.arguments);
  if (category === "sentry.event") return breadcrumb.message || valueAt(data, "event_id");
  return breadcrumb.message || valueAt(data, "selector") || valueAt(data, "url");
}

function breadcrumbSecondary(category: string, data: AnyRecord) {
  if (category === "console") return valueAt(data, "logger");
  if (category === "sentry.event") return valueAt(data, "event_id");
  if (category === "xhr" || category === "fetch") return valueAt(data, "method") && valueAt(data, "url") ? "" : formatUnknown(data);
  return "";
}

function SectionTitle({ icon: Icon, title }: { icon: typeof AlertTriangle; title: string }) {
  return (
    <div className="flex items-center gap-2 text-sm font-semibold text-slate-800">
      <Icon className="h-4 w-4 text-slate-500" />
      {title}
    </div>
  );
}

function SummaryItem({ label, value, tone }: { label: string; value: string; tone?: "danger" }) {
  return (
    <div className="min-w-0 rounded-md border border-slate-200 p-3">
      <div className="text-xs font-semibold text-slate-500">{label}</div>
      <div className={tone === "danger" ? "mt-1 truncate text-sm font-semibold text-red-700" : "mt-1 truncate text-sm font-semibold text-slate-900"}>{value}</div>
    </div>
  );
}

function ContextCard({ title, context, fallback }: { title: string; context?: EventNamedContext; fallback: string }) {
  const name = context?.name || context?.family || fallback;
  const version = context?.version;
  return (
    <div className="rounded-md border border-slate-200 p-3">
      <div className="text-xs font-semibold text-slate-500">{title}</div>
      <div className="mt-1 text-sm font-semibold text-slate-900">{name}</div>
      <div className="mt-1 text-xs text-slate-500">{version ? `版本: ${version}` : "版本: -"}</div>
    </div>
  );
}

function InfoPanel({ title, icon, value }: { title: string; icon: typeof AlertTriangle; value: unknown }) {
  return (
    <section className="card overflow-hidden">
      <div className="border-b border-slate-100 px-4 py-3">
        <SectionTitle icon={icon} title={title} />
      </div>
      <div className="p-4">{isPlainObject(value) ? <KeyValueList value={value} /> : <div className="text-sm text-slate-500">-</div>}</div>
    </section>
  );
}

function CollapsibleJSON({ title, value }: { title: string; value?: AnyRecord }) {
  if (!value || Object.keys(value).length === 0) return null;
  return (
    <details className="rounded-md border border-slate-200 p-3">
      <summary className="cursor-pointer text-xs font-semibold text-slate-600">{title}</summary>
      <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap text-xs leading-5 text-slate-700">{JSON.stringify(value, null, 2)}</pre>
    </details>
  );
}

function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[82px_minmax(0,1fr)] gap-2 text-xs">
      <span className="font-semibold text-slate-500">{label}</span>
      <span className="min-w-0 break-words text-slate-800">{value}</span>
    </div>
  );
}

function KeyValueList({ value, compact = false }: { value: AnyRecord; compact?: boolean }) {
  const entries = Object.entries(value).filter(([, item]) => item !== undefined && item !== "");
  if (entries.length === 0) return <div className="text-sm text-slate-500">-</div>;
  return (
    <div className={compact ? "mt-2 grid gap-1" : "grid gap-2"}>
      {entries.map(([key, item]) => (
        <KeyValue key={key} label={key} value={formatUnknown(item)} />
      ))}
    </div>
  );
}

function parseJSON(raw?: string) {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw) as AnyRecord;
  } catch {
    return undefined;
  }
}

function contextFromRaw(raw: AnyRecord | undefined, key: string): EventNamedContext | undefined {
  const context = rawObject(rawObject(raw?.contexts)?.[key]);
  if (!context) return undefined;
  return { name: valueAt(context, "name"), version: valueAt(context, "version"), family: valueAt(context, "family"), data: context };
}

function breadcrumbsFromRaw(raw: AnyRecord | undefined) {
  const breadcrumbs = raw?.breadcrumbs;
  const nestedValues = rawObject(breadcrumbs)?.values;
  const values: unknown[] = Array.isArray(breadcrumbs) ? breadcrumbs : Array.isArray(nestedValues) ? nestedValues : [];
  return values.flatMap((item): EventBreadcrumb[] => {
    const entry = rawObject(item);
    if (!entry) return [];
    return [{
      timestamp: valueAt(entry, "timestamp"),
      type: valueAt(entry, "type"),
      category: valueAt(entry, "category"),
      level: valueAt(entry, "level"),
      message: valueAt(entry, "message"),
      data: rawObject(entry.data),
    }];
  });
}

function latestException(exception: unknown) {
  const values = rawObject(exception)?.values;
  if (Array.isArray(values) && values.length > 0) return rawObject(values[values.length - 1]);
  return rawObject(exception);
}

function latestStacktrace(exception: unknown) {
  const latest = latestException(exception);
  const frames = rawObject(rawObject(latest?.stacktrace)?.frames) ? [] : rawObject(latest?.stacktrace)?.frames;
  if (Array.isArray(frames)) return frames.map((frame) => rawObject(frame) ?? {}).reverse();
  return [];
}

function requestUserAgent(request?: AnyRecord) {
  const headers = rawObject(request?.headers);
  if (!headers) return "";
  const entry = Object.entries(headers).find(([key]) => key.toLowerCase() === "user-agent");
  return entry ? formatUnknown(entry[1]) : "";
}

function rawObject(value: unknown): AnyRecord | undefined {
  return isPlainObject(value) ? value : undefined;
}

function isPlainObject(value: unknown): value is AnyRecord {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function valueAt(value: unknown, key: string) {
  const item = rawObject(value)?.[key];
  if (typeof item === "string") return item;
  if (typeof item === "number" || typeof item === "boolean") return String(item);
  return "";
}

function compactNameVersion(name?: string, version?: string) {
  if (!name && !version) return "";
  return [name, version].filter(Boolean).join("@");
}

function formatUnknown(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (value === null || value === undefined) return "-";
  return JSON.stringify(value);
}
