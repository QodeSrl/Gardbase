import React from "react";
import { LuTriangleAlert, LuShieldCheck, LuArrowRight } from "react-icons/lu";

const traditional = [
  "Stores your data in plaintext — one breach exposes everything",
  "Operators, infra, and backups can all read sensitive records",
  "Compliance bolted on as config you must get right, every time",
  "Trust is a promise, not something you can verify",
];

const gardbase = [
  "Data is encrypted client-side — the server only holds ciphertext",
  "Keys are unwrapped only inside attested AWS Nitro Enclaves",
  "Searchable indexes work directly on encrypted data",
  "Zero-trust by design — and you can cryptographically verify it",
];

const ProblemStatementSection: React.FC = () => {
  return (
    <section id="why" className="relative py-20 sm:py-28">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto mb-14 max-w-3xl text-center sm:mb-16">
          <span className="text-sm font-semibold uppercase tracking-widest text-accent-2">
            The purpose
          </span>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl md:text-5xl">
            Encryption shouldn&apos;t be an afterthought
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-base text-muted sm:text-lg">
            Most databases are built to read your data. Gardbase is built so it never can. The
            difference is architectural — and it&apos;s why the entire engine is open source for you
            to audit.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2 lg:gap-8">
          {/* Traditional */}
          <div className="glass relative overflow-hidden rounded-2xl p-7 sm:p-8">
            <div className="absolute right-6 top-6 h-24 w-24 rounded-full bg-red-500/10 blur-3xl" />
            <div className="mb-5 inline-flex items-center gap-2 rounded-lg bg-red-500/10 px-3 py-1.5 text-sm font-medium text-red-500 dark:text-red-300">
              <LuTriangleAlert className="h-4 w-4" />
              Traditional databases
            </div>
            <ul className="space-y-3">
              {traditional.map(item => (
                <li key={item} className="flex items-start gap-3 text-sm text-muted sm:text-base">
                  <span className="mt-2 h-1.5 w-1.5 flex-shrink-0 rounded-full bg-red-400/70" />
                  {item}
                </li>
              ))}
            </ul>
          </div>

          {/* Gardbase */}
          <div className="ring-glow relative overflow-hidden rounded-2xl border border-accent/30 bg-gradient-to-b from-accent/10 to-transparent p-7 sm:p-8">
            <div className="absolute right-6 top-6 h-24 w-24 rounded-full bg-accent/20 blur-3xl" />
            <div className="mb-5 inline-flex items-center gap-2 rounded-lg bg-accent/15 px-3 py-1.5 text-sm font-medium text-accent">
              <LuShieldCheck className="h-4 w-4" />
              The Gardbase way
            </div>
            <ul className="space-y-3">
              {gardbase.map(item => (
                <li key={item} className="flex items-start gap-3 text-sm text-fg sm:text-base">
                  <LuArrowRight className="mt-1 h-4 w-4 flex-shrink-0 text-accent-2" />
                  {item}
                </li>
              ))}
            </ul>
          </div>
        </div>

        <p className="mx-auto mt-10 max-w-2xl text-center text-sm text-subtle sm:text-base">
          Built for teams in <span className="text-fg">healthcare</span>,{" "}
          <span className="text-fg">finance</span>, and anywhere data confidentiality has to be
          provable — not just promised.
        </p>
      </div>
    </section>
  );
};

export default ProblemStatementSection;
