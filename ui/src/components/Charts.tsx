import { BreakdownItem, TrendPoint } from "../services/api";
import { formatDate } from "../utils/format";
import { Empty } from "./Empty";

export function Sparkline({ data }: { data: TrendPoint[] }) {
  const max = Math.max(...data.map((item) => item.count), 1);
  if (data.length === 0) return <Empty label="暂无趋势数据" />;
  return (
    <div className="flex h-44 items-end gap-1.5">
      {data.map((item) => (
        <div
          key={item.bucket}
          className="min-w-1 flex-1 rounded-t bg-gradient-to-b from-blue-700 to-emerald-600"
          title={`${formatDate(item.bucket)} ${item.count}`}
          style={{ height: `${Math.max(8, (item.count / max) * 100)}%` }}
        />
      ))}
    </div>
  );
}

export function Bars({ data }: { data: BreakdownItem[] }) {
  const max = Math.max(...data.map((item) => item.count), 1);
  if (data.length === 0) return <Empty label="暂无分布数据" />;
  return (
    <div className="grid gap-3">
      {data.map((item) => (
        <div className="grid grid-cols-[92px_minmax(0,1fr)_42px] items-center gap-3 text-sm" key={item.key}>
          <span className="truncate text-slate-600">{item.key || "-"}</span>
          <div className="h-2.5 overflow-hidden rounded-full bg-slate-100">
            <i className="block h-full rounded-full bg-blue-700" style={{ width: `${(item.count / max) * 100}%` }} />
          </div>
          <strong className="text-right text-slate-700">{item.count}</strong>
        </div>
      ))}
    </div>
  );
}

export function RankList({ data, empty }: { data: BreakdownItem[]; empty: string }) {
  if (data.length === 0) return <Empty label={empty} />;
  return (
    <div className="grid gap-2">
      {data.map((item, index) => (
        <div className="grid grid-cols-[28px_minmax(0,1fr)_56px] items-center gap-3 rounded-md border border-slate-100 p-2" key={item.key || index}>
          <span className="grid h-6 w-6 place-items-center rounded-full bg-slate-100 text-xs text-slate-600">{index + 1}</span>
          <strong className="truncate text-sm text-slate-800">{item.key || "-"}</strong>
          <em className="text-right text-sm not-italic text-slate-500">{item.count}</em>
        </div>
      ))}
    </div>
  );
}
