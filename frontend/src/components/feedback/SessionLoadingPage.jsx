export const SessionLoadingPage = () => (
  <main className="grid min-h-screen place-items-center bg-white px-6 text-slate-900">
    <div role="status" aria-live="polite" className="text-center">
      <span
        aria-hidden="true"
        className="mx-auto block size-9 animate-spin rounded-full border-2 border-cyan-300 border-t-transparent motion-reduce:animate-none"
      />
      <p className="mt-4 text-sm text-slate-600">Memeriksa sesi GSNpeeps…</p>
    </div>
  </main>
);

