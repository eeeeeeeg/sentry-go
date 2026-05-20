import { RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { Panel } from "../../components/Panel";
import { getHealth, getMetrics, getReady } from "../../services/api";

export function SystemPage() {
  const [health, setHealth] = useState<Record<string, unknown> | null>(null);
  const [ready, setReady] = useState<Record<string, unknown> | null>(null);
  const [metrics, setMetrics] = useState("");
  const [error, setError] = useState("");

  async function load() {
    setError("");
    try {
      const [healthPayload, readyPayload, metricsPayload] = await Promise.all([getHealth(), getReady(), getMetrics()]);
      setHealth(healthPayload);
      setReady(readyPayload);
      setMetrics(metricsPayload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "系统状态加载失败");
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <section className="grid gap-4 p-4 lg:p-6">
      {error && <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">{error}</div>}
      <div className="flex justify-end">
        <button className="btn" onClick={() => void load()}>
          <RefreshCw className="h-4 w-4" />
          刷新状态
        </button>
      </div>
      <div className="grid gap-4 xl:grid-cols-2">
        <Panel title="Health">
          <pre className="max-h-96 overflow-auto rounded-md bg-slate-950 p-3 text-xs leading-6 text-slate-100">{JSON.stringify(health, null, 2)}</pre>
        </Panel>
        <Panel title="Readiness">
          <pre className="max-h-96 overflow-auto rounded-md bg-slate-950 p-3 text-xs leading-6 text-slate-100">{JSON.stringify(ready, null, 2)}</pre>
        </Panel>
      </div>
      <Panel title="Metrics">
        <pre className="max-h-[520px] overflow-auto rounded-md bg-slate-950 p-3 text-xs leading-6 text-slate-100">{metrics || "-"}</pre>
      </Panel>
    </section>
  );
}
