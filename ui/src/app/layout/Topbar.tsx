import clsx from "clsx";
import { RefreshCw, Search } from "lucide-react";
import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { listProjects, type Project } from "../../services/api";

export function Topbar({
  projectId,
  setProjectId,
  query,
  setQuery,
  loading,
  refresh,
}: {
  projectId: string;
  setProjectId: (projectId: string) => void;
  query: string;
  setQuery: (query: string) => void;
  loading: boolean;
  refresh: () => void;
}) {
  const location = useLocation();
  const searchDisabled = location.pathname === "/system";
  const [projects, setProjects] = useState<Project[]>([]);

  useEffect(() => {
    let active = true;
    listProjects()
      .then((items) => {
        if (active) {
          setProjects(items);
        }
      })
      .catch(() => {
        if (active) {
          setProjects([]);
        }
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <header className="sticky top-[56px] z-10 grid gap-3 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur lg:top-0 lg:grid-cols-[220px_minmax(220px,1fr)_40px] lg:px-6">
      <label className="grid h-10 grid-cols-[52px_minmax(0,1fr)] overflow-hidden rounded-md border border-slate-300 bg-white text-sm">
        <span className="grid place-items-center border-r border-slate-200 text-xs font-medium text-slate-500">项目</span>
        {projects.length > 0 ? (
          <select className="min-w-0 px-3 outline-none" value={projectId} onChange={(event) => setProjectId(event.target.value)}>
            {projects.map((project) => (
              <option key={project.id} value={project.sentry_project_id}>
                {project.name} / {project.sentry_project_id}
              </option>
            ))}
          </select>
        ) : (
          <input className="min-w-0 px-3 outline-none" value={projectId} onChange={(event) => setProjectId(event.target.value.trim() || "1")} />
        )}
      </label>
      <label className={clsx("flex h-10 items-center gap-2 rounded-md border border-slate-300 bg-white px-3", searchDisabled && "opacity-50")}>
        <Search className="h-4 w-4 text-slate-500" />
        <input className="min-w-0 flex-1 text-sm outline-none" placeholder="搜索当前列表" value={query} disabled={searchDisabled} onChange={(event) => setQuery(event.target.value)} />
      </label>
      <button className="btn h-10 px-0" onClick={refresh} disabled={loading} title="刷新">
        <RefreshCw className={clsx("h-4 w-4", loading && "animate-spin")} />
      </button>
    </header>
  );
}
