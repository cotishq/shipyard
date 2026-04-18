export default function ProjectsPage() {
  return (
    <main className="min-h-screen bg-slate-950 px-6 py-16 text-slate-50">
      <div className="mx-auto max-w-5xl">
        <p className="mb-3 text-xs font-semibold uppercase tracking-[0.24em] text-emerald-300">
          Shipyard
        </p>
        <h1 className="text-4xl font-semibold tracking-tight">Projects</h1>
        <p className="mt-4 max-w-2xl text-base leading-7 text-slate-300">
          Frontend foundation is in place. Next step is wiring this page to the
          real `GET /projects` API and adding project creation.
        </p>
      </div>
    </main>
  );
}
