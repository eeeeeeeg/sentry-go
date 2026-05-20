import {
  createProject,
  createProjectKey,
  listProjectKeysPage,
  listProjectsPage,
  updateProject,
  updateProjectKey,
  updateProjectKeyStatus,
  updateProjectStatus,
  type PageMeta,
  type Project,
  type ProjectKey,
} from "../services/api";
import { useAsyncData } from "./useAsyncData";

type ProjectsState = {
  projects: Project[];
  keys: ProjectKey[];
  current?: Project;
  projectsPage: PageMeta;
  keysPage: PageMeta;
};

function numberField(form: FormData, key: string, fallback: number) {
  const value = form.get(key);
  if (value === null || value === "") {
    return fallback;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

export function useProjects(selectedProjectId: string, refreshKey = 0, projectOffset = 0, keyOffset = 0, pageSize = 10) {
  const state = useAsyncData<ProjectsState>(
    async () => {
      const projectResult = await listProjectsPage({ limit: pageSize, offset: projectOffset });
      const projects = projectResult.items;
      const current = projects.find((item) => item.slug === selectedProjectId || item.id === selectedProjectId) ?? projects[0];
      const keyResult = current ? await listProjectKeysPage(current.slug, { limit: pageSize, offset: keyOffset }) : { items: [], page: { limit: pageSize, offset: 0, total: 0 } };
      return { projects, keys: keyResult.items, current, projectsPage: projectResult.page, keysPage: keyResult.page };
    },
    {
      projects: [],
      keys: [],
      current: undefined,
      projectsPage: { limit: pageSize, offset: 0, total: 0 },
      keysPage: { limit: pageSize, offset: 0, total: 0 },
    },
    [selectedProjectId, refreshKey, projectOffset, keyOffset, pageSize],
  );

  async function addProject(form: FormData) {
    await createProject({
      organization_slug: String(form.get("organization_slug") || "demo"),
      slug: String(form.get("slug") ?? ""),
      name: String(form.get("name") ?? ""),
      platform: String(form.get("platform") || "javascript"),
      sample_rate: numberField(form, "sample_rate", 1),
    });
    await state.reload();
  }

  async function saveProject(projectRef: string, form: FormData) {
    await updateProject(projectRef, {
      name: String(form.get("name") ?? ""),
      platform: String(form.get("platform") || "javascript"),
      sample_rate: numberField(form, "sample_rate", 1),
    });
    await state.reload();
  }

  async function setProjectStatus(projectRef: string, status: string) {
    await updateProjectStatus(projectRef, status);
    await state.reload();
  }

  async function addKey(projectRef: string, form: FormData) {
    await createProjectKey(projectRef, {
      name: String(form.get("name") || "Default Key"),
      rate_limit_per_minute: numberField(form, "rate_limit_per_minute", 6000),
    });
    await state.reload();
  }

  async function saveKey(keyId: string, form: FormData) {
    await updateProjectKey(keyId, {
      name: String(form.get("name") ?? ""),
      rate_limit_per_minute: numberField(form, "rate_limit_per_minute", 6000),
    });
    await state.reload();
  }

  async function setKeyStatus(keyId: string, status: string) {
    await updateProjectKeyStatus(keyId, status);
    await state.reload();
  }

  return { ...state, addProject, saveProject, setProjectStatus, addKey, saveKey, setKeyStatus };
}
