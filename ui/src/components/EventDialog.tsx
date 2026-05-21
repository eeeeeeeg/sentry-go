import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { X } from "lucide-react";
import { EventItem } from "../services/api";
import { EventDetails } from "./EventDetails";

export function EventDialog({ event, close }: { event: EventItem | null; close: () => void }) {
  return (
    <Dialog open={Boolean(event)} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="max-h-[86vh] w-full max-w-6xl overflow-auto rounded-lg bg-white p-5 shadow-xl">
          {event && (
            <>
              <div className="mb-4 flex items-center justify-between gap-4">
                <DialogTitle className="truncate text-lg font-semibold text-slate-900">{event.exception_type || "Event Detail"}</DialogTitle>
                <button className="btn h-8 w-8 px-0" onClick={close} title="Close">
                  <X className="h-4 w-4" />
                </button>
              </div>
              <EventDetails event={event} />
            </>
          )}
        </DialogPanel>
      </div>
    </Dialog>
  );
}
