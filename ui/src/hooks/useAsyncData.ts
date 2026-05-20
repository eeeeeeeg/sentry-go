import { Dispatch, SetStateAction, useCallback, useEffect, useState } from "react";

export type AsyncDataState<T> = {
  data: T;
  loading: boolean;
  error: string;
  reload: () => Promise<void>;
  setData: Dispatch<SetStateAction<T>>;
};

export function useAsyncData<T>(load: () => Promise<T>, initialData: T, deps: readonly unknown[]): AsyncDataState<T> {
  const [data, setData] = useState<T>(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setData(await load());
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, deps);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { data, loading, error, reload, setData };
}
