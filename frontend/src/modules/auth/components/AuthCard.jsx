import { Link } from "react-router-dom";

export const AuthCard = ({ title, description, children, footer }) => (
  <main className="min-h-screen bg-slate-950 px-5 py-10 text-slate-100 sm:py-16">
    <section className="mx-auto max-w-md" aria-labelledby="auth-title">
      <Link to="/" className="inline-block focus-visible:outline focus-visible:outline-2">
        <span className="text-xs font-semibold uppercase tracking-[0.25em] text-cyan-300">
          HR Information System
        </span>
        <span className="mt-1 block text-2xl font-bold">GSNpeeps</span>
      </Link>
      <div className="mt-8 rounded-2xl border border-white/10 bg-white/[0.04] p-6 shadow-2xl sm:p-8">
        <h1 id="auth-title" className="text-2xl font-bold">
          {title}
        </h1>
        <p className="mt-2 text-sm leading-6 text-slate-400">{description}</p>
        <div className="mt-7">{children}</div>
        {footer ? <div className="mt-6 border-t border-white/10 pt-5">{footer}</div> : null}
      </div>
    </section>
  </main>
);

