import { type PageMeta } from "../services/api";
import { Button } from "./ui";

export function Pagination({ page, setOffset }: { page: PageMeta; setOffset: (offset: number) => void }) {
  const start = page.total === 0 ? 0 : page.offset + 1;
  const end = Math.min(page.offset + page.limit, page.total);
  const hasPrev = page.offset > 0;
  const hasNext = page.offset + page.limit < page.total;

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-3">
      <span className="text-sm text-slate-500">
        {start}-{end} / {page.total}
      </span>
      <div className="flex gap-2">
        <Button disabled={!hasPrev} onClick={() => setOffset(Math.max(0, page.offset - page.limit))}>
          上一页
        </Button>
        <Button disabled={!hasNext} onClick={() => setOffset(page.offset + page.limit)}>
          下一页
        </Button>
      </div>
    </div>
  );
}
