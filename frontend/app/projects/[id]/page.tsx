"use client";

import { use, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import type { Project, Deployment, ProjectWebhook } from "@/lib/types";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Clock,
  Copy,
  ExternalLink,
  GitBranch,
  Loader2,
  Play,
  RefreshCw,
  Webhook,
} from "lucide-react";

interface ProjectDetailPageProps {
  params: Promise<{ id: string }>;
}

type TriggerDeploymentResponse = {
  deployment_id: string;
};

const ACTIVE_STATUSES = new Set(["QUEUED", "BUILDING"]);

function isMissingWebhookError(message: string) {
  return message === "webhook not found" || message.includes("status 404");
}

export default function ProjectDetailPage({ params }: ProjectDetailPageProps) {
  const { id } = use(params);
  const [project, setProject] = useState<Project | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [deployError, setDeployError] = useState("");
  const [webhookState, setWebhookState] = useState<ProjectWebhook | null>(null);
  const [webhookBusy, setWebhookBusy] = useState(false);
  const [webhookError, setWebhookError] = useState("");
  const [copiedField, setCopiedField] = useState<"secret" | "endpoint" | null>(null);

  const loadProject = useCallback(async () => {
    const [projectData, deploymentsData] = await Promise.all([
      apiFetch<Project>(`/projects/${id}`),
      apiFetch<Deployment[]>("/deployments"),
    ]);

    setProject(projectData);
    setDeployments(
      deploymentsData
        .filter((deployment) => deployment.project_id === id)
        .sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        ),
    );
  }, [id]);

  const loadWebhook = useCallback(async () => {
    try {
      const webhook = await apiFetch<ProjectWebhook>(`/projects/${id}/webhook`);
      setWebhookState(webhook);
      setWebhookError("");
    } catch (e) {
      const message = e instanceof Error ? e.message : "Failed to load webhook";
      if (isMissingWebhookError(message)) {
        setWebhookState(null);
        setWebhookError("");
        return;
      }
      throw e;
    }
  }, [id]);

  useEffect(() => {
    let cancelled = false;

    async function loadInitialProject() {
      try {
        const [projectData, deploymentsData, webhookResult] = await Promise.all([
          apiFetch<Project>(`/projects/${id}`),
          apiFetch<Deployment[]>("/deployments"),
          apiFetch<ProjectWebhook>(`/projects/${id}/webhook`)
            .then((webhook) => ({ webhook, error: "" }))
            .catch((e) => {
              const message = e instanceof Error ? e.message : "Failed to load webhook";
              if (isMissingWebhookError(message)) {
                return { webhook: null, error: "" };
              }
              return { webhook: null, error: message };
            }),
        ]);
        if (cancelled) return;
        setProject(projectData);
        setDeployments(
          deploymentsData
            .filter((deployment) => deployment.project_id === id)
            .sort(
              (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
            ),
        );
        setWebhookState(webhookResult.webhook);
        setWebhookError(webhookResult.error);
        setLoadError("");
      } catch (e) {
        if (!cancelled) {
          setLoadError(e instanceof Error ? e.message : "Failed to load project");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadInitialProject();

    return () => {
      cancelled = true;
    };
  }, [id]);

  const handleRefresh = useCallback(
    async (silent = false) => {
      if (!silent) {
        setRefreshing(true);
        setLoadError("");
      }

      try {
        await Promise.all([loadProject(), loadWebhook()]);
      } catch (e) {
        setLoadError(e instanceof Error ? e.message : "Failed to refresh project");
      } finally {
        if (!silent) {
          setRefreshing(false);
        }
      }
    },
    [loadProject, loadWebhook],
  );

  const latestDeployment = deployments[0] ?? null;

  useEffect(() => {
    if (!latestDeployment || !ACTIVE_STATUSES.has(latestDeployment.status)) return;

    const interval = window.setInterval(() => {
      void handleRefresh(true);
    }, 4000);

    return () => window.clearInterval(interval);
  }, [handleRefresh, latestDeployment]);

  const stats = useMemo(() => {
    const total = deployments.length;
    const ready = deployments.filter((deployment) => deployment.status === "READY").length;
    const failed = deployments.filter((deployment) => deployment.status === "FAILED").length;
    const active = deployments.filter((deployment) => ACTIVE_STATUSES.has(deployment.status)).length;

    return { total, ready, failed, active };
  }, [deployments]);

  async function handleDeploy() {
    if (!project) return;
    setDeploying(true);
    setDeployError("");

    try {
      const { deployment_id } = await apiFetch<TriggerDeploymentResponse>(
        `/projects/${project.id}/deployments`,
        { method: "POST" },
      );
      const newDeployment = await apiFetch<Deployment>(`/deployments/${deployment_id}`);
      setDeployments((prev) =>
        [newDeployment, ...prev.filter((deployment) => deployment.id !== newDeployment.id)].sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        ),
      );
    } catch (e) {
      setDeployError(e instanceof Error ? e.message : "Failed to trigger deployment");
    } finally {
      setDeploying(false);
    }
  }

  async function handleWebhookCreate() {
    if (!project) return;
    setWebhookBusy(true);
    setWebhookError("");

      try {
      const response = await apiFetch<ProjectWebhook>(
        `/projects/${project.id}/webhook`,
        { method: "POST" },
      );
      setWebhookState(response);
    } catch (e) {
      setWebhookError(e instanceof Error ? e.message : "Failed to create webhook");
    } finally {
      setWebhookBusy(false);
    }
  }

  async function copyValue(value: string, field: "secret" | "endpoint") {
    await navigator.clipboard.writeText(value);
    setCopiedField(field);
    window.setTimeout(() => setCopiedField(null), 1500);
  }

  if (loading) {
    return (
      <main className="min-h-screen bg-[#09090b] text-zinc-50">
        <div className="mx-auto max-w-6xl px-6 py-12">
          <div className="py-12 text-center text-zinc-500">Loading project...</div>
        </div>
      </main>
    );
  }

  if (loadError || !project) {
    return (
      <main className="min-h-screen bg-[#09090b] text-zinc-50">
        <div className="mx-auto max-w-6xl px-6 py-12">
          <Link
            href="/projects"
            className="mb-6 inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-300"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Projects
          </Link>
          <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {loadError || "Project not found"}
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#09090b] text-zinc-50">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <header className="mb-8">
          <Link
            href="/projects"
            className="mb-6 inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-300"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Projects
          </Link>

          <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
            <div>
              <p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-400">
                Shipyard
              </p>
              <h1 className="text-3xl font-semibold tracking-tight">{project.name}</h1>
              <p className="mt-3 max-w-2xl text-sm text-zinc-400">
                Deploy this project, inspect its latest state, and manage its GitHub webhook.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button variant="outline" onClick={() => void handleRefresh()} disabled={refreshing}>
                {refreshing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
                Refresh
              </Button>
              <Button
                onClick={handleDeploy}
                disabled={deploying}
                className="gap-2 bg-emerald-600 hover:bg-emerald-700"
              >
                {deploying ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Deploying...
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4" />
                    Deploy
                  </>
                )}
              </Button>
            </div>
          </div>
        </header>

        {deployError ? (
          <div className="mb-6 rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {deployError}
          </div>
        ) : null}

        <section className="mb-8 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Latest Status">
            {latestDeployment ? getStatusBadge(latestDeployment.status) : <span className="text-zinc-500">Never deployed</span>}
          </SummaryCard>
          <SummaryCard label="Latest Duration">
            <span className="text-2xl font-semibold tracking-tight text-zinc-50">
              {latestDeployment?.build_duration_seconds
                ? `${latestDeployment.build_duration_seconds}s`
                : latestDeployment && ACTIVE_STATUSES.has(latestDeployment.status)
                  ? "Running"
                  : "-"}
            </span>
          </SummaryCard>
          <SummaryCard label="Successful Deployments">
            <span className="text-2xl font-semibold tracking-tight text-emerald-400">{stats.ready}</span>
          </SummaryCard>
          <SummaryCard label="Failures / Active">
            <div className="flex items-baseline gap-3">
              <span className="text-2xl font-semibold tracking-tight text-red-400">{stats.failed}</span>
              <span className="text-sm text-zinc-500">active {stats.active}</span>
            </div>
          </SummaryCard>
        </section>

        <div className="mb-8 grid gap-6 xl:grid-cols-[1.4fr_0.8fr]">
          <section className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-6">
            <div className="mb-5 flex items-center justify-between">
              <h2 className="text-lg font-medium">Project Details</h2>
              <span className="text-xs uppercase tracking-[0.18em] text-zinc-500">
                {stats.total} deployments
              </span>
            </div>
            <div className="grid gap-6 md:grid-cols-2">
              <div>
                <p className="mb-1 text-xs text-zinc-500">Repository</p>
                <a
                  href={project.repo_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-zinc-300 hover:text-emerald-400"
                >
                  {formatRepo(project.repo_url)}
                  <ExternalLink className="h-3 w-3" />
                </a>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Build Preset</p>
                <span className="inline-flex rounded bg-zinc-800 px-2 py-0.5 text-xs text-zinc-300">
                  {project.build_preset}
                </span>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Output Directory</p>
                <p className="text-zinc-300">{project.output_dir || "/"}</p>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Default Branch</p>
                <div className="flex items-center gap-1 text-zinc-300">
                  <GitBranch className="h-4 w-4 text-zinc-500" />
                  {project.default_branch}
                </div>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Created</p>
                <div className="flex items-center gap-1 text-zinc-400">
                  <Clock className="h-4 w-4" />
                  {new Date(project.created_at).toLocaleString()}
                </div>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Latest Deployment</p>
                {latestDeployment ? (
                  <Link
                    href={`/deployments/${latestDeployment.id}`}
                    className="inline-flex items-center gap-1 text-zinc-300 hover:text-emerald-400"
                  >
                    {truncateId(latestDeployment.id)}
                    <ArrowRight className="h-3 w-3" />
                  </Link>
                ) : (
                  <p className="text-zinc-500">No deployments yet</p>
                )}
              </div>
            </div>
          </section>

          <section className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-6">
            <div className="mb-5 flex items-center gap-2">
              <Webhook className="h-4 w-4 text-zinc-500" />
              <h2 className="text-lg font-medium">Webhook Control</h2>
            </div>
            <div className="space-y-5">
              <div>
                <p className="mb-1 text-xs text-zinc-500">Provider</p>
                <p className="text-zinc-300">GitHub push events</p>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Tracked Branch</p>
                <p className="text-zinc-300">{project.default_branch}</p>
              </div>
              <div>
                <p className="mb-1 text-xs text-zinc-500">Endpoint</p>
                <p className="break-all text-sm text-zinc-400">
                  {webhookState ? webhookState.endpoint : "/webhooks/github"}
                </p>
              </div>
              {webhookState ? (
                <>
                  <div>
                    <p className="mb-1 text-xs text-zinc-500">Webhook ID</p>
                    <p className="break-all text-sm text-zinc-300">{webhookState.webhook_id}</p>
                  </div>
                  <div>
                    <div className="mb-1 flex items-center justify-between gap-3">
                      <p className="text-xs text-zinc-500">Secret</p>
                      <Button
                        variant="outline"
                        size="xs"
                        onClick={() => void copyValue(webhookState.secret, "secret")}
                      >
                        {copiedField === "secret" ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                        Copy
                      </Button>
                    </div>
                    <p className="break-all border border-zinc-800 bg-[#09090b] px-3 py-2 text-sm text-zinc-300">
                      {webhookState.secret}
                    </p>
                  </div>
                  <div>
                    <Button
                      variant="outline"
                      size="xs"
                      onClick={() => void copyValue(webhookState.endpoint, "endpoint")}
                    >
                      {copiedField === "endpoint" ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                      Copy Endpoint
                    </Button>
                  </div>
                </>
              ) : (
                <p className="text-sm text-zinc-500">
                  Generate a project webhook secret, then configure GitHub to send push events here.
                </p>
              )}

              {webhookError ? (
                <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
                  {webhookError}
                </div>
              ) : null}

              <Button
                variant="outline"
                onClick={handleWebhookCreate}
                disabled={webhookBusy}
                className="w-full"
              >
                {webhookBusy ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Webhook className="h-4 w-4" />
                )}
                {webhookState ? "Regenerate Secret" : "Create Webhook"}
              </Button>
            </div>
          </section>
        </div>

        <section>
          <div className="mb-4 flex items-center justify-between gap-4">
            <div>
              <h2 className="text-lg font-medium">Recent Deployments</h2>
              <p className="mt-1 text-sm text-zinc-500">
                Review the latest runs, branch activity, and served outputs for this project.
              </p>
            </div>
            {latestDeployment ? (
              <Button asChild variant="outline" size="sm">
                <Link href={`/deployments/${latestDeployment.id}`}>Open Latest</Link>
              </Button>
            ) : null}
          </div>

          {deployments.length === 0 ? (
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 px-6 py-12 text-center">
              <p className="text-zinc-400">No deployments yet. Trigger one to get started.</p>
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950/40">
              <Table>
                <TableHeader>
                  <TableRow className="border-zinc-800 hover:bg-transparent">
                    <TableHead className="text-zinc-400">Status</TableHead>
                    <TableHead className="text-zinc-400">Branch</TableHead>
                    <TableHead className="text-zinc-400">Created</TableHead>
                    <TableHead className="text-zinc-400">Duration</TableHead>
                    <TableHead className="text-zinc-400">Served URL</TableHead>
                    <TableHead className="text-right text-zinc-400">Details</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deployments.map((deployment) => (
                    <TableRow
                      key={deployment.id}
                      className="border-zinc-800/50 hover:bg-zinc-800/20"
                    >
                      <TableCell>{getStatusBadge(deployment.status)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1 text-zinc-300">
                          <GitBranch className="h-4 w-4 text-zinc-500" />
                          {deployment.branch || "-"}
                        </div>
                      </TableCell>
                      <TableCell className="text-zinc-400">
                        {new Date(deployment.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-zinc-400">
                        {deployment.build_duration_seconds
                          ? `${deployment.build_duration_seconds}s`
                          : ACTIVE_STATUSES.has(deployment.status)
                            ? "Running"
                            : "-"}
                      </TableCell>
                      <TableCell>
                        {deployment.status === "READY" && deployment.url ? (
                          <a
                            href={deployment.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 break-all text-emerald-400 hover:text-emerald-300"
                          >
                            {truncateUrl(deployment.url)}
                            <ExternalLink className="h-3 w-3" />
                          </a>
                        ) : (
                          <span className="text-zinc-600">-</span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button asChild variant="ghost" size="xs">
                          <Link href={`/deployments/${deployment.id}`}>
                            Open
                            <ArrowRight className="h-3 w-3" />
                          </Link>
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function SummaryCard({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5">
      <p className="text-xs font-medium uppercase tracking-[0.18em] text-zinc-500">{label}</p>
      <div className="mt-3">{children}</div>
    </div>
  );
}

function getStatusBadge(status: string) {
  const styles: Record<string, string> = {
    QUEUED: "bg-zinc-800 text-zinc-400",
    BUILDING: "bg-amber-900/30 text-amber-400",
    READY: "bg-emerald-900/30 text-emerald-400",
    FAILED: "bg-red-900/30 text-red-400",
  };
  return (
    <span
      className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${styles[status] || "bg-zinc-800 text-zinc-400"}`}
    >
      {status}
    </span>
  );
}

function formatRepo(repoUrl: string) {
  try {
    const url = new URL(repoUrl);
    return url.pathname.replace(/^\//, "") || repoUrl.replace(/^https?:\/\//, "");
  } catch {
    return repoUrl.replace(/^https?:\/\//, "");
  }
}

function truncateId(value: string) {
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

function truncateUrl(value: string) {
  if (value.length <= 48) return value;
  return `${value.slice(0, 36)}...${value.slice(-10)}`;
}
