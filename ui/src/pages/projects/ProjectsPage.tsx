import { Dialog, DialogPanel, DialogTitle, Switch } from "@headlessui/react";
import clsx from "clsx";
import { BookOpen, Check, Copy, Edit3, KeyRound, Plus, Save, Search, Send, Settings2, X } from "lucide-react";
import { FormEvent, ReactNode, useEffect, useMemo, useState } from "react";
import { Empty } from "../../components/Empty";
import { Panel } from "../../components/Panel";
import { useProjects } from "../../hooks/useProjects";
import { sendTestEvent, type PageMeta, type Project, type ProjectKey } from "../../services/api";
import { formatDate } from "../../utils/format";

const pageSize = 8;

export function ProjectsPage({
  projectId,
  setProjectId,
  refreshKey,
  onLoadingChange,
  onError,
}: {
  projectId: string;
  setProjectId: (projectId: string) => void;
  refreshKey: number;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string) => void;
}) {
  const [projectOffset, setProjectOffset] = useState(0);
  const [keyOffset, setKeyOffset] = useState(0);
  const [projectModal, setProjectModal] = useState<"create" | "edit" | null>(null);
  const [keyModal, setKeyModal] = useState<"create" | ProjectKey | null>(null);
  const [keyDrawerOpen, setKeyDrawerOpen] = useState(false);
  const [onboardingOpen, setOnboardingOpen] = useState(false);
  const [projectQuery, setProjectQuery] = useState("");
  const [copiedKey, setCopiedKey] = useState("");
  const { data, loading, error, addProject, saveProject, setProjectStatus, addKey, saveKey, setKeyStatus } = useProjects(projectId, refreshKey, projectOffset, keyOffset, pageSize);
  const { projects, keys, current, projectsPage, keysPage } = data;

  const filteredProjects = useMemo(() => {
    const keyword = projectQuery.trim().toLowerCase();
    if (!keyword) {
      return projects;
    }
    return projects.filter((project) => `${project.name} ${project.slug} ${project.platform}`.toLowerCase().includes(keyword));
  }, [projectQuery, projects]);

  useEffect(() => onLoadingChange(loading), [loading, onLoadingChange]);
  useEffect(() => onError(error), [error, onError]);
  useEffect(() => setKeyOffset(0), [current?.id]);

  async function copyKey(publicKey: string) {
    await navigator.clipboard.writeText(publicKey);
    setCopiedKey(publicKey);
    window.setTimeout(() => setCopiedKey(""), 1200);
  }

  async function submitProject(form: FormData) {
    if (projectModal === "edit" && current) {
      await saveProject(current.slug, form);
    } else {
      await addProject(form);
    }
    setProjectModal(null);
  }

  async function submitKey(form: FormData) {
    if (!current) {
      return;
    }
    if (keyModal && keyModal !== "create") {
      await saveKey(keyModal.id, form);
    } else {
      await addKey(current.slug, form);
    }
    setKeyModal(null);
  }

  return (
    <section className="grid gap-4 p-4 lg:p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-slate-950">项目管理</h1>
          <p className="mt-1 text-sm text-slate-500">管理项目基础信息、启停状态和上报 Key。</p>
        </div>
        <button className="btn-primary" onClick={() => setProjectModal("create")}>
          <Plus className="h-4 w-4" />
          新建项目
        </button>
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(560px,1fr)_400px]">
        <Panel title="项目列表">
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <label className="flex h-9 min-w-64 items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-500">
              <Search className="h-4 w-4" />
              <input
                className="min-w-0 flex-1 bg-transparent text-slate-900 outline-none"
                placeholder="搜索项目名称、标识或平台"
                value={projectQuery}
                onChange={(event) => setProjectQuery(event.target.value)}
              />
            </label>
            <span className="text-sm text-slate-500">共 {projectsPage.total} 个项目</span>
          </div>

          <div className="overflow-hidden rounded-md border border-slate-200">
            <table className="w-full min-w-[680px] table-fixed text-left text-sm">
              <thead className="bg-slate-50 text-xs font-semibold text-slate-500">
                <tr>
                  <th className="w-[32%] px-3 py-2">项目</th>
                  <th className="w-[18%] px-3 py-2">平台</th>
                  <th className="w-[16%] px-3 py-2">采样率</th>
                  <th className="w-[18%] px-3 py-2">状态</th>
                  <th className="w-[16%] px-3 py-2 text-right">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {filteredProjects.map((project) => (
                  <tr key={project.id} className={clsx("transition", current?.id === project.id ? "bg-blue-50/70" : "hover:bg-slate-50")}>
                    <td className="px-3 py-3">
                      <button className="block min-w-0 text-left" onClick={() => setProjectId(project.slug)}>
                        <span className="block truncate font-medium text-slate-900">{project.name}</span>
                        <span className="block truncate text-xs text-slate-500">{project.slug}</span>
                      </button>
                    </td>
                    <td className="px-3 py-3 text-slate-600">{project.platform}</td>
                    <td className="px-3 py-3 text-slate-600">{project.sample_rate}</td>
                    <td className="px-3 py-3">
                      <StatusBadge status={project.status} />
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex justify-end gap-2">
                        <button className="btn h-8 px-2" onClick={() => setProjectId(project.slug)}>
                          查看
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {projects.length === 0 && <Empty label="暂无项目" />}
          {projects.length > 0 && filteredProjects.length === 0 && <Empty label="没有匹配的项目" />}
          <Pagination page={projectsPage} setOffset={setProjectOffset} />
        </Panel>

        <Panel title="项目详情">
          {current ? (
            <div className="grid gap-4">
              <div className="min-w-0 border-b border-slate-100 pb-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate text-lg font-semibold text-slate-950">{current.name}</div>
                    <div className="mt-1 truncate text-sm text-slate-500">{current.slug}</div>
                  </div>
                  <StatusBadge status={current.status} />
                </div>
                <div className="mt-4 flex flex-wrap gap-2">
                  <button className="btn" onClick={() => setProjectModal("edit")}>
                    <Edit3 className="h-4 w-4" />
                    编辑
                  </button>
                  <button className="btn" onClick={() => setKeyDrawerOpen(true)}>
                    <KeyRound className="h-4 w-4" />
                    管理 Key
                  </button>
                  <button className="btn" onClick={() => setOnboardingOpen(true)}>
                    <BookOpen className="h-4 w-4" />
                    接入指引
                  </button>
                </div>
              </div>

              <div className="grid gap-3">
                <Metric label="平台" value={current.platform} />
                <Metric label="采样率" value={String(current.sample_rate)} />
                <Metric label="组织 ID" value={current.organization_id} />
                <Metric label="创建时间" value={formatDate(current.created_at)} />
                <Metric label="更新时间" value={formatDate(current.updated_at)} />
              </div>

              <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold text-slate-900">项目启用状态</div>
                    <div className="mt-1 text-xs text-slate-500">停用后将拒绝该项目的上报请求。</div>
                  </div>
                  <Switch
                    checked={current.status === "active"}
                    onChange={(checked) => void setProjectStatus(current.slug, checked ? "active" : "disabled")}
                    className={clsx("relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition", current.status === "active" ? "bg-emerald-600" : "bg-slate-300")}
                  >
                    <span className={clsx("inline-block h-5 w-5 rounded-full bg-white transition", current.status === "active" ? "translate-x-5" : "translate-x-1")} />
                  </Switch>
                </div>
              </div>

              <div className="rounded-md border border-slate-200 p-3">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold text-slate-900">DSN Key</div>
                    <div className="mt-1 text-xs text-slate-500">当前页显示 {keys.length} 个，共 {keysPage.total} 个。</div>
                  </div>
                  <button className="btn h-8 px-2" onClick={() => setKeyDrawerOpen(true)}>
                    <Settings2 className="h-4 w-4" />
                    配置
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <Empty label="请选择项目" />
          )}
        </Panel>
      </div>

      <ProjectDialog mode={projectModal} project={current} close={() => setProjectModal(null)} submit={submitProject} />
      <KeyDrawer
        open={keyDrawerOpen}
        close={() => setKeyDrawerOpen(false)}
        current={current}
        keys={keys}
        keysPage={keysPage}
        setKeyOffset={setKeyOffset}
        copiedKey={copiedKey}
        copyKey={copyKey}
        setKeyModal={setKeyModal}
        setKeyStatus={setKeyStatus}
      />
      <OnboardingDrawer open={onboardingOpen} close={() => setOnboardingOpen(false)} current={current} projectKey={keys.find((key) => key.status === "active") ?? keys[0]} />
      <KeyDialog mode={keyModal} close={() => setKeyModal(null)} submit={submitKey} />
    </section>
  );
}

function OnboardingDrawer({
  open,
  close,
  current,
  projectKey,
}: {
  open: boolean;
  close: () => void;
  current?: Project;
  projectKey?: ProjectKey;
}) {
  const [message, setMessage] = useState("");
  const endpoint = current ? `${window.location.origin}/api/${current.slug}/envelope` : "";
  const publicKey = projectKey?.public_key ?? "";
  const snippet = current && publicKey ? javascriptSnippet(endpoint, publicKey) : "";

  async function copy(text: string) {
    await navigator.clipboard.writeText(text);
    setMessage("已复制到剪贴板");
    window.setTimeout(() => setMessage(""), 1200);
  }

  async function sendTest() {
    if (!current || !publicKey) {
      setMessage("请先创建并启用一个 DSN Key");
      return;
    }
    setMessage("");
    try {
      await sendTestEvent(current.slug, publicKey);
      setMessage("测试事件已发送，稍后可在 Issue / Event 页面查看");
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "测试事件发送失败");
    }
  }

  return (
    <Dialog open={open} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-y-0 right-0 flex w-full justify-end sm:w-[680px]">
        <DialogPanel className="flex h-full w-full flex-col bg-white shadow-xl">
          <div className="border-b border-slate-200 p-5">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0">
                <DialogTitle className="text-lg font-semibold text-slate-950">项目接入指引</DialogTitle>
                <p className="mt-1 truncate text-sm text-slate-500">{current ? `${current.name} / ${current.slug}` : "请选择项目"}</p>
              </div>
              <button className="btn h-8 w-8 px-0" onClick={close} title="关闭">
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-auto p-5">
            {current && projectKey ? (
              <div className="grid gap-4">
                {message && <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">{message}</div>}

                <section className="grid gap-3 rounded-md border border-slate-200 p-4">
                  <FieldRow label="上报地址" value={endpoint} copy={() => void copy(endpoint)} />
                  <FieldRow label="Public Key" value={publicKey} copy={() => void copy(publicKey)} />
                  <FieldRow label="项目标识" value={current.slug} copy={() => void copy(current.slug)} />
                </section>

                <section className="rounded-md border border-slate-200">
                  <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
                    <div className="text-sm font-semibold text-slate-800">JavaScript 示例</div>
                    <button className="btn h-8 px-2" onClick={() => void copy(snippet)}>
                      <Copy className="h-4 w-4" />
                      复制
                    </button>
                  </div>
                  <pre className="max-h-96 overflow-auto rounded-b-md bg-slate-950 p-4 text-xs leading-6 text-slate-100">{snippet}</pre>
                </section>

                <button className="btn-primary w-fit" onClick={() => void sendTest()}>
                  <Send className="h-4 w-4" />
                  发送测试事件
                </button>
              </div>
            ) : (
              <Empty label="请先选择项目并创建 DSN Key" />
            )}
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
}

function FieldRow({ label, value, copy }: { label: string; value: string; copy: () => void }) {
  return (
    <div className="grid gap-2">
      <div className="text-xs font-semibold text-slate-500">{label}</div>
      <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
        <code className="min-w-0 truncate rounded-md bg-slate-100 px-2 py-2 text-xs text-slate-700">{value}</code>
        <button className="btn h-8 px-2" onClick={copy}>
          <Copy className="h-4 w-4" />
          复制
        </button>
      </div>
    </div>
  );
}

function javascriptSnippet(endpoint: string, publicKey: string) {
  return `window.addEventListener("error", (event) => {
  fetch("${endpoint}", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Sentry-Key": "${publicKey}"
    },
    body: JSON.stringify({
      event_id: crypto.randomUUID(),
      timestamp: new Date().toISOString(),
      platform: "javascript",
      level: "error",
      message: event.message,
      exception: {
        type: event.error?.name || "Error",
        value: event.error?.message || event.message,
        stacktrace: event.error?.stack ? event.error.stack.split("\\n") : []
      },
      release: "1.0.0",
      environment: "production"
    })
  });
});`;
}

function KeyDrawer({
  open,
  close,
  current,
  keys,
  keysPage,
  setKeyOffset,
  copiedKey,
  copyKey,
  setKeyModal,
  setKeyStatus,
}: {
  open: boolean;
  close: () => void;
  current?: Project;
  keys: ProjectKey[];
  keysPage: PageMeta;
  setKeyOffset: (offset: number) => void;
  copiedKey: string;
  copyKey: (publicKey: string) => Promise<void>;
  setKeyModal: (mode: "create" | ProjectKey | null) => void;
  setKeyStatus: (keyId: string, status: string) => Promise<void>;
}) {
  return (
    <Dialog open={open} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-y-0 right-0 flex w-full justify-end sm:w-[560px]">
        <DialogPanel className="flex h-full w-full flex-col bg-white shadow-xl">
          <div className="border-b border-slate-200 p-5">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0">
                <DialogTitle className="text-lg font-semibold text-slate-950">DSN Key 管理</DialogTitle>
                <p className="mt-1 truncate text-sm text-slate-500">{current ? `${current.name} / ${current.slug}` : "请选择项目"}</p>
              </div>
              <button className="btn h-8 w-8 px-0" onClick={close} title="关闭">
                <X className="h-4 w-4" />
              </button>
            </div>
            <button className="btn-primary mt-4" disabled={!current} onClick={() => setKeyModal("create")}>
              <KeyRound className="h-4 w-4" />
              新建 Key
            </button>
          </div>

          <div className="min-h-0 flex-1 overflow-auto p-5">
            {current ? (
              <div className="grid gap-3">
                {keys.map((key) => (
                  <div key={key.id} className="grid gap-3 rounded-md border border-slate-200 p-3">
                    <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start">
                      <div className="min-w-0">
                        <div className="truncate font-medium text-slate-900">{key.name}</div>
                        <div className="mt-1 text-xs text-slate-500">限流 {key.rate_limit_per_minute}/min</div>
                      </div>
                      <div className="flex items-center gap-2">
                        <button className="btn h-8 w-8 px-0" onClick={() => setKeyModal(key)} title="编辑 Key">
                          <Edit3 className="h-4 w-4" />
                        </button>
                        <button className="btn h-8 w-8 px-0" onClick={() => void copyKey(key.public_key)} title="复制 public key">
                          {copiedKey === key.public_key ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                        </button>
                        <Switch
                          checked={key.status === "active"}
                          onChange={(checked) => void setKeyStatus(key.id, checked ? "active" : "disabled")}
                          className={clsx("relative inline-flex h-6 w-11 items-center rounded-full transition", key.status === "active" ? "bg-emerald-600" : "bg-slate-300")}
                        >
                          <span className={clsx("inline-block h-5 w-5 rounded-full bg-white transition", key.status === "active" ? "translate-x-5" : "translate-x-1")} />
                        </Switch>
                      </div>
                    </div>
                    <code className="block min-w-0 truncate rounded-md bg-slate-100 px-2 py-2 text-xs text-slate-700">{key.public_key}</code>
                  </div>
                ))}
                {keys.length === 0 && <Empty label="暂无 DSN Key" />}
              </div>
            ) : (
              <Empty label="请选择项目" />
            )}
          </div>

          <div className="border-t border-slate-200 px-5 pb-5">
            <Pagination page={keysPage} setOffset={setKeyOffset} />
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
}

function Pagination({ page, setOffset }: { page: PageMeta; setOffset: (offset: number) => void }) {
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
        <button className="btn" disabled={!hasPrev} onClick={() => setOffset(Math.max(0, page.offset - page.limit))}>
          上一页
        </button>
        <button className="btn" disabled={!hasNext} onClick={() => setOffset(page.offset + page.limit)}>
          下一页
        </button>
      </div>
    </div>
  );
}

function ProjectDialog({
  mode,
  project,
  close,
  submit,
}: {
  mode: "create" | "edit" | null;
  project?: Project;
  close: () => void;
  submit: (form: FormData) => Promise<void>;
}) {
  return (
    <FormDialog open={Boolean(mode)} title={mode === "edit" ? "编辑项目" : "新建项目"} close={close}>
      <form
        className="grid gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void submit(new FormData(event.currentTarget));
        }}
      >
        {mode === "create" && <input className="field" name="organization_slug" defaultValue="demo" placeholder="组织 slug" />}
        {mode === "create" && <input className="field" name="slug" placeholder="项目标识" required />}
        <input className="field" name="name" placeholder="项目名称" defaultValue={project?.name ?? ""} required />
        <select className="field" name="platform" defaultValue={project?.platform ?? "javascript"}>
          <option value="javascript">javascript</option>
          <option value="go">go</option>
          <option value="python">python</option>
          <option value="java">java</option>
        </select>
        <input className="field" name="sample_rate" type="number" min="0" max="1" step="0.01" defaultValue={project?.sample_rate ?? 1} />
        <button className="btn-primary">
          <Save className="h-4 w-4" />
          保存
        </button>
      </form>
    </FormDialog>
  );
}

function KeyDialog({
  mode,
  close,
  submit,
}: {
  mode: "create" | ProjectKey | null;
  close: () => void;
  submit: (form: FormData) => Promise<void>;
}) {
  const key = mode && mode !== "create" ? mode : undefined;

  return (
    <FormDialog open={Boolean(mode)} title={key ? "编辑 DSN Key" : "新建 DSN Key"} close={close}>
      <form
        className="grid gap-3"
        onSubmit={(event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          void submit(new FormData(event.currentTarget));
        }}
      >
        <input className="field" name="name" placeholder="Key 名称" defaultValue={key?.name ?? ""} required />
        <input className="field" name="rate_limit_per_minute" type="number" min="1" defaultValue={key?.rate_limit_per_minute ?? 6000} />
        <button className="btn-primary">
          <Save className="h-4 w-4" />
          保存
        </button>
      </form>
    </FormDialog>
  );
}

function FormDialog({ open, title, close, children }: { open: boolean; title: string; close: () => void; children: ReactNode }) {
  return (
    <Dialog open={open} onClose={close} className="relative z-50">
      <div className="fixed inset-0 bg-slate-950/35" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="w-full max-w-lg rounded-lg bg-white p-5 shadow-xl">
          <div className="mb-4 flex items-center justify-between gap-4">
            <DialogTitle className="text-lg font-semibold text-slate-900">{title}</DialogTitle>
            <button className="btn h-8 w-8 px-0" onClick={close} title="关闭">
              <X className="h-4 w-4" />
            </button>
          </div>
          {children}
        </DialogPanel>
      </div>
    </Dialog>
  );
}

function StatusBadge({ status }: { status: string }) {
  return <span className={clsx("badge", status === "active" ? "bg-emerald-100 text-emerald-700" : "bg-slate-200 text-slate-600")}>{status}</span>;
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs font-semibold text-slate-500">{label}</div>
      <div className="mt-1 truncate text-sm font-medium text-slate-900">{value}</div>
    </div>
  );
}
