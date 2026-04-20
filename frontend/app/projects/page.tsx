"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import type { Project } from "@/lib/types";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Plus, ExternalLink } from "lucide-react";

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<Project[]>("/projects")
      .then(setProjects)
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load projects"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <main className="min-h-screen bg-[#09090b] text-zinc-50">
      <div className="mx-auto max-w-5xl px-6 py-12">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-400">
              Shipyard
            </p>
            <h1 className="text-3xl font-semibold tracking-tight">Projects</h1>
          </div>
          <Button asChild className="gap-2">
            <Link href="/projects/new">
              <Plus className="h-4 w-4" />
              Create Project
            </Link>
          </Button>
        </div>

        {loading ? (
          <div className="py-12 text-center text-zinc-500">Loading projects...</div>
        ) : error ? (
          <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
            {error}
          </div>
        ) : projects.length === 0 ? (
          <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 px-6 py-12 text-center">
            <p className="text-zinc-400">No projects yet. Create your first one.</p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-zinc-800">
            <Table>
              <TableHeader>
                <TableRow className="border-zinc-800 hover:bg-transparent">
                  <TableHead className="text-zinc-400">Name</TableHead>
                  <TableHead className="text-zinc-400">Repo</TableHead>
                  <TableHead className="text-zinc-400">Preset</TableHead>
                  <TableHead className="text-zinc-400">Branch</TableHead>
                  <TableHead className="text-zinc-400">Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {projects.map((project) => (
                  <TableRow
                    key={project.id}
                    className="border-zinc-800/50 hover:bg-zinc-800/30"
                  >
                    <TableCell>
                      <Link
                        href={`/projects/${project.id}`}
                        className="font-medium text-zinc-50 hover:text-emerald-400"
                      >
                        {project.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <a
                        href={project.repo_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-zinc-400 hover:text-zinc-200"
                      >
                        {project.repo_url.replace("https://", "")}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    </TableCell>
                    <TableCell>
                      <span className="inline-flex rounded bg-zinc-800 px-2 py-0.5 text-xs text-zinc-300">
                        {project.build_preset}
                      </span>
                    </TableCell>
                    <TableCell className="text-zinc-400">{project.default_branch}</TableCell>
                    <TableCell className="text-zinc-500">
                      {new Date(project.created_at).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </main>
  );
}
