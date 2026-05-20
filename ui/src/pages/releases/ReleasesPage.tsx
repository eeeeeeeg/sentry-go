import { ArrowRight, GitBranch } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Empty } from "../../components/Empty";
import { QueryFilters, type QueryFilterValues } from "../../components/QueryFilters";
import { Button } from "../../components/ui";
import { useReleases } from "../../hooks/useReleases";
import { compactNumber, datetimeLocalToRFC3339, formatDate } from "../../utils/format";

const emptyFilters: QueryFilterValues = {
  level: "",
  environment: "",
  release: "",
  since: "",
  until: "",
};

export function ReleasesPage({
  projectId,
  query,
  refreshKey,
  onLoadingChange,
  onError,
}: {
  projectId: string;
  query: string;
  refreshKey: number;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string) => void;
}) {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<QueryFilterValues>(emptyFilters);
  const params = {
    environment: filters.environment.trim() || undefined,
    release: filters.release.trim() || undefined,
    since: datetimeLocalToRFC3339(filters.since),
    until: datetimeLocalToRFC3339(filters.until),
    limit: 50,
  };
  const { data: releases, loading, error } = useReleases(projectId, params, refreshKey);
  const filteredReleases = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return releases;
    }
    return releases.filter((item) => `${item.release} ${item.environment ?? ""}`.toLowerCase().includes(needle));
  }, [query, releases]);

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);

  function openReleaseIssues(release: string, environment?: string) {
    const search = new URLSearchParams({ release });
    if (environment) {
      search.set("environment", environment);
    }
    navigate(`/issues?${search.toString()}`);
  }

  return (
    <section className="grid gap-4 p-4 lg:p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold text-slate-900">Releases</h1>
          <p className="mt-1 text-sm text-slate-500">按版本查看错误影响范围，快速定位新版本引入的问题。</p>
        </div>
        <span className="text-sm text-slate-500">共 {filteredReleases.length} 个版本</span>
      </div>

      <QueryFilters
        title="Release 过滤"
        values={filters}
        showLevel={false}
        onChange={setFilters}
        onReset={() => setFilters(emptyFilters)}
      />

      <div className="grid gap-3">
        {filteredReleases.map((item) => (
          <section key={item.release} className="card grid gap-4 p-4 lg:grid-cols-[minmax(220px,1fr)_repeat(4,120px)_auto] lg:items-center">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <GitBranch className="h-4 w-4 text-slate-500" />
                <h2 className="truncate font-semibold text-slate-900">{item.release}</h2>
              </div>
              <p className="mt-1 truncate text-sm text-slate-500">{item.environment || "all environments"}</p>
            </div>
            <Metric label="事件" value={compactNumber(item.event_count)} />
            <Metric label="Issue" value={compactNumber(item.issue_count)} />
            <Metric label="用户" value={compactNumber(item.user_count)} />
            <Metric label="最近出现" value={formatDate(item.last_seen)} />
            <Button onClick={() => openReleaseIssues(item.release, item.environment)} className="justify-self-start lg:justify-self-end">
              查看 Issue
              <ArrowRight className="h-4 w-4" />
            </Button>
          </section>
        ))}
        {filteredReleases.length === 0 && <Empty label="暂无 Release 数据" />}
      </div>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-slate-500">{label}</div>
      <div className="mt-1 text-sm font-semibold text-slate-900">{value}</div>
    </div>
  );
}
