import React from "react";
import { LuArrowRight, LuBookOpen } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";

const CTASection: React.FC = () => {
  return (
    <section className="relative py-20 sm:py-28">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Intentionally a dark spotlight panel in both themes */}
        <div className="relative overflow-hidden rounded-3xl border border-white/10 bg-gradient-to-br from-brand via-surface-2 to-ink px-6 py-14 text-center sm:px-12 sm:py-20">
          <div className="bg-grid mask-fade absolute inset-0 opacity-60" aria-hidden />
          <div
            className="animate-pulse-glow absolute left-1/2 top-0 h-64 w-[480px] -translate-x-1/2 rounded-full bg-accent/30 blur-[120px]"
            aria-hidden
          />
          <div className="relative">
            <h2 className="mx-auto max-w-2xl text-3xl font-bold tracking-tight text-white sm:text-4xl md:text-5xl">
              Build on a database that{" "}
              <span className="text-gradient-bright">can&apos;t betray you</span>
            </h2>
            <p className="mx-auto mt-4 max-w-xl text-base text-slate-300 sm:text-lg">
              Clone it, audit it, deploy it on your own AWS — all open source. Star the repo to
              follow along as the managed cloud takes shape.
            </p>
            <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
              <a
                href={site.repo}
                target="_blank"
                rel="noreferrer"
                className="group inline-flex w-full items-center justify-center gap-2 rounded-xl bg-white px-7 py-3.5 text-base font-semibold text-brand transition-transform hover:scale-[1.03] sm:w-auto"
              >
                <SiGithub className="h-5 w-5" />
                Star on GitHub
                <LuArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
              </a>
              <a
                href={site.docs}
                target="_blank"
                rel="noreferrer"
                className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-white/20 bg-white/10 px-7 py-3.5 text-base font-semibold text-white backdrop-blur-sm transition-colors hover:bg-white/15 sm:w-auto"
              >
                <LuBookOpen className="h-5 w-5" />
                Read the docs
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default CTASection;
