import clsx from "clsx";
import { CheckCircle2, CircleOff } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Empty } from "../../components/Empty";
import { Pagination } from "../../components/Pagination";
import { QueryFilters, type QueryFilterValues } from "../../components/QueryFilters";
import { Button } from "../../components/ui";
import { useIssues } from "../../hooks/useIssues";
import { compactNumber, datetimeLocalToRFC3339, formatDate, levelClass } from "../../utils/format";

const pageSize = 20;
const emptyFilters: QueryFilterValues = {
  level: "",
  environment: "",
  release: "",
  since: "",
  until: "",
};

export function IssuesPage({
  projectId,
  query,
  status,
  setStatus,
  refreshKey,
  onLoadingChange,
  onError,
}: {
  projectId: string;
  query: string;
  status: string;
  setStatus: (status: string) => void;
  refreshKey: number;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string) => void;
}) {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [offset, setOffset] = useState(0);
  const [filters, setFilters] = useState<QueryFilterValues>(emptyFilters);

  const params = {
    status,
    level: filters.level || undefined,
    environment: filters.environment.trim() || undefined,
    release: filters.release.trim() || undefined,
    since: datetimeLocalToRFC3339(filters.since),
    until: datetimeLocalToRFC3339(filters.until),
    limit: pageSize,
    offset,
  };
  const { data, loading, error, setIssueStatus } = useIssues(projectId, params, refreshKey);
  const { items: issues, page } = data;
  const filteredIssues = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return issues;
    }
    return issues.filter((issue) => `${issue.title} ${issue.culprit ?? ""} ${issue.level} ${issue.release ?? ""} ${issue.environment ?? ""}`.toLowerCase().includes(needle));
  }, [issues, query]);

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);
  useEffect(() => setOffset(0), [projectId, status, filters.level, filters.environment, filters.release, filters.since, filters.until]);
  useEffect(() => {
    const release = searchParams.get("release") ?? "";
    const environment = searchParams.get("environment") ?? "";
    if (release || environment) {
      setFilters((current) => ({ ...current, release, environment }));
      setOffset(0);
    }
  }, [searchParams]);

  return (
    <section className="grid gap-4 p-4 lg:p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="inline-flex overflow-hidden rounded-md border border-slate-300 bg-white">
          {["unresolved", "resolved", "ignored", "all"].map((item) => (
            <button
              key={item}
              className={clsx("h-9 border-r border-slate-300 px-3 text-sm last:border-r-0", status === item ? "bg-slate-900 text-white" : "text-slate-600")}
              onClick={() => setStatus(item)}
            >
              {statusLabel(item)}
            </button>
          ))}
        </div>
        <span className="text-sm text-slate-500">共 {page.total} 个 Issue</span>
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
        <div className="hidden min-h-10 grid-cols-[minmax(240px,1fr)_88px_80px_110px_126px_126px_96px] items-center gap-3 bg-slate-50 px-4 text-xs font-semibold uppercase text-slate-500 md:grid">
          <span>标题</span>
          <span>级别</span>
          <span>事件</span>
          <span>环境</span>
          <span>Release</span>
          <span>最后出现</span>
          <span>操作</span>
        </div>
        {filteredIssues.map((issue) => (
          <div key={issue.id} className="grid gap-3 border-t border-slate-100 px-4 py-3 md:grid-cols-[minmax(240px,1fr)_88px_80px_110px_126px_126px_96px] md:items-center">
            <button className="min-w-0 text-left" onClick={() => navigate(`/issues/${issue.id}`)}>
              <div className="truncate font-medium text-slate-900">{issue.title}</div>
              <div className="truncate text-xs text-slate-500">{issue.culprit || issue.fingerprint}</div>
            </button>
            <span className={levelClass(issue.level)}>{issue.level}</span>
            <span className="text-sm text-slate-700">{compactNumber(issue.event_count)}</span>
            <span className="truncate text-sm text-slate-600">{issue.environment || "-"}</span>
            <span className="truncate text-sm text-slate-600">{issue.release || "-"}</span>
            <span className="text-sm text-slate-600">{formatDate(issue.last_seen)}</span>
            <div className="flex gap-2">
              <Button className="h-8 w-8 px-0" onClick={() => void setIssueStatus(issue.id, "resolved")} title="标记 resolved">
                <CheckCircle2 className="h-4 w-4" />
              </Button>
              <Button className="h-8 w-8 px-0" onClick={() => void setIssueStatus(issue.id, "ignored")} title="忽略">
                <CircleOff className="h-4 w-4" />
              </Button>
            </div>
          </div>
        ))}
        {filteredIssues.length === 0 && <Empty label="暂无匹配 Issue" />}
      </div>

      <Pagination page={page} setOffset={setOffset} />
    </section>
  );
}

function statusLabel(status: string) {
  switch (status) {
    case "unresolved":
      return "未解决";
    case "resolved":
      return "已解决";
    case "ignored":
      return "已忽略";
    default:
      return "全部";
  }
}
