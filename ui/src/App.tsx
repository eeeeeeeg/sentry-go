import { useCallback, useState } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { EventDialog } from "./components/EventDialog";
import { Sidebar } from "./app/layout/Sidebar";
import { Topbar } from "./app/layout/Topbar";
import { AlertsPage } from "./pages/alerts/AlertsPage";
import { EventsPage } from "./pages/events/EventsPage";
import { IssuesPage } from "./pages/issues/IssuesPage";
import { OverviewPage } from "./pages/overview/OverviewPage";
import { ProjectsPage } from "./pages/projects/ProjectsPage";
import { ReleasesPage } from "./pages/releases/ReleasesPage";
import { SystemPage } from "./pages/system/SystemPage";
import { EventItem } from "./services/api";

export function App() {
  const [projectId, setProjectId] = useState("web");
  const [status, setStatus] = useState("unresolved");
  const [selectedEvent, setSelectedEvent] = useState<EventItem | null>(null);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  const handleLoadingChange = useCallback((nextLoading: boolean) => setLoading(nextLoading), []);
  const handleError = useCallback((nextError: string) => setError(nextError), []);
  const refresh = useCallback(() => setRefreshKey((value) => value + 1), []);

  return (
    <div className="min-h-screen bg-slate-100 lg:grid lg:grid-cols-[248px_minmax(0,1fr)]">
      <Sidebar />
      <main className="min-w-0">
        <Topbar projectId={projectId} setProjectId={setProjectId} query={query} setQuery={setQuery} loading={loading} refresh={refresh} />

        {error && <div className="mx-4 mt-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 lg:mx-6">{error}</div>}

        <Routes>
          <Route path="/" element={<Navigate to="/overview" replace />} />
          <Route path="/overview" element={<OverviewPage projectId={projectId} refreshKey={refreshKey} onLoadingChange={handleLoadingChange} onError={handleError} />} />
          <Route
            path="/issues"
            element={
              <IssuesPage
                projectId={projectId}
                query={query}
                status={status}
                setStatus={setStatus}
                refreshKey={refreshKey}
                onLoadingChange={handleLoadingChange}
                onError={handleError}
                openEvent={setSelectedEvent}
              />
            }
          />
          <Route
            path="/events"
            element={
              <EventsPage
                projectId={projectId}
                query={query}
                refreshKey={refreshKey}
                openEvent={setSelectedEvent}
                onLoadingChange={handleLoadingChange}
                onError={handleError}
              />
            }
          />
          <Route
            path="/releases"
            element={
              <ReleasesPage
                projectId={projectId}
                query={query}
                refreshKey={refreshKey}
                onLoadingChange={handleLoadingChange}
                onError={handleError}
              />
            }
          />
          <Route
            path="/alerts"
            element={
              <AlertsPage
                projectId={projectId}
                refreshKey={refreshKey}
                onLoadingChange={handleLoadingChange}
                onError={handleError}
              />
            }
          />
          <Route
            path="/projects"
            element={
              <ProjectsPage
                projectId={projectId}
                setProjectId={setProjectId}
                refreshKey={refreshKey}
                onLoadingChange={handleLoadingChange}
                onError={handleError}
              />
            }
          />
          <Route path="/system" element={<SystemPage />} />
          <Route path="*" element={<Navigate to="/overview" replace />} />
        </Routes>
      </main>
      <EventDialog event={selectedEvent} close={() => setSelectedEvent(null)} />
    </div>
  );
}
