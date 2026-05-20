import { Dialog, DialogPanel, DialogTitle, Switch } from "@headlessui/react";
import clsx from "clsx";
import { Send, Webhook, X } from "lucide-react";
import { ReactNode, useEffect, useState } from "react";
import { Empty } from "../../components/Empty";
import { Pagination } from "../../components/Pagination";
import { Panel } from "../../components/Panel";
import { useAlerts } from "../../hooks/useAlerts";
import { type AlertDelivery } from "../../services/api";
import { formatDate } from "../../utils/format";

const deliveryPageSize = 20;

export function AlertsPage({
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
  const [deliveryOffset, setDeliveryOffset] = useState(0);
  const [deliveryStatus, setDeliveryStatus] = useState("");
  const [selectedDelivery, setSelectedDelivery] = useState<AlertDelivery | null>(null);
  const [testMessage, setTestMessage] = useState("");
  const { data, loading, error, createAlert, setAlertStatus, sendTestAlert } = useAlerts(projectId, refreshKey, deliveryOffset, deliveryPageSize, deliveryStatus);
  const { alerts, deliveries } = data;

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);
  useEffect(() => setDeliveryOffset(0), [projectId]);
  useEffect(() => setDeliveryOffset(0), [deliveryStatus]);

  async function testRule(alertId: string) {
    setTestMessage("");
    try {
      await sendTestAlert(alertId);
      setTestMessage("测试 webhook 已发送");
    } catch (err) {
      setTestMessage(err instanceof Error ? err.message : "测试发送失败");
    }
  }

  return (
    <section className="grid gap-4 p-4 xl:grid-cols-[minmax(420px,1fr)_minmax(420px,0.9fr)] lg:p-6">
      <Panel title="Webhook 规则">
        <form
          className="mb-4 grid gap-3 md:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            void createAlert(new FormData(event.currentTarget));
            event.currentTarget.reset();
          }}
        >
          <input className="field" name="name" placeholder="规则名称" required />
          <input className="field md:col-span-2" name="webhook_url" placeholder="Webhook URL" required />
          <select className="field" name="event_type" defaultValue="new_issue">
            <option value="new_issue">new_issue</option>
            <option value="regression">regression</option>
            <option value="frequency">frequency</option>
          </select>
          <select className="field" name="min_level" defaultValue="error">
            <option value="warning">warning</option>
            <option value="error">error</option>
            <option value="fatal">fatal</option>
          </select>
          <input className="field" name="threshold_count" type="number" min="1" defaultValue="1" />
          <input className="field" name="window_seconds" type="number" min="1" defaultValue="300" />
          <input className="field" name="cooldown_seconds" type="number" min="0" defaultValue="300" />
          <button className="btn-primary md:col-span-2">
            <Webhook className="h-4 w-4" />
            创建规则
          </button>
        </form>

        {testMessage && <div className="mb-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">{testMessage}</div>}

        <div className="grid gap-2">
          {alerts.map((alert) => (
            <div key={alert.id} className="grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 rounded-md border border-slate-100 p-3">
              <div className="min-w-0">
                <div className="truncate font-medium text-slate-900">{alert.name}</div>
                <div className="truncate text-xs text-slate-500">
                  {alert.event_type} / {alert.min_level} / {alert.cooldown_seconds}s
                </div>
              </div>
              <button className="btn h-8 w-8 px-0" onClick={() => void testRule(alert.id)} title="测试发送">
                <Send className="h-4 w-4" />
              </button>
              <Switch
                checked={alert.status === "active"}
                onChange={(checked) => void setAlertStatus(alert.id, checked ? "active" : "disabled")}
                className={clsx("relative inline-flex h-6 w-11 items-center rounded-full transition", alert.status === "active" ? "bg-emerald-600" : "bg-slate-300")}
              >
                <span className={clsx("inline-block h-5 w-5 rounded-full bg-white transition", alert.status === "active" ? "translate-x-5" : "translate-x-1")} />
              </Switch>
            </div>
          ))}
          {alerts.length === 0 && <Empty label="暂无告警规则" />}
        </div>
      </Panel>

      <Panel title="投递记录">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <select className="field w-40" value={deliveryStatus} onChange={(event) => setDeliveryStatus(event.target.value)}>
            <option value="">全部状态</option>
            <option value="sent">sent</option>
            <option value="failed">failed</option>
            <option value="suppressed">suppressed</option>
          </select>
          <span className="text-sm text-slate-500">共 {deliveries.page.total} 条记录</span>
        </div>
        <div className="grid gap-2">
          {deliveries.items.map((delivery) => (
            <button
              key={delivery.id}
              className="grid grid-cols-[10px_minmax(0,1fr)_auto] items-center gap-3 rounded-md border border-slate-100 p-3 text-left hover:bg-slate-50"
              onClick={() => setSelectedDelivery(delivery)}
            >
              <span className={clsx("h-2 w-2 rounded-full", delivery.status === "sent" ? "bg-emerald-500" : "bg-amber-500")} />
              <div className="min-w-0">
                <div className="truncate font-medium text-slate-900">
                  {delivery.event_type} / {delivery.status}
                </div>
                <div className="truncate text-xs text-slate-500">
                  {formatDate(delivery.created_at)} / {delivery.error || delivery.event_id}
                </div>
              </div>
              <span className="text-xs font-medium text-slate-500">详情</span>
            </button>
          ))}
          {deliveries.items.length === 0 && <Empty label="暂无投递记录" />}
        </div>
        <Pagination page={deliveries.page} setOffset={setDeliveryOffset} />
      </Panel>

      <DeliveryDialog delivery={selectedDelivery} close={() => setSelectedDelivery(null)} />
    </section>
  );
}

function DeliveryDialog({ delivery, close }: { delivery: AlertDelivery | null; close: () => void }) {
  return (
    <Dialog open={Boolean(delivery)} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="w-full max-w-2xl rounded-lg bg-white p-5 shadow-xl">
          <div className="mb-4 flex items-center justify-between gap-4">
            <DialogTitle className="text-lg font-semibold text-slate-900">投递详情</DialogTitle>
            <button className="btn h-8 w-8 px-0" onClick={close} title="关闭">
              <X className="h-4 w-4" />
            </button>
          </div>
          {delivery ? (
            <div className="grid gap-3">
              <Detail label="状态" value={delivery.status} />
              <Detail label="事件类型" value={delivery.event_type} />
              <Detail label="通道" value={delivery.channel} />
              <Detail label="Issue ID" value={delivery.issue_id} />
              <Detail label="Event ID" value={delivery.event_id} />
              <Detail label="错误" value={delivery.error || "-"} />
              <Detail label="创建时间" value={formatDate(delivery.created_at)} />
              <Detail label="投递时间" value={formatDate(delivery.delivered_at)} />
            </div>
          ) : null}
        </DialogPanel>
      </div>
    </Dialog>
  );
}

function Detail({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="grid gap-1 rounded-md border border-slate-100 p-3">
      <span className="text-xs font-semibold text-slate-500">{label}</span>
      <span className="break-all text-sm text-slate-900">{value}</span>
    </div>
  );
}
