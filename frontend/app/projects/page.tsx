"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import type { Deployment, Project } from "@/lib/types";
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
  ExternalLink,
  Loader2,
  Play,
  Plus,
  RefreshCw,
  Search,
} from "lucide-react";

type ProjectWithLatestDeployment = Project & {
  latestDeployment: Deployment | null;
};

type TriggerDeploymentResponse = {
  deployment_id: string;
};

const ACTIVE_STATUSES = new Set(["QUEUED", "BUILDING"]);
const STATUS_OPTIONS = ["ALL", "READY", "FAILED", "BUILDING", "QUEUED", "NEVER_DEPLOYED"] as const;
type StatusFilter = (typeof STATUS_OPTIONS)[number];

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("ALL");
  const [deployingProjectId, setDeployingProjectId] = useState<string | null>(null);
  const [deployError, setDeployError] = useState("");

  async function loadProjects() {
    const [projectsData, deploymentsData] = await Promise.all([
      apiFetch<Project[]>("/projects"),
      apiFetch<Deployment[]>("/deployments"),
    ]);
    setProjects(projectsData);
    setDeployments(
      [...deploymentsData].sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
      ),
    );
  }

  useEffect(() => {
    let cancelled = false;

    async function loadInitialProjects() {
      try {
        const [projectsData, deploymentsData] = await Promise.all([
          apiFetch<Project[]>("/projects"),
          apiFetch<Deployment[]>("/deployments"),
        ]);
        if (cancelled) return;
        setProjects(projectsData);
        setDeployments(
          [...deploymentsData].sort(
            (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
          ),
        );
        setError("");
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Failed to load projects");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadInitialProjects();

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleRefresh() {
    setRefreshing(true);
    setError("");
    try {
      await loadProjects();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to refresh projects");
    } finally {
      setRefreshing(false);
    }
  }

  const projectsWithLatestDeployment = useMemo<ProjectWithLatestDeployment[]>(() => {
    const latestByProject = new Map<string, Deployment>();

    for (const deployment of deployments) {
      if (!deployment.project_id || latestByProject.has(deployment.project_id)) {
        continue;
      }
      latestByProject.set(deployment.project_id, deployment);
    }

    return projects
      .map((project) => ({
        ...project,
        latestDeployment: latestByProject.get(project.id) ?? null,
      }))
      .sort((a, b) => {
        const aTime = a.latestDeployment
          ? new Date(a.latestDeployment.created_at).getTime()
          : new Date(a.created_at).getTime();
        const bTime = b.latestDeployment
          ? new Date(b.latestDeployment.created_at).getTime()
          : new Date(b.created_at).getTime();
        return bTime - aTime;
      });
  }, [deployments, projects]);

  const filteredProjects = useMemo(() => {
    const query = search.trim().toLowerCase();

    return projectsWithLatestDeployment.filter((project) => {
      const compactRepo = formatRepo(project.repo_url).toLowerCase();
      const matchesSearch =
        query === "" ||
        project.name.toLowerCase().includes(query) ||
        compactRepo.includes(query) ||
        project.repo_url.toLowerCase().includes(query);

      const status = project.latestDeployment?.status ?? "NEVER_DEPLOYED";
      const matchesStatus = statusFilter === "ALL" || status === statusFilter;

      return matchesSearch && matchesStatus;
    });
  }, [projectsWithLatestDeployment, search, statusFilter]);

  const summary = useMemo(() => {
    let ready = 0;
    let failed = 0;
    let active = 0;

    for (const project of projectsWithLatestDeployment) {
      const status = project.latestDeployment?.status;
      if (status === "READY") ready += 1;
      if (status === "FAILED") failed += 1;
      if (status && ACTIVE_STATUSES.has(status)) active += 1;
    }

    return {
      total: projectsWithLatestDeployment.length,
      ready,
      failed,
      active,
    };
  }, [projectsWithLatestDeployment]);

  async function handleDeploy(projectId: string) {
    setDeployingProjectId(projectId);
    setDeployError("");

    try {
      const { deployment_id } = await apiFetch<TriggerDeploymentResponse>(
        `/projects/${projectId}/deployments`,
        { method: "POST" },
      );
      const deployment = await apiFetch<Deployment>(`/deployments/${deployment_id}`);
      setDeployments((prev) => {
        const next = [deployment, ...prev.filter((item) => item.id !== deployment.id)];
        return next.sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        );
      });
    } catch (e) {
      setDeployError(e instanceof Error ? e.message : "Failed to trigger deployment");
    } finally {
      setDeployingProjectId(null);
    }
  }

  return (
    <main className="min-h-screen bg-[#09090b] text-zinc-50">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <header className="mb-8 flex flex-col gap-6 md:flex-row md:items-start md:justify-between">
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-400">
              Shipyard
            </p>
            <h1 className="text-3xl font-semibold tracking-tight">Projects</h1>
            <p className="mt-3 max-w-2xl text-sm text-zinc-400">
              Manage project settings, trigger deployments, and inspect recent activity.
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button
              variant="outline"
              onClick={handleRefresh}
              disabled={refreshing || loading}
              className="gap-2"
            >
              {refreshing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              Refresh
            </Button>
            <Button asChild className="gap-2">
              <Link href="/projects/new">
                <Plus className="h-4 w-4" />
                Create Project
              </Link>
            </Button>
          </div>
        </header>

        <section className="mb-8 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Total Projects" value={summary.total} />
          <SummaryCard label="Ready" value={summary.ready} tone="emerald" />
          <SummaryCard label="Failures" value={summary.failed} tone="red" />
          <SummaryCard label="Active Builds" value={summary.active} tone="amber" />
        </section>

        <section className="mb-6 flex flex-col gap-3 rounded-lg border border-zinc-800 bg-zinc-900/30 p-4 md:flex-row md:items-center md:justify-between">
          <div className="relative w-full md:max-w-sm">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search projects"
              className="h-10 w-full border border-zinc-800 bg-[#09090b] pl-10 pr-3 text-sm text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-emerald-500"
            />
          </div>

          <div className="flex items-center gap-3">
            <label className="text-xs uppercase tracking-[0.18em] text-zinc-500">
              Status
            </label>
            <select
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
              className="h-10 border border-zinc-800 bg-[#09090b] px-3 text-sm text-zinc-100 outline-none focus:border-emerald-500"
            >
              {STATUS_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {option === "NEVER_DEPLOYED" ? "Never Deployed" : option}
                </option>
              ))}
            </select>
          </div>
        </section>

        {deployError ? (
          <div className="mb-6 rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {deployError}
          </div>
        ) : null}

        {loading ? (
          <ProjectsTableSkeleton />
        ) : error ? (
          <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {error}
          </div>
        ) : filteredProjects.length === 0 ? (
          <EmptyState hasProjects={projectsWithLatestDeployment.length > 0} />
        ) : (
          <div className="overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950/40">
            <Table>
              <TableHeader>
                <TableRow className="border-zinc-800 hover:bg-transparent">
                  <TableHead className="text-zinc-400">Name</TableHead>
                  <TableHead className="text-zinc-400">Repo</TableHead>
                  <TableHead className="text-zinc-400">Preset</TableHead>
                  <TableHead className="text-zinc-400">Branch</TableHead>
                  <TableHead className="text-zinc-400">Last Status</TableHead>
                  <TableHead className="text-zinc-400">Last Deploy</TableHead>
                  <TableHead className="text-zinc-400">Duration</TableHead>
                  <TableHead className="text-right text-zinc-400">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredProjects.map((project) => {
                  const deployment = project.latestDeployment;
                  const isDeploying = deployingProjectId === project.id;

                  return (
                    <TableRow
                      key={project.id}
                      className="border-zinc-800/50 hover:bg-zinc-800/20"
                    >
                      <TableCell>
                        <div className="space-y-1">
                          <Link
                            href={`/projects/${project.id}`}
                            className="font-medium text-zinc-50 hover:text-emerald-400"
                          >
                            {project.name}
                          </Link>
                          <p className="text-xs text-zinc-500">
                            Created {formatShortDate(project.created_at)}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <a
                          href={project.repo_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 text-zinc-400 hover:text-zinc-200"
                          title={project.repo_url}
                        >
                          {formatRepo(project.repo_url)}
                          <ExternalLink className="h-3 w-3" />
                        </a>
                      </TableCell>
                      <TableCell>
                        <span className="inline-flex rounded bg-zinc-800 px-2 py-0.5 text-xs text-zinc-300">
                          {project.build_preset}
                        </span>
                      </TableCell>
                      <TableCell className="text-zinc-400">{project.default_branch}</TableCell>
                      <TableCell>{getStatusBadge(deployment?.status)}</TableCell>
                      <TableCell className="text-zinc-400">
                        {deployment ? formatRelativeTime(deployment.created_at) : "Never"}
                      </TableCell>
                      <TableCell className="text-zinc-400">
                        {deployment?.build_duration_seconds
                          ? `${deployment.build_duration_seconds}s`
                          : "-"}
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end gap-2">
                          <Button asChild variant="outline" size="xs">
                            <Link href={`/projects/${project.id}`}>Open</Link>
                          </Button>
                          <Button
                            size="xs"
                            onClick={() => handleDeploy(project.id)}
                            disabled={isDeploying}
                            className="gap-1"
                          >
                            {isDeploying ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <Play className="h-3 w-3" />
                            )}
                            Deploy
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </main>
  );
}

function SummaryCard({
  label,
  value,
  tone = "zinc",
}: {
  label: string;
  value: number;
  tone?: "zinc" | "emerald" | "red" | "amber";
}) {
  const valueClass = {
    zinc: "text-zinc-50",
    emerald: "text-emerald-400",
    red: "text-red-400",
    amber: "text-amber-400",
  }[tone];

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5">
      <p className="text-xs font-medium uppercase tracking-[0.18em] text-zinc-500">{label}</p>
      <p className={`mt-3 text-3xl font-semibold tracking-tight ${valueClass}`}>{value}</p>
    </div>
  );
}

function ProjectsTableSkeleton() {
  return (
    <div className="overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950/40">
      <Table>
        <TableHeader>
          <TableRow className="border-zinc-800 hover:bg-transparent">
            <TableHead className="text-zinc-400">Name</TableHead>
            <TableHead className="text-zinc-400">Repo</TableHead>
            <TableHead className="text-zinc-400">Preset</TableHead>
            <TableHead className="text-zinc-400">Branch</TableHead>
            <TableHead className="text-zinc-400">Last Status</TableHead>
            <TableHead className="text-zinc-400">Last Deploy</TableHead>
            <TableHead className="text-zinc-400">Duration</TableHead>
            <TableHead className="text-right text-zinc-400">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: 5 }).map((_, index) => (
            <TableRow key={index} className="border-zinc-800/50">
              {Array.from({ length: 8 }).map((__, cellIndex) => (
                <TableCell key={cellIndex}>
                  <div className="h-4 animate-pulse bg-zinc-800" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function EmptyState({ hasProjects }: { hasProjects: boolean }) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 px-6 py-12 text-center">
      <h2 className="text-lg font-medium text-zinc-100">
        {hasProjects ? "No projects match these filters" : "No projects yet"}
      </h2>
      <p className="mt-2 text-sm text-zinc-400">
        {hasProjects
          ? "Adjust search or status filters to see more projects."
          : "Create your first project to store repo settings and start deploying."}
      </p>
      {!hasProjects ? (
        <Button asChild className="mt-6 gap-2">
          <Link href="/projects/new">
            <Plus className="h-4 w-4" />
            Create Project
          </Link>
        </Button>
      ) : null}
    </div>
  );
}

function getStatusBadge(status?: string) {
  const resolvedStatus = status ?? "NEVER_DEPLOYED";
  const styles: Record<string, string> = {
    QUEUED: "bg-zinc-800 text-zinc-300",
    BUILDING: "bg-amber-900/30 text-amber-400",
    READY: "bg-emerald-900/30 text-emerald-400",
    FAILED: "bg-red-900/30 text-red-400",
    NEVER_DEPLOYED: "bg-zinc-900 text-zinc-500",
  };

  return (
    <span
      className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${styles[resolvedStatus]}`}
    >
      {resolvedStatus === "NEVER_DEPLOYED" ? "Never deployed" : resolvedStatus}
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

function formatShortDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function formatRelativeTime(value: string) {
  const diffMs = Date.now() - new Date(value).getTime();
  const diffMinutes = Math.floor(diffMs / 60000);

  if (diffMinutes < 1) return "Just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d ago`;

  return formatShortDate(value);
}
