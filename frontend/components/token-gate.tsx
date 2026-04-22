"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

import { apiFetch } from "@/lib/api";
import { readToken, writeToken } from "@/lib/token";
import type { Project } from "@/lib/types";
import { Eye, EyeOff } from "lucide-react";

export function TokenGate() {
  const router = useRouter();
  const [token, setToken] = useState(() => readToken());
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showToken, setShowToken] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmed = token.trim();
    if (!trimmed) {
      setError("API token is required");
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      await apiFetch<Project[]>("/projects", { token: trimmed });
      writeToken(trimmed);
      router.push("/projects");
    } catch (submitError) {
      setError(
        submitError instanceof Error ? submitError.message : "Authentication failed",
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#09090b] px-6 py-16 text-zinc-50">
      <div className="w-full max-w-xl rounded-[28px] border border-zinc-800 bg-zinc-900/50 p-8">
        <div className="mb-8">
          <p className="mb-3 inline-flex rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-emerald-400">
            Shipyard Control Plane
          </p>
          <h1 className="text-4xl font-semibold tracking-tight text-zinc-50">
            Enter your API token
          </h1>
        </div>

        <form className="space-y-5" onSubmit={handleSubmit}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-zinc-300">
              API token
            </span>
            <div className="relative">
              <input
                className="w-full rounded-2xl border border-zinc-800 bg-zinc-900/80 px-4 py-3 pr-12 font-mono text-sm text-zinc-50 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                type={showToken ? "text" : "password"}
                value={token}
                onChange={(event) => setToken(event.target.value)}
                placeholder="shipyard_..."
                autoComplete="off"
                spellCheck={false}
              />
              <button
                type="button"
                onClick={() => setShowToken(!showToken)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300"
              >
                {showToken ? (
                  <EyeOff className="h-5 w-5" />
                ) : (
                  <Eye className="h-5 w-5" />
                )}
              </button>
            </div>
          </label>

          {error ? (
            <p className="rounded-2xl border border-red-900 bg-red-950/50 px-4 py-3 text-sm text-red-400">
              {error}
            </p>
          ) : null}

          <button
            className="inline-flex w-full items-center justify-center rounded-2xl bg-emerald-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
            type="submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Verifying token..." : "Continue to projects"}
          </button>
        </form>
      </div>
    </main>
  );
}
