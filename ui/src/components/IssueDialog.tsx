import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { getIssue, listEventsPage, listIssueStatusChanges, updateIssueStatus, type EventItem, type Issue, type IssueStatusChange, type PageMeta } from "../services/api";
import { compactNumber, formatDate, levelClass } from "../utils/format";
import { Empty } from "./Empty";
import { Pagination } from "./Pagination";

const eventPageSize = 10;

export function IssueDialog({
  issueId,
  projectId,
  close,
  openEvent,
}: {
  issueId: string | null;
  projectId: string;
  close: () => void;
  openEvent: (event: EventItem) => void;
}) {
  const [issue, setIssue] = useState<Issue | null>(null);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [eventsPage, setEventsPage] = useState<PageMeta>({ limit: eventPageSize, offset: 0, total: 0 });
  const [eventOffset, setEventOffset] = useState(0);
  const [changes, setChanges] = useState<IssueStatusChange[]>([]);
  const [error, setError] = useState("");

  const environmentBreakdown = useMemo(() => breakdown(events, (event) => event.environment || "-"), [events]);
  const releaseBreakdown = useMemo(() => breakdown(events, (event) => event.release || "-"), [events]);

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

  useEffect(() => {
    setEventOffset(0);
  }, [issueId]);

  useEffect(() => {
    if (!issueId) {
      setIssue(null);
      setEvents([]);
      setEventsPage({ limit: eventPageSize, offset: 0, total: 0 });
      setChanges([]);
      return;
    }

    let active = true;
    setError("");
    Promise.all([
      getIssue(issueId),
      listEventsPage(projectId, { issue_id: issueId, limit: eventPageSize, offset: eventOffset }),
      listIssueStatusChanges(issueId),
    ])
      .then(([nextIssue, nextEvents, nextChanges]) => {
        if (!active) {
          return;
        }
        setIssue(nextIssue);
        setEvents(nextEvents.items);
        setEventsPage(nextEvents.page);
        setChanges(nextChanges);
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : "Issue 详情加载失败");
        }
      });

    return () => {
      active = false;
    };
  }, [issueId, projectId, eventOffset]);

  return (
    <Dialog open={Boolean(issueId)} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-y-0 right-0 flex w-full justify-end sm:pl-16">
        <DialogPanel className="h-full w-full max-w-5xl overflow-auto bg-white p-5 shadow-xl">
          <div className="mb-4 flex items-center justify-between gap-4">
            <DialogTitle className="truncate text-lg font-semibold text-slate-900">{issue?.title || "Issue 详情"}</DialogTitle>
            <div className="flex shrink-0 items-center gap-2">
              {issue && issue.status !== "resolved" && (
                <button className="btn h-8 px-2" onClick={() => void changeStatus("resolved")}>
                  Resolve
                </button>
              )}
              {issue && issue.status !== "ignored" && (
                <button className="btn h-8 px-2" onClick={() => void changeStatus("ignored")}>
                  Ignore
                </button>
              )}
              {issue && issue.status !== "unresolved" && (
                <button className="btn h-8 px-2" onClick={() => void changeStatus("unresolved")}>
                  Reopen
                </button>
              )}
              <button className="btn h-8 w-8 px-0" onClick={close} title="关闭">
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          {error && <div className="mb-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">{error}</div>}

          {issue ? (
            <div className="grid gap-4">
              <section className="grid gap-3 rounded-md border border-slate-200 p-4 md:grid-cols-4">
                <Metric label="状态" value={issue.status} />
                <Metric label="级别" value={issue.level} className={levelClass(issue.level)} />
                <Metric label="事件数" value={compactNumber(issue.event_count)} />
                <Metric label="用户数" value={compactNumber(issue.user_count)} />
                <Metric label="首次出现" value={formatDate(issue.first_seen)} />
                <Metric label="最后出现" value={formatDate(issue.last_seen)} />
                <Metric label="环境" value={issue.environment || "-"} />
                <Metric label="版本" value={issue.release || "-"} />
              </section>

              <div className="grid gap-4 xl:grid-cols-2">
                <BreakdownPanel title="当前页环境分布" items={environmentBreakdown} />
                <BreakdownPanel title="当前页版本分布" items={releaseBreakdown} />
              </div>

              <section className="rounded-md border border-slate-200">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
                  <div className="text-sm font-semibold text-slate-800">最近事件</div>
                  <div className="text-sm text-slate-500">共 {eventsPage.total} 条</div>
                </div>
                {events.map((event) => (
                  <button
                    key={event.event_id}
                    className="grid w-full gap-2 border-b border-slate-100 px-4 py-3 text-left last:border-b-0 md:grid-cols-[90px_minmax(0,1fr)_110px_110px_120px]"
                    onClick={() => openEvent(event)}
                  >
                    <span className={levelClass(event.level)}>{event.level}</span>
                    <span className="min-w-0 truncate text-sm text-slate-800">{event.message || event.exception_value || event.event_id}</span>
                    <span className="truncate text-sm text-slate-500">{event.environment || "-"}</span>
                    <span className="truncate text-sm text-slate-500">{event.release || "-"}</span>
                    <span className="text-sm text-slate-500">{formatDate(event.timestamp)}</span>
                  </button>
                ))}
                {events.length === 0 && <Empty label="暂无事件" />}
                <div className="px-4 pb-4">
                  <Pagination page={eventsPage} setOffset={setEventOffset} />
                </div>
              </section>

              <section className="rounded-md border border-slate-200">
                <div className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-slate-800">状态时间线</div>
                <div className="grid gap-0">
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
                </div>
                {changes.length === 0 && <Empty label="暂无状态变更" />}
              </section>
            </div>
          ) : (
            <Empty label="正在加载 Issue" />
          )}
        </DialogPanel>
      </div>
    </Dialog>
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
    <section className="rounded-md border border-slate-200">
      <div className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-slate-800">{title}</div>
      <div className="grid gap-2 p-4">
        {items.map((item) => (
          <div key={item.key} className="flex items-center justify-between gap-3 text-sm">
            <span className="min-w-0 truncate text-slate-700">{item.key}</span>
            <span className="font-semibold text-slate-900">{item.count}</span>
          </div>
        ))}
        {items.length === 0 && <Empty label="暂无分布数据" />}
      </div>
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
