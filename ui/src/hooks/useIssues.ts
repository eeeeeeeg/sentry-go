import { listIssuesPage, updateIssueStatus, type IssueListParams, type Paginated, type Issue } from "../services/api";
import { useAsyncData } from "./useAsyncData";

const initialIssues: Paginated<Issue> = {
  items: [],
  page: { limit: 20, offset: 0, total: 0 },
};

export function useIssues(projectId: string, params: IssueListParams, refreshKey = 0) {
  const state = useAsyncData(() => listIssuesPage(projectId, params), initialIssues, [
    projectId,
    params.status,
    params.level,
    params.environment,
    params.release,
    params.since,
    params.until,
    params.limit,
    params.offset,
    refreshKey,
  ]);

  async function setIssueStatus(issueId: string, nextStatus: string) {
    await updateIssueStatus(issueId, nextStatus);
    await state.reload();
  }

  return { ...state, setIssueStatus };
}
