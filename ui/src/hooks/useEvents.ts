import { getEvent, listEventsPage, type EventItem, type EventListParams, type Paginated } from "../services/api";
import { useAsyncData } from "./useAsyncData";

const initialEvents: Paginated<EventItem> = {
  items: [],
  page: { limit: 20, offset: 0, total: 0 },
};

export function useEvents(projectId: string, params: EventListParams, refreshKey = 0) {
  const state = useAsyncData(() => listEventsPage(projectId, params), initialEvents, [
    projectId,
    params.issue_id,
    params.level,
    params.environment,
    params.release,
    params.since,
    params.until,
    params.limit,
    params.offset,
    refreshKey,
  ]);

  async function loadEvent(eventId: string) {
    return getEvent(eventId);
  }

  return { ...state, loadEvent };
}
