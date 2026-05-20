import { Filter, RotateCcw } from "lucide-react";
import { Button, Card, SelectInput, TextInput } from "./ui";

export type QueryFilterValues = {
  level?: string;
  environment: string;
  release: string;
  since: string;
  until: string;
};

export function QueryFilters({
  title = "查询条件",
  values,
  showLevel = true,
  onChange,
  onReset,
}: {
  title?: string;
  values: QueryFilterValues;
  showLevel?: boolean;
  onChange: (values: QueryFilterValues) => void;
  onReset: () => void;
}) {
  function patch(next: Partial<QueryFilterValues>) {
    onChange({ ...values, ...next });
  }

  return (
    <Card className="p-4">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-slate-800">
        <Filter className="h-4 w-4" />
        {title}
      </div>
      <div className={`grid gap-3 md:grid-cols-3 ${showLevel ? "xl:grid-cols-6" : "xl:grid-cols-5"}`}>
        {showLevel && (
          <SelectInput value={values.level ?? ""} onChange={(event) => patch({ level: event.target.value })}>
            <option value="">全部级别</option>
            <option value="fatal">fatal</option>
            <option value="error">error</option>
            <option value="warning">warning</option>
            <option value="info">info</option>
          </SelectInput>
        )}
        <TextInput value={values.environment} onChange={(event) => patch({ environment: event.target.value })} placeholder="环境" />
        <TextInput value={values.release} onChange={(event) => patch({ release: event.target.value })} placeholder="版本" />
        <TextInput type="datetime-local" value={values.since} onChange={(event) => patch({ since: event.target.value })} />
        <TextInput type="datetime-local" value={values.until} onChange={(event) => patch({ until: event.target.value })} />
        <Button onClick={onReset}>
          <RotateCcw className="h-4 w-4" />
          重置
        </Button>
      </div>
    </Card>
  );
}
