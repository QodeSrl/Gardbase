import React from "react";
import { LuArrowRight, LuLock, LuShieldCheck, LuBookOpen } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";

const trust = [
  { icon: <LuLock className="h-4 w-4" />, label: "Client-side AES-256-GCM" },
  { icon: <LuShieldCheck className="h-4 w-4" />, label: "AWS Nitro Enclave attestation" },
  { icon: <SiGithub className="h-4 w-4" />, label: `${site.license} licensed` },
];

const HeroSection: React.FC = () => {
  return (
    <section className="relative overflow-hidden pb-16 pt-20 sm:pb-24 sm:pt-28">
      {/* Backdrop */}
      <div className="bg-grid mask-fade absolute inset-0 -z-10" aria-hidden />
      <div
        className="animate-pulse-glow absolute left-1/2 top-[-10%] -z-10 h-[420px] w-[640px] -translate-x-1/2 rounded-full bg-accent/25 blur-[120px]"
        aria-hidden
      />
      <div
        className="absolute right-[-5%] top-[30%] -z-10 h-[320px] w-[320px] rounded-full bg-accent-3/20 blur-[120px]"
        aria-hidden
      />

      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl text-center">
          <a
            href={site.repo}
            target="_blank"
            rel="noreferrer"
            className="animate-fade-up glass inline-flex items-center gap-2 rounded-full px-4 py-1.5 text-xs font-medium text-muted transition-colors hover:border-fg/30 sm:text-sm"
          >
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-2 opacity-75" />
              <span className="relative inline-flex h-2 w-2 rounded-full bg-accent-2" />
            </span>
            Open source · {site.license} · Self-host on your own AWS
          </a>

          <h1 className="animate-fade-up mt-6 text-4xl font-extrabold leading-[1.05] tracking-tight sm:text-5xl md:text-6xl lg:text-7xl">
            <span className="text-fg">The database that</span>
            <span className="text-gradient animate-gradient block">can&apos;t read your data</span>
          </h1>

          <p className="animate-fade-up mx-auto mt-6 max-w-2xl text-base text-muted sm:text-lg md:text-xl">
            Gardbase is an open-source, zero-trust NoSQL database. Records are encrypted inside your
            own application, and the keys exist only within hardware-isolated AWS Nitro Enclaves — so
            the server stores ciphertext and nothing else. A breach walks away with{" "}
            <strong className="text-fg">data it can never decrypt</strong>.
          </p>

          <div className="animate-fade-up mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
            <a
              href={site.repo}
              target="_blank"
              rel="noreferrer"
              className="group inline-flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-accent to-accent-3 px-7 py-3.5 text-base font-semibold text-white shadow-xl shadow-accent/30 transition-transform hover:scale-[1.03] sm:w-auto"
            >
              <SiGithub className="h-5 w-5" />
              Star on GitHub
              <LuArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </a>
            <a
              href={site.docs}
              target="_blank"
              rel="noreferrer"
              className="glass inline-flex w-full items-center justify-center gap-2 rounded-xl px-7 py-3.5 text-base font-semibold text-fg transition-colors hover:border-fg/30 sm:w-auto"
            >
              <LuBookOpen className="h-5 w-5" />
              Read the docs
            </a>
          </div>

          <div className="animate-fade-up mt-8 flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-subtle sm:text-sm">
            {trust.map(t => (
              <span key={t.label} className="inline-flex items-center gap-2">
                <span className="text-accent-2">{t.icon}</span>
                {t.label}
              </span>
            ))}
          </div>
        </div>

        {/* Code window (intentionally dark in both themes) */}
        <div className="animate-fade-up mt-14 sm:mt-20">
          <div className="ring-glow mx-auto max-w-3xl overflow-hidden rounded-2xl border border-white/10 bg-surface/90 backdrop-blur-xl">
            <div className="flex items-center gap-2 border-b border-white/10 bg-white/5 px-4 py-3">
              <span className="h-3 w-3 rounded-full bg-red-400/80" />
              <span className="h-3 w-3 rounded-full bg-amber-400/80" />
              <span className="h-3 w-3 rounded-full bg-green-400/80" />
              <span className="ml-2 font-mono text-xs text-slate-400">main.go</span>
            </div>
            <pre className="overflow-x-auto p-5 font-mono text-[13px] leading-relaxed text-slate-200 sm:p-6 sm:text-sm">
              <code>
                <span className="text-slate-500">
                  {"// Encrypted on your side — before it ever leaves your app"}
                </span>
                {"\n"}
                <span className="text-violet-300">type</span>{" "}
                <span className="text-cyan-300">Patient</span>{" "}
                <span className="text-violet-300">struct</span> {"{\n"}
                {"  "}Email <span className="text-cyan-300">string</span>{" "}
                <span className="text-amber-300">`gardbase:&quot;index&quot;`</span>{" "}
                <span className="text-slate-500">{"// searchable, encrypted"}</span>
                {"\n  "}Record <span className="text-cyan-300">string</span>{" "}
                <span className="text-amber-300">`gardbase:&quot;encrypt&quot;`</span>
                {"\n}\n\n"}
                patients := gardbase.<span className="text-blue-300">Collection</span>[Patient](db)
                {"\n"}
                patients.<span className="text-blue-300">Insert</span>(ctx, Patient{"{"}Email:{" "}
                <span className="text-emerald-300">&quot;ada@corp.io&quot;</span>, Record:{" "}
                <span className="text-emerald-300">&quot;confidential&quot;</span>
                {"})\n\n"}
                <span className="text-slate-500">
                  {"// Query the encrypted index — server only ever sees ciphertext"}
                </span>
                {"\n"}
                found, _ := patients.<span className="text-blue-300">Where</span>(
                <span className="text-emerald-300">&quot;Email&quot;</span>,{" "}
                <span className="text-emerald-300">&quot;ada@corp.io&quot;</span>).
                <span className="text-blue-300">Find</span>(ctx)
              </code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
};

export default HeroSection;
