import React from "react";
import { LuScale, LuEye, LuGitPullRequestArrow, LuServer, LuArrowRight } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";

const points = [
  {
    icon: LuEye,
    title: "Auditable by design",
    desc: "Security you can read line by line. The crypto, the enclave service, and the API are all in the open.",
  },
  {
    icon: LuServer,
    title: "Self-host on your AWS",
    desc: "Deploy the whole stack into your own account with Terraform. Your keys, your infrastructure, your control.",
  },
  {
    icon: LuScale,
    title: `${site.license} licensed`,
    desc: "Permissive licensing for commercial and personal use. No lock-in, no surprise relicensing.",
  },
  {
    icon: LuGitPullRequestArrow,
    title: "Built in the open",
    desc: "Issues, pull requests, and a roadmap you can shape. Contributions are welcome and reviewed.",
  },
];

const OpenSourceSection: React.FC = () => {
  return (
    <section id="open-source" className="relative py-20 sm:py-28">
      <div
        className="absolute right-[5%] top-1/4 -z-10 h-72 w-72 rounded-full bg-accent/15 blur-[120px]"
        aria-hidden
      />
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-16">
          <div>
            <span className="text-sm font-semibold uppercase tracking-widest text-accent-2">
              Open source
            </span>
            <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl md:text-5xl">
              Don&apos;t trust us. <span className="text-gradient">Read the code.</span>
            </h2>
            <p className="mt-4 max-w-xl text-base text-muted sm:text-lg">
              A database that claims to be unbreachable should be the easiest one in the world to
              verify. Gardbase&apos;s entire engine — encryption, enclave attestation, and storage —
              is open source under {site.license}.
            </p>

            <div className="mt-8 grid gap-4 sm:grid-cols-2">
              {points.map(p => (
                <div key={p.title} className="flex gap-3">
                  <div className="mt-0.5 inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-accent/15 text-accent-2 ring-1 ring-fg/10">
                    <p.icon className="h-4 w-4" />
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-fg">{p.title}</h3>
                    <p className="mt-1 text-sm text-subtle">{p.desc}</p>
                  </div>
                </div>
              ))}
            </div>

            <a
              href={site.repo}
              target="_blank"
              rel="noreferrer"
              className="group mt-8 inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-accent to-accent-3 px-6 py-3 text-base font-semibold text-white shadow-xl shadow-accent/30 transition-transform hover:scale-[1.03]"
            >
              <SiGithub className="h-5 w-5" />
              Explore {site.repoLabel}
              <LuArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </a>
          </div>

          {/* Terminal (intentionally dark in both themes) */}
          <div className="ring-glow overflow-hidden rounded-2xl border border-white/10 bg-surface/90 backdrop-blur-xl">
            <div className="flex items-center gap-2 border-b border-white/10 bg-white/5 px-4 py-3">
              <span className="h-3 w-3 rounded-full bg-red-400/80" />
              <span className="h-3 w-3 rounded-full bg-amber-400/80" />
              <span className="h-3 w-3 rounded-full bg-green-400/80" />
              <span className="ml-2 font-mono text-xs text-slate-400">~/gardbase</span>
            </div>
            <pre className="overflow-x-auto p-5 font-mono text-[13px] leading-relaxed text-slate-200 sm:p-6 sm:text-sm">
              <code>
                <span className="text-accent-2">$</span> git clone {site.repo}
                {"\n"}
                <span className="text-slate-500">Cloning into &apos;Gardbase&apos;...</span>
                {"\n\n"}
                <span className="text-accent-2">$</span> tree -L 1 apps pkg
                {"\n"}
                <span className="text-blue-300">apps/</span>
                {"\n"}
                {"├─ "}api<span className="text-slate-500">{"            // Go API server (EC2)"}</span>
                {"\n"}
                {"└─ "}enclave-service{" "}
                <span className="text-slate-500">{"// runs in a Nitro Enclave"}</span>
                {"\n"}
                <span className="text-blue-300">pkg/</span>
                {"\n"}
                {"├─ "}crypto{" "}
                <span className="text-slate-500">{"        // client-side crypto + attestation"}</span>
                {"\n"}
                {"├─ "}enclaveproto{" "}
                <span className="text-slate-500">{"  // parent⇄enclave protocol"}</span>
                {"\n"}
                {"└─ "}models{" "}
                <span className="text-slate-500">{"        // objects, indexes, tenants"}</span>
                {"\n\n"}
                <span className="text-accent-2">$</span>{" "}
                <span className="text-emerald-300">cat LICENSE | head -2</span>
                {"\n"}
                Apache License, Version 2.0
              </code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
};

export default OpenSourceSection;
