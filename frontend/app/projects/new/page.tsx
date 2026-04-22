"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Loader2 } from "lucide-react";

interface CreateProjectResponse {
  project_id: string;
}

export default function NewProjectPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState("");

  const [name, setName] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [buildPreset, setBuildPreset] = useState("static-copy");
  const [outputDir, setOutputDir] = useState("dist");
  const [defaultBranch, setDefaultBranch] = useState("main");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSubmitting(true);
    setError("");

    try {
      await apiFetch<CreateProjectResponse>("/projects", {
        method: "POST",
        body: JSON.stringify({
          name,
          repo_url: repoUrl,
          build_preset: buildPreset,
          output_dir: outputDir,
          default_branch: defaultBranch,
        }),
      });
      router.push("/projects");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create project");
      setIsSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen bg-[#09090b] text-zinc-50">
      <div className="mx-auto max-w-2xl px-6 py-12">
        <div className="mb-8">
          <Link
            href="/projects"
            className="mb-6 inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-300"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to projects
          </Link>
          <p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-400">
            Shipyard
          </p>
          <h1 className="text-3xl font-semibold tracking-tight">
            Create Project
          </h1>
        </div>

        <form className="space-y-6" onSubmit={handleSubmit}>
          {error ? (
            <div className="rounded-lg border border-red-900 bg-red-950 px-4 py-3 text-sm text-red-400">
              {error}
            </div>
          ) : null}

          <div className="space-y-2">
            <label htmlFor="name" className="block text-sm font-medium text-zinc-300">
              Project Name
            </label>
            <input
              id="name"
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-awesome-site"
              className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-zinc-50 placeholder-zinc-600 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="repoUrl" className="block text-sm font-medium text-zinc-300">
              Repository URL
            </label>
            <input
              id="repoUrl"
              type="url"
              required
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              placeholder="https://github.com/user/repo"
              className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-zinc-50 placeholder-zinc-600 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label htmlFor="buildPreset" className="block text-sm font-medium text-zinc-300">
                Build Preset
              </label>
              <select
                id="buildPreset"
                required
                value={buildPreset}
                onChange={(e) => setBuildPreset(e.target.value)}
                className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-zinc-50 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
              >
                <option value="static-copy">Static Copy</option>
                <option value="next-export">Next.js</option>
                <option value="vite">Vite</option>
                <option value="npm">npm</option>
              </select>
            </div>

            <div className="space-y-2">
              <label htmlFor="outputDir" className="block text-sm font-medium text-zinc-300">
                Output Directory
              </label>
              <input
                id="outputDir"
                type="text"
                required
                value={outputDir}
                onChange={(e) => setOutputDir(e.target.value)}
                placeholder="dist"
                className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-zinc-50 placeholder-zinc-600 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label htmlFor="defaultBranch" className="block text-sm font-medium text-zinc-300">
              Default Branch
            </label>
            <input
              id="defaultBranch"
              type="text"
              required
              value={defaultBranch}
              onChange={(e) => setDefaultBranch(e.target.value)}
              placeholder="main"
              className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-zinc-50 placeholder-zinc-600 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
            />
          </div>

          <div className="flex gap-4 pt-4">
            <Button
              type="submit"
              disabled={isSubmitting}
              className="gap-2"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Project"
              )}
            </Button>
            <Button
              type="button"
              variant="outline"
              asChild
            >
              <Link href="/projects">Cancel</Link>
            </Button>
          </div>
        </form>
      </div>
    </main>
  );
}