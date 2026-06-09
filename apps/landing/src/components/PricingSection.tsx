import React from "react";
import { LuCheck } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";

type Plan = {
  name: string;
  price: string;
  cadence?: string;
  description: string;
  features: string[];
  cta: { label: string; href: string; icon?: boolean };
  highlight?: boolean;
  badge?: string;
};

const plans: Plan[] = [
  {
    name: "Open Source",
    price: "Free",
    cadence: "forever",
    description: "The full engine, self-hosted on your own AWS account.",
    features: [
      "Every security feature, no gates",
      "Client-side encryption & enclave attestation",
      "Deploy with Terraform on your infrastructure",
      `${site.license} license — yours to keep`,
      "Community support via GitHub",
    ],
    cta: { label: "Get the code", href: site.repo, icon: true },
  },
  {
    name: "Managed Cloud",
    price: "Early access",
    description: "Let us run the enclaves, KMS, and scaling. You ship features.",
    features: [
      "Fully managed Nitro Enclave fleet",
      "Automated key management & rotation",
      "Transparent usage-based pricing",
      "Dashboards, metrics & alerts",
      "Priority support",
    ],
    cta: { label: "Request early access", href: site.companyUrl },
    highlight: true,
    badge: "In development",
  },
  {
    name: "Enterprise",
    price: "Custom",
    description: "For regulated teams with compliance and scale requirements.",
    features: [
      "Dedicated deployment & onboarding",
      "Custom data residency & SLAs",
      "Security review & audit support",
      "Direct line to the engineering team",
    ],
    cta: { label: "Talk to us", href: site.companyUrl },
  },
];

const PricingSection: React.FC = () => {
  return (
    <section id="pricing" className="relative py-20 sm:py-28">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto mb-14 max-w-3xl text-center sm:mb-16">
          <span className="text-sm font-semibold uppercase tracking-widest text-accent-2">
            Pricing
          </span>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl md:text-5xl">
            Free and open. Managed when you&apos;re ready.
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-base text-muted sm:text-lg">
            Start by self-hosting the open-source engine at zero cost. When you&apos;d rather not run
            enclaves yourself, the managed cloud is on the way.
          </p>
        </div>

        <div className="grid items-stretch gap-6 md:grid-cols-3 lg:gap-8">
          {plans.map(plan => (
            <div
              key={plan.name}
              className={`relative flex flex-col rounded-2xl p-7 sm:p-8 ${
                plan.highlight
                  ? "ring-glow border border-accent/40 bg-gradient-to-b from-accent/10 to-transparent"
                  : "glass"
              }`}
            >
              {plan.badge && (
                <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-gradient-to-r from-accent to-accent-3 px-3 py-1 text-xs font-semibold text-white shadow-lg shadow-accent/30">
                  {plan.badge}
                </span>
              )}
              <h3 className="text-lg font-semibold text-fg">{plan.name}</h3>
              <div className="mt-3 flex items-end gap-1.5">
                <span className="text-4xl font-extrabold text-fg">{plan.price}</span>
                {plan.cadence && <span className="pb-1 text-sm text-subtle">{plan.cadence}</span>}
              </div>
              <p className="mt-3 text-sm text-muted">{plan.description}</p>

              <ul className="mt-6 flex-1 space-y-3">
                {plan.features.map(f => (
                  <li key={f} className="flex items-start gap-3 text-sm text-muted">
                    <LuCheck className="mt-0.5 h-4 w-4 flex-shrink-0 text-accent-2" />
                    {f}
                  </li>
                ))}
              </ul>

              <a
                href={plan.cta.href}
                target="_blank"
                rel="noreferrer"
                className={`mt-8 inline-flex items-center justify-center gap-2 rounded-xl px-5 py-3 text-sm font-semibold transition-all ${
                  plan.highlight
                    ? "bg-gradient-to-r from-accent to-accent-3 text-white shadow-lg shadow-accent/30 hover:scale-[1.02]"
                    : "border border-line bg-fg/5 text-fg hover:border-fg/30 hover:bg-fg/10"
                }`}
              >
                {plan.cta.icon && <SiGithub className="h-4 w-4" />}
                {plan.cta.label}
              </a>
            </div>
          ))}
        </div>

        <p className="mt-8 text-center text-xs text-subtle sm:text-sm">
          Managed cloud pricing will be published at launch. Request early access to be the first to know.
        </p>
      </div>
    </section>
  );
};

export default PricingSection;
