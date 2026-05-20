import { useEffect, useMemo, useState } from "react";
import { Empty } from "../../components/Empty";
import { Pagination } from "../../components/Pagination";
import { QueryFilters, type QueryFilterValues } from "../../components/QueryFilters";
import { useEvents } from "../../hooks/useEvents";
import { EventItem } from "../../services/api";
import { datetimeLocalToRFC3339, formatDate, levelClass } from "../../utils/format";

const pageSize = 20;
const emptyFilters: QueryFilterValues = {
  level: "",
  environment: "",
  release: "",
  since: "",
  until: "",
};

export function EventsPage({
  projectId,
  query,
  refreshKey,
  openEvent,
  onLoadingChange,
  onError,
}: {
  projectId: string;
  query: string;
  refreshKey: number;
  openEvent: (event: EventItem) => void;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string) => void;
}) {
  const [offset, setOffset] = useState(0);
  const [filters, setFilters] = useState<QueryFilterValues>(emptyFilters);

  const params = {
    level: filters.level || undefined,
    environment: filters.environment.trim() || undefined,
    release: filters.release.trim() || undefined,
    since: datetimeLocalToRFC3339(filters.since),
    until: datetimeLocalToRFC3339(filters.until),
    limit: pageSize,
    offset,
  };
  const { data, loading, error, loadEvent } = useEvents(projectId, params, refreshKey);
  const { items: events, page } = data;
  const filteredEvents = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return events;
    }
    return events.filter((event) => `${event.message} ${event.exception_type} ${event.level} ${event.release ?? ""} ${event.environment ?? ""}`.toLowerCase().includes(needle));
  }, [events, query]);

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);
  useEffect(() => setOffset(0), [projectId, filters.level, filters.environment, filters.release, filters.since, filters.until]);

  return (
    <section className="grid gap-4 p-4 lg:p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-slate-950">事件查询</h1>
          <p className="mt-1 text-sm text-slate-500">按级别、环境、版本和时间范围定位原始事件。</p>
        </div>
        <span className="text-sm text-slate-500">共 {page.total} 个事件</span>
      </div>

      <QueryFilters
        values={filters}
        onChange={setFilters}
        onReset={() => {
          setFilters(emptyFilters);
          setOffset(0);
        }}
      />

      <div className="card overflow-hidden">
        <div className="hidden min-h-10 grid-cols-[minmax(260px,1fr)_88px_110px_110px_126px] items-center gap-3 bg-slate-50 px-4 text-xs font-semibold uppercase text-slate-500 md:grid">
          <span>消息</span>
          <span>级别</span>
          <span>环境</span>
          <span>版本</span>
          <span>时间</span>
        </div>
        {filteredEvents.map((event) => (
          <button
            key={event.event_id}
            className="grid w-full gap-3 border-t border-slate-100 px-4 py-3 text-left hover:bg-slate-50 md:grid-cols-[minmax(260px,1fr)_88px_110px_110px_126px] md:items-center"
            onClick={() => void loadEvent(event.event_id).then(openEvent).catch((err) => onError(err instanceof Error ? err.message : "事件加载失败"))}
          >
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-900">{event.message || event.exception_value || event.event_id}</div>
              <div className="truncate text-xs text-slate-500">{event.exception_type || event.event_id}</div>
            </div>
            <span className={levelClass(event.level)}>{event.level}</span>
            <span className="truncate text-sm text-slate-600">{event.environment || "-"}</span>
            <span className="truncate text-sm text-slate-600">{event.release || "-"}</span>
            <span className="text-sm text-slate-600">{formatDate(event.timestamp)}</span>
          </button>
        ))}
        {filteredEvents.length === 0 && <Empty label="暂无事件" />}
      </div>

      <Pagination page={page} setOffset={setOffset} />
    </section>
  );
}
