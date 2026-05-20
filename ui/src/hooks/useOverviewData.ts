import { getLevels, getTopIssues, getTopReleases, getTrend, listEventsPage, listIssuesPage, type BreakdownItem, type EventItem, type StatsParams, type TrendPoint } from "../services/api";
import { useAsyncData } from "./useAsyncData";

type OverviewData = {
  issueTotal: number;
  eventTotal: number;
  lastEvent?: EventItem;
  trend: TrendPoint[];
  levels: BreakdownItem[];
  topIssues: BreakdownItem[];
  topReleases: BreakdownItem[];
};

const initialOverview: OverviewData = {
  issueTotal: 0,
  eventTotal: 0,
  trend: [],
  levels: [],
  topIssues: [],
  topReleases: [],
};

export function useOverviewData(projectId: string, params: StatsParams = {}, refreshKey = 0) {
  return useAsyncData(
    async () => {
      const [issues, events, trend, levels, topIssues, topReleases] = await Promise.all([
        listIssuesPage(projectId, { status: "unresolved", ...params, limit: 1, offset: 0 }),
        listEventsPage(projectId, { ...params, limit: 1, offset: 0 }),
        getTrend(projectId, params),
        getLevels(projectId, params),
        getTopIssues(projectId, params),
        getTopReleases(projectId, params),
      ]);
      return { issueTotal: issues.page.total, eventTotal: events.page.total, lastEvent: events.items[0], trend, levels, topIssues, topReleases };
    },
    initialOverview,
    [projectId, params.environment, params.release, params.since, params.until, params.limit, refreshKey],
  );
}
