"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

import { apiFetch } from "@/lib/api";
import { readToken, writeToken } from "@/lib/token";
import type { Project } from "@/lib/types";

export function TokenGate() {
  const router = useRouter();
  const [token, setToken] = useState(() => readToken());
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

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
    <main className="flex min-h-screen items-center justify-center bg-[radial-gradient(circle_at_top_left,_rgba(76,175,80,0.14),_transparent_30%),linear-gradient(180deg,_#f7faf7_0%,_#edf4ee_100%)] px-6 py-16 text-slate-900">
      <div className="w-full max-w-xl rounded-[28px] border border-slate-200/80 bg-white/90 p-8 shadow-[0_24px_80px_rgba(32,53,38,0.12)] backdrop-blur">
        <div className="mb-8">
          <p className="mb-3 inline-flex rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-emerald-700">
            Shipyard Control Plane
          </p>
          <h1 className="text-4xl font-semibold tracking-tight text-slate-950">
            Enter your API token
          </h1>
          <p className="mt-3 max-w-lg text-base leading-7 text-slate-600">
            This frontend talks directly to your Shipyard API. The first step is a
            thin token gate, then the projects and deployment views sit behind it.
          </p>
        </div>

        <form className="space-y-5" onSubmit={handleSubmit}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-700">
              API token
            </span>
            <input
              className="w-full rounded-2xl border border-slate-300 bg-white px-4 py-3 font-mono text-sm text-slate-900 outline-none transition focus:border-emerald-500 focus:ring-4 focus:ring-emerald-100"
              type="password"
              value={token}
              onChange={(event) => setToken(event.target.value)}
              placeholder="shipyard_..."
              autoComplete="off"
              spellCheck={false}
            />
          </label>

          {error ? (
            <p className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
              {error}
            </p>
          ) : null}

          <button
            className="inline-flex w-full items-center justify-center rounded-2xl bg-slate-950 px-4 py-3 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
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
