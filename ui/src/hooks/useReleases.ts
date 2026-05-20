import { listReleases, type ReleaseSummary, type StatsParams } from "../services/api";
import { useAsyncData } from "./useAsyncData";

const initialReleases: ReleaseSummary[] = [];

export function useReleases(projectId: string, params: StatsParams, refreshKey = 0) {
  return useAsyncData(() => listReleases(projectId, params), initialReleases, [
    projectId,
    params.environment,
    params.release,
    params.since,
    params.until,
    params.limit,
    refreshKey,
  ]);
}
