import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { X } from "lucide-react";
import { EventItem } from "../services/api";
import { formatDate, prettyJSON } from "../utils/format";

export function EventDialog({ event, close }: { event: EventItem | null; close: () => void }) {
  const raw = parseJSON(event?.raw_event);
  const rawException = raw?.exception;
  const stacktrace = rawException && typeof rawException === "object" && "stacktrace" in rawException ? rawException.stacktrace : undefined;

  return (
    <Dialog open={Boolean(event)} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="max-h-[86vh] w-full max-w-5xl overflow-auto rounded-lg bg-white p-5 shadow-xl">
          {event && (
            <>
              <div className="mb-4 flex items-center justify-between gap-4">
                <DialogTitle className="truncate text-lg font-semibold text-slate-900">{event.exception_type || "Event Detail"}</DialogTitle>
                <button className="btn h-8 w-8 px-0" onClick={close} title="关闭">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="grid gap-4">
                <section className="grid gap-3 rounded-md border border-slate-200 p-4 md:grid-cols-3">
                  <Detail label="Event ID" value={event.event_id} />
                  <Detail label="Issue ID" value={event.issue_id || "-"} />
                  <Detail label="Level" value={event.level} />
                  <Detail label="Environment" value={event.environment || "-"} />
                  <Detail label="Release" value={event.release || "-"} />
                  <Detail label="Timestamp" value={formatDate(event.timestamp)} />
                  <Detail label="Platform" value={event.platform || "-"} />
                  <Detail label="User ID" value={event.user_id || "-"} />
                  <Detail label="Received" value={formatDate(event.received_at)} />
                </section>

                <section className="rounded-md border border-slate-200 p-4">
                  <div className="mb-1 text-xs font-semibold text-slate-500">Message</div>
                  <div className="break-words text-sm text-slate-900">{event.message || event.exception_value || "-"}</div>
                </section>

                <div className="grid gap-4 xl:grid-cols-2">
                  <JSONPanel title="Exception" value={rawException ?? { type: event.exception_type, value: event.exception_value }} />
                  <JSONPanel title="Stacktrace" value={stacktrace ?? "-"} />
                  <JSONPanel title="Tags" value={event.tags} />
                  <JSONPanel title="Contexts" value={event.contexts} />
                </div>

                <JSONPanel title="Raw Event" value={event.raw_event} tall />
              </div>
            </>
          )}
        </DialogPanel>
      </div>
    </Dialog>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="mb-1 text-xs font-semibold text-slate-500">{label}</div>
      <div className="break-all text-sm text-slate-800">{value}</div>
    </div>
  );
}

function JSONPanel({ title, value, tall }: { title: string; value: unknown; tall?: boolean }) {
  return (
    <section className="rounded-md border border-slate-200">
      <div className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-slate-800">{title}</div>
      <pre className={`${tall ? "max-h-[520px]" : "max-h-72"} overflow-auto rounded-b-md bg-slate-950 p-3 text-xs leading-6 text-slate-100`}>{formatPanelValue(value)}</pre>
    </section>
  );
}

function formatPanelValue(value: unknown) {
  if (typeof value === "string") {
    return prettyJSON(value);
  }
  if (value === undefined || value === null) {
    return "-";
  }
  return JSON.stringify(value, null, 2);
}

function parseJSON(raw?: string) {
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw) as Record<string, unknown>;
  } catch {
    return undefined;
  }
}
