import { Activity, AlertTriangle, Bug, Clock3 } from "lucide-react";
import { useEffect, useState } from "react";
import { Bars, RankList, Sparkline } from "../../components/Charts";
import { MetricCard } from "../../components/MetricCard";
import { Panel } from "../../components/Panel";
import { QueryFilters, type QueryFilterValues } from "../../components/QueryFilters";
import { useOverviewData } from "../../hooks/useOverviewData";
import { compactNumber, datetimeLocalToRFC3339, formatDate } from "../../utils/format";

const emptyFilters: QueryFilterValues = {
  environment: "",
  release: "",
  since: "",
  until: "",
};

export function OverviewPage({
  projectId,
  refreshKey,
  onLoadingChange,
  onError,
}: {
  projectId: string;
  refreshKey: number;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string) => void;
}) {
  const [filters, setFilters] = useState<QueryFilterValues>(emptyFilters);
  const statsParams = {
    environment: filters.environment.trim() || undefined,
    release: filters.release.trim() || undefined,
    since: datetimeLocalToRFC3339(filters.since),
    until: datetimeLocalToRFC3339(filters.until),
    limit: 10,
  };
  const { data, loading, error } = useOverviewData(projectId, statsParams, refreshKey);
  const { issueTotal, eventTotal, lastEvent, trend, levels, topIssues, topReleases } = data;

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);

  return (
    <section className="grid gap-5 p-4 lg:p-6">
      <QueryFilters title="统计筛选" values={filters} showLevel={false} onChange={setFilters} onReset={() => setFilters(emptyFilters)} />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard icon={Bug} label="未解决 Issue" value={compactNumber(issueTotal)} />
        <MetricCard icon={Activity} label="最近事件" value={compactNumber(eventTotal)} />
        <MetricCard icon={AlertTriangle} label="主要级别" value={levels[0]?.key ?? "-"} />
        <MetricCard icon={Clock3} label="最后上报" value={formatDate(lastEvent?.timestamp)} />
      </div>
      <div className="grid gap-4 xl:grid-cols-2">
        <Panel title="错误趋势">
          <Sparkline data={trend} />
        </Panel>
        <Panel title="Level 分布">
          <Bars data={levels} />
        </Panel>
        <Panel title="Top Issue">
          <RankList data={topIssues} empty="暂无 Issue 聚合数据" />
        </Panel>
        <Panel title="Top Release">
          <RankList data={topReleases} empty="暂无 Release 数据" />
        </Panel>
      </div>
    </section>
  );
}
