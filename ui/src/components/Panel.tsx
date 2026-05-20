import { ReactNode } from "react";

export function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="card min-w-0 p-4">
      <h2 className="mb-4 text-sm font-semibold text-slate-800">{title}</h2>
      {children}
    </section>
  );
}
