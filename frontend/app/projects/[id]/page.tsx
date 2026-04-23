"use client";

import { useEffect, useState, useTransition } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import type { Project, Deployment } from "@/lib/types";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { ArrowLeft, GitBranch, Play, ExternalLink, Clock, Loader2 } from "lucide-react";

interface ProjectDetailPageProps {
  params: Promise<{ id: string }>;
}

export default function ProjectDetailPage({ params }: ProjectDetailPageProps) {
  const [project, setProject] = useState<Project | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    params.then(async (p) => {
      const id = p.id;
      try {
        const [projectData, deploymentsData] = await Promise.all([
          apiFetch<Project>(`/projects/${id}`),
          apiFetch<Deployment[]>("/deployments"),
        ]);
        setProject(projectData);
        const filtered = deploymentsData
          .filter((d) => d.project_id === id)
          .sort(
            (a, b) =>
              new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
          );
        setDeployments(filtered);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load project");
      } finally {
        setLoading(false);
      }
    });
  }, [params]);

  const handleDeploy = async () => {
    if (!project) return;
    setDeploying(true);
    try {
      const newDeployment = await apiFetch<Deployment>(
        `/projects/${project.id}/deployments`,
        { method: "POST" }
      );
      setDeployments((prev) => [newDeployment, ...prev]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to trigger deployment");
    } finally {
      setDeploying(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      QUEUED: "bg-zinc-800 text-zinc-400",
      BUILDING: "bg-amber-900/30 text-amber-400",
      READY: "bg-emerald-900/30 text-emerald-400",
      FAILED: "bg-red-900/30 text-red-400",
    };
    return (
      <span
        className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${
          styles[status] || "bg-zinc-800 text-zinc-400"
        }`}
      >
        {status}
      </span>
    );
  };

  if (loading) {
    return (
      <main className="min-h-screen bg-[#09090b] text-zinc-50">
        <div className="mx-auto max-w-5xl px-6 py-12">
          <div className="py-12 text-center text-zinc-500">Loading project...</div>
        </div>
      </main>
    );
  }

  if (error || !project) {
    return (
      <main className="min-h-screen bg-[#09090b] text-zinc-50">
        <div className="mx-auto max-w-5xl px-6 py-12">
          <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {error || "Project not found"}
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#09090b] text-zinc-50">
      <div className="mx-auto max-w-5xl px-6 py-12">
        <div className="mb-8">
          <Link
            href="/projects"
            className="mb-6 inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-300"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Projects
          </Link>

          <div className="flex items-start justify-between">
            <div>
              <p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-400">
                Shipyard
              </p>
              <h1 className="text-3xl font-semibold tracking-tight">
                {project.name}
              </h1>
            </div>
            <Button
              onClick={handleDeploy}
              disabled={deploying}
              className="gap-2 bg-emerald-600 hover:bg-emerald-700"
            >
              {deploying ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Play className="h-4 w-4" />
              )}
              Deploy
            </Button>
          </div>
        </div>

        <div className="mb-8 rounded-lg border border-zinc-800 bg-zinc-900/30 p-6">
          <h2 className="mb-4 text-lg font-medium">Project Details</h2>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <p className="mb-1 text-xs text-zinc-500">Repository</p>
              <a
                href={project.repo_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-zinc-300 hover:text-emerald-400"
              >
                {project.repo_url.replace("https://", "")}
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
              <p className="text-zinc-300">{project.output_dir}</p>
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
          </div>
        </div>

        <div>
          <h2 className="mb-4 text-lg font-medium">Recent Deployments</h2>
          {deployments.length === 0 ? (
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 px-6 py-12 text-center">
              <p className="text-zinc-400">No deployments yet. Trigger one to get started.</p>
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-zinc-800">
              <Table>
                <TableHeader>
                  <TableRow className="border-zinc-800 hover:bg-transparent">
                    <TableHead className="text-zinc-400">Status</TableHead>
                    <TableHead className="text-zinc-400">Branch</TableHead>
                    <TableHead className="text-zinc-400">Created</TableHead>
                    <TableHead className="text-zinc-400">Duration</TableHead>
                    <TableHead className="text-zinc-400">URL</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deployments.map((deployment, index) => (
                    <TableRow
                      key={deployment.id || `deploy-${index}-${deployment.created_at}`}
                      className="border-zinc-800/50 hover:bg-zinc-800/30"
                    >
                      <TableCell>{getStatusBadge(deployment.status)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1 text-zinc-300">
                          <GitBranch className="h-4 w-4 text-zinc-500" />
                          {deployment.branch}
                        </div>
                      </TableCell>
                      <TableCell className="text-zinc-400">
                        {new Date(deployment.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-zinc-400">
                        {deployment.build_duration_seconds
                          ? `${deployment.build_duration_seconds}s`
                          : "-"}
                      </TableCell>
                      <TableCell>
                        {deployment.url ? (
                          <a
                            href={deployment.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 text-emerald-400 hover:text-emerald-300"
                          >
                            {deployment.url.replace("https://", "")}
                            <ExternalLink className="h-3 w-3" />
                          </a>
                        ) : (
                          <span className="text-zinc-600">-</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}