import clsx from "clsx";
import { ArrowLeft, CheckCircle2, CircleOff, RotateCcw } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Empty } from "../../components/Empty";
import { EventDetails } from "../../components/EventDetails";
import { Pagination } from "../../components/Pagination";
import { Button } from "../../components/ui";
import {
  getEvent,
  getIssue,
  listEventsPage,
  listIssueComments,
  listIssueStatusChanges,
  listIssueUserReports,
  listMergedIssues,
  updateIssueStatus,
  type EventItem,
  type Issue,
  type IssueStatusChange,
  type PageMeta,
} from "../../services/api";
import { compactNumber, formatDate, levelClass } from "../../utils/format";

const eventPageSize = 10;
const tabs = [
  { id: "events", label: "事件详情" },
  { id: "reports", label: "用户反馈" },
  { id: "comments", label: "评论" },
  { id: "merged", label: "Merged" },
] as const;

type TabId = (typeof tabs)[number]["id"];

export function IssueDetailsPage({
  projectId,
  onLoadingChange,
  onError,
}: {
  projectId: string;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string) => void;
}) {
  const { issueId = "" } = useParams();
  const navigate = useNavigate();
  const [issue, setIssue] = useState<Issue | null>(null);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [eventsPage, setEventsPage] = useState<PageMeta>({ limit: eventPageSize, offset: 0, total: 0 });
  const [eventOffset, setEventOffset] = useState(0);
  const [selectedEvent, setSelectedEvent] = useState<EventItem | null>(null);
  const [changes, setChanges] = useState<IssueStatusChange[]>([]);
  const [reports, setReports] = useState<unknown[]>([]);
  const [comments, setComments] = useState<unknown[]>([]);
  const [merged, setMerged] = useState<unknown[]>([]);
  const [activeTab, setActiveTab] = useState<TabId>("events");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const environmentBreakdown = useMemo(() => breakdown(events, (event) => event.environment || "-"), [events]);
  const releaseBreakdown = useMemo(() => breakdown(events, (event) => event.release || "-"), [events]);

  useEffect(() => {
    setEventOffset(0);
    setSelectedEvent(null);
  }, [issueId]);

  useEffect(() => {
    if (!issueId) {
      return;
    }
    let active = true;
    setLoading(true);
    setError("");
    Promise.all([
      getIssue(issueId),
      listEventsPage(projectId, { issue_id: issueId, limit: eventPageSize, offset: eventOffset }),
      listIssueStatusChanges(issueId),
      listIssueUserReports(issueId),
      listIssueComments(issueId),
      listMergedIssues(issueId),
    ])
      .then(async ([nextIssue, nextEvents, nextChanges, nextReports, nextComments, nextMerged]) => {
        if (!active) {
          return;
        }
        setIssue(nextIssue);
        setEvents(nextEvents.items);
        setEventsPage(nextEvents.page);
        setChanges(nextChanges);
        setReports(nextReports);
        setComments(nextComments);
        setMerged(nextMerged);
        if (!selectedEvent && nextEvents.items[0]) {
          setSelectedEvent(await getEvent(nextEvents.items[0].event_id));
        }
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : "Issue 详情加载失败");
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [issueId, projectId, eventOffset]);

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);

  async function changeStatus(nextStatus: string) {
    if (!issueId) {
      return;
    }
    setError("");
    try {
      const nextIssue = await updateIssueStatus(issueId, nextStatus);
      const nextChanges = await listIssueStatusChanges(issueId);
      setIssue(nextIssue);
      setChanges(nextChanges);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Issue 状态更新失败");
    }
  }

  async function selectEvent(eventId: string) {
    setError("");
    try {
      setSelectedEvent(await getEvent(eventId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "事件加载失败");
    }
  }

  return (
    <section className="grid gap-4 p-4 lg:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <button className="mb-3 inline-flex items-center gap-2 text-sm font-medium text-slate-600 hover:text-slate-950" onClick={() => navigate("/issues")}>
            <ArrowLeft className="h-4 w-4" />
            返回 Issue 列表
          </button>
          <h1 className="break-words text-xl font-semibold text-slate-950">{issue?.title || "Issue 详情"}</h1>
          <p className="mt-1 break-all text-sm text-slate-500">{issue?.culprit || issue?.fingerprint || issueId}</p>
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          {issue?.status !== "resolved" && (
            <Button onClick={() => void changeStatus("resolved")}>
              <CheckCircle2 className="h-4 w-4" />
              解决
            </Button>
          )}
          {issue?.status !== "ignored" && (
            <Button onClick={() => void changeStatus("ignored")}>
              <CircleOff className="h-4 w-4" />
              忽略
            </Button>
          )}
          {issue && issue.status !== "unresolved" && (
            <Button onClick={() => void changeStatus("unresolved")}>
              <RotateCcw className="h-4 w-4" />
              重新打开
            </Button>
          )}
        </div>
      </div>

      {issue ? (
        <>
          <section className="card grid gap-3 p-4 md:grid-cols-4">
            <Metric label="状态" value={issue.status} />
            <Metric label="级别" value={issue.level} className={levelClass(issue.level)} />
            <Metric label="事件数" value={compactNumber(issue.event_count)} />
            <Metric label="用户数" value={compactNumber(issue.user_count)} />
            <Metric label="首次出现" value={formatDate(issue.first_seen)} />
            <Metric label="最后出现" value={formatDate(issue.last_seen)} />
            <Metric label="环境" value={issue.environment || "-"} />
            <Metric label="Release" value={issue.release || "-"} />
          </section>

          <div className="border-b border-slate-200">
            <div className="flex flex-wrap gap-2">
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  className={clsx("border-b-2 px-3 py-2 text-sm font-semibold", activeTab === tab.id ? "border-slate-950 text-slate-950" : "border-transparent text-slate-500 hover:text-slate-800")}
                  onClick={() => setActiveTab(tab.id)}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {activeTab === "events" && (
            <div className="grid gap-4 xl:grid-cols-[340px_minmax(0,1fr)]">
              <section className="grid content-start gap-4">
                <div className="card overflow-hidden">
                  <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
                    <div className="text-sm font-semibold text-slate-800">最近事件</div>
                    <div className="text-sm text-slate-500">{eventsPage.total}</div>
                  </div>
                  {events.map((event) => (
                    <button
                      key={event.event_id}
                      className={clsx("grid w-full gap-2 border-b border-slate-100 px-4 py-3 text-left last:border-b-0 hover:bg-slate-50", selectedEvent?.event_id === event.event_id && "bg-slate-50")}
                      onClick={() => void selectEvent(event.event_id)}
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className={levelClass(event.level)}>{event.level}</span>
                        <span className="text-xs text-slate-500">{formatDate(event.timestamp)}</span>
                      </div>
                      <div className="truncate text-sm font-medium text-slate-900">{event.message || event.exception_value || event.event_id}</div>
                      <div className="truncate text-xs text-slate-500">{event.environment || "-"} / {event.release || "-"}</div>
                    </button>
                  ))}
                  {events.length === 0 && <Empty label="暂无事件明细" />}
                  <div className="px-4 pb-4">
                    <Pagination page={eventsPage} setOffset={setEventOffset} />
                  </div>
                </div>

                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
                  <BreakdownPanel title="环境分布" items={environmentBreakdown} />
                  <BreakdownPanel title="Release 分布" items={releaseBreakdown} />
                </div>
                <Timeline changes={changes} />
              </section>

              <div>
                {selectedEvent ? (
                  <EventDetails event={selectedEvent} />
                ) : (
                  <section className="card p-8 text-center">
                    <div className="text-base font-semibold text-slate-900">暂无事件详情</div>
                    <p className="mx-auto mt-2 max-w-xl text-sm leading-6 text-slate-500">
                      当前 Issue 在 Postgres 中有聚合计数，但 ClickHouse 事件明细为空。已修复新的事件写入链路，后续上报的事件会在这里展示浏览器、操作系统、设备、请求和点击面包屑。
                    </p>
                  </section>
                )}
              </div>
            </div>
          )}

          {activeTab === "reports" && <Placeholder title="用户反馈" items={reports} />}
          {activeTab === "comments" && <Placeholder title="评论" items={comments} />}
          {activeTab === "merged" && <Placeholder title="合并记录" items={merged} />}
        </>
      ) : (
        <Empty label="正在加载 Issue" />
      )}
    </section>
  );
}

function Metric({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs font-semibold text-slate-500">{label}</div>
      <div className={className || "mt-1 truncate text-sm font-medium text-slate-900"}>{value}</div>
    </div>
  );
}

function BreakdownPanel({ title, items }: { title: string; items: { key: string; count: number }[] }) {
  return (
    <section className="card">
      <div className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-slate-800">{title}</div>
      <div className="grid gap-2 p-4">
        {items.map((item) => (
          <div key={item.key} className="flex items-center justify-between gap-3 text-sm">
            <span className="min-w-0 truncate text-slate-700">{item.key}</span>
            <span className="font-semibold text-slate-900">{item.count}</span>
          </div>
        ))}
        {items.length === 0 && <Empty label="暂无数据" />}
      </div>
    </section>
  );
}

function Timeline({ changes }: { changes: IssueStatusChange[] }) {
  return (
    <section className="card">
      <div className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-slate-800">状态时间线</div>
      <div className="grid">
        {changes.map((change) => (
          <div key={change.id} className="grid grid-cols-[20px_minmax(0,1fr)] gap-3 border-b border-slate-100 px-4 py-3 last:border-b-0">
            <span className="mt-1 h-2 w-2 rounded-full bg-slate-400" />
            <div className="min-w-0">
              <div className="text-sm text-slate-800">
                {change.old_status} -&gt; {change.new_status}
              </div>
              <div className="mt-1 text-xs text-slate-500">
                {formatDate(change.created_at)} {change.reason ? `/ ${change.reason}` : ""}
              </div>
            </div>
          </div>
        ))}
        {changes.length === 0 && <Empty label="暂无状态变更" />}
      </div>
    </section>
  );
}

function Placeholder({ title, items }: { title: string; items: unknown[] }) {
  return (
    <section className="card p-6">
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>
      {items.length === 0 ? <Empty label="暂无记录" /> : <pre className="mt-4 overflow-auto rounded-md bg-slate-950 p-4 text-xs text-white">{JSON.stringify(items, null, 2)}</pre>}
    </section>
  );
}

function breakdown(items: EventItem[], pick: (item: EventItem) => string) {
  const counts = new Map<string, number>();
  for (const item of items) {
    const key = pick(item);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return Array.from(counts, ([key, count]) => ({ key, count })).sort((a, b) => b.count - a.count);
}
