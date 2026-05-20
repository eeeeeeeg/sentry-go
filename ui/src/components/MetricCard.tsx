import { LucideIcon } from "lucide-react";

export function MetricCard({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return (
    <div className="card grid min-h-24 grid-cols-[24px_minmax(0,1fr)] content-center gap-x-3 gap-y-1 p-4">
      <Icon className="h-5 w-5 text-blue-700" />
      <span className="text-sm text-slate-500">{label}</span>
      <strong className="col-span-2 truncate text-2xl text-slate-950">{value}</strong>
    </div>
  );
}
