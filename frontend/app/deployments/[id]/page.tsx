"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Clock,
  ExternalLink,
  GitBranch,
  Loader2,
  RotateCw,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api";
import type { Deployment, DeploymentLog } from "@/lib/types";

interface DeploymentDetailPageProps {
  params: Promise<{ id: string }>;
}

const ACTIVE_STATUSES = new Set(["QUEUED", "BUILDING"]);

export default function DeploymentDetailPage({
  params,
}: DeploymentDetailPageProps) {
  const { id: deploymentId } = use(params);
  const [deployment, setDeployment] = useState<Deployment | null>(null);
  const [logs, setLogs] = useState<DeploymentLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadInitialDeployment() {
      try {
        const [deploymentData, logsData] = await Promise.all([
          apiFetch<Deployment>(`/deployments/${deploymentId}`),
          apiFetch<DeploymentLog[]>(`/logs/${deploymentId}`),
        ]);
        if (cancelled) return;
        setDeployment(deploymentData);
        setLogs(logsData);
        setError("");
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Failed to load deployment");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadInitialDeployment();

    return () => {
      cancelled = true;
    };
  }, [deploymentId]);

  const refreshDeployment = useCallback(async () => {
    setRefreshing(true);
    setError("");

    try {
      const [deploymentData, logsData] = await Promise.all([
        apiFetch<Deployment>(`/deployments/${deploymentId}`),
        apiFetch<DeploymentLog[]>(`/logs/${deploymentId}`),
      ]);
      setDeployment(deploymentData);
      setLogs(logsData);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load deployment");
    } finally {
      setRefreshing(false);
    }
  }, [deploymentId]);

  useEffect(() => {
    if (!deployment || !ACTIVE_STATUSES.has(deployment.status)) return;

    const interval = window.setInterval(() => {
      void refreshDeployment();
    }, 4000);

    return () => window.clearInterval(interval);
  }, [deployment, refreshDeployment]);

  if (loading) {
    return (
      <main className="min-h-screen bg-[#09090b] text-zinc-50">
        <div className="mx-auto max-w-5xl px-6 py-12">
          <div className="py-12 text-center text-zinc-500">
            Loading deployment...
          </div>
        </div>
      </main>
    );
  }

  if (error || !deployment) {
    return (
      <main className="min-h-screen bg-[#09090b] text-zinc-50">
        <div className="mx-auto max-w-5xl px-6 py-12">
          <Link
            href="/projects"
            className="mb-6 inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-300"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Projects
          </Link>
          <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {error || "Deployment not found"}
          </div>
        </div>
      </main>
    );
  }

  const isActive = ACTIVE_STATUSES.has(deployment.status);

  return (
    <main className="min-h-screen bg-[#09090b] text-zinc-50">
      <div className="mx-auto max-w-5xl px-6 py-12">
        <div className="mb-8">
          <Link
            href={
              deployment.project_id
                ? `/projects/${deployment.project_id}`
                : "/projects"
            }
            className="mb-6 inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-300"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Project
          </Link>

          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-400">
                Deployment
              </p>
              <h1 className="break-all text-3xl font-semibold tracking-tight">
                {deployment.id}
              </h1>
            </div>
            <Button
              variant="outline"
              onClick={refreshDeployment}
              disabled={refreshing}
            >
              {refreshing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RotateCw className="h-4 w-4" />
              )}
              Refresh
            </Button>
          </div>
        </div>

        <div className="mb-8 grid gap-4 md:grid-cols-4">
          <Metric label="Status">{getStatusBadge(deployment.status)}</Metric>
          <Metric label="Branch">
            <span className="inline-flex items-center gap-1 text-zinc-300">
              <GitBranch className="h-4 w-4 text-zinc-500" />
              {deployment.branch || "-"}
            </span>
          </Metric>
          <Metric label="Duration">
            {deployment.build_duration_seconds
              ? `${deployment.build_duration_seconds}s`
              : isActive
                ? "Running"
                : "-"}
          </Metric>
          <Metric label="Attempts">
            {deployment.attempt_count}/{deployment.max_attempts}
          </Metric>
        </div>

        <div className="mb-8 rounded-lg border border-zinc-800 bg-zinc-900/30 p-6">
          <h2 className="mb-4 text-lg font-medium">Details</h2>
          <div className="grid gap-6 md:grid-cols-2">
            <Detail label="Created" value={formatDate(deployment.created_at)} />
            <Detail label="Started" value={formatDate(deployment.started_at)} />
            <Detail label="Finished" value={formatDate(deployment.finished_at)} />
            <div>
              <p className="mb-1 text-xs text-zinc-500">Served URL</p>
              {deployment.status === "READY" && deployment.url ? (
                <a
                  href={deployment.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 break-all text-emerald-400 hover:text-emerald-300"
                >
                  {deployment.url}
                  <ExternalLink className="h-3 w-3" />
                </a>
              ) : (
                <p className="text-zinc-600">-</p>
              )}
            </div>
          </div>
        </div>

        {deployment.error_message ? (
          <div className="mb-8 rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-300">
            {deployment.error_message}
          </div>
        ) : null}

        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-medium">Logs</h2>
            {isActive ? (
              <span className="inline-flex items-center gap-2 text-xs text-zinc-500">
                <Clock className="h-3.5 w-3.5" />
                Polling every 4s
              </span>
            ) : null}
          </div>

          <div className="max-h-[520px] overflow-auto rounded-lg border border-zinc-800 bg-black p-4 font-mono text-xs text-zinc-300">
            {logs.length === 0 ? (
              <p className="text-zinc-600">No logs yet.</p>
            ) : (
              <div className="space-y-3">
                {logs.map((log, index) => (
                  <div
                    key={`${log.time}-${index}`}
                    className="grid gap-3 border-b border-zinc-900 pb-3 last:border-0 last:pb-0 md:grid-cols-[180px_1fr]"
                  >
                    <time className="text-zinc-600">{formatDate(log.time)}</time>
                    <pre className="whitespace-pre-wrap break-words">
                      {log.message}
                    </pre>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}

function Metric({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <p className="mb-2 text-xs text-zinc-500">{label}</p>
      <div className="text-sm text-zinc-300">{children}</div>
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="mb-1 text-xs text-zinc-500">{label}</p>
      <p className="text-zinc-300">{value}</p>
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
      className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${
        styles[status] || "bg-zinc-800 text-zinc-400"
      }`}
    >
      {status}
    </span>
  );
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}
