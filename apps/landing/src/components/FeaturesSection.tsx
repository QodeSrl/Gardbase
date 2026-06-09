import React from "react";
import {
  LuLock,
  LuShieldCheck,
  LuSearch,
  LuDatabase,
  LuKeyRound,
  LuGitBranch,
} from "react-icons/lu";

const features = [
  {
    icon: LuLock,
    title: "Client-side encryption",
    desc: "Every object is sealed with a unique AES-256-GCM data key before it leaves your app. The backend only ever stores ciphertext.",
  },
  {
    icon: LuShieldCheck,
    title: "Attested Nitro Enclaves",
    desc: "Key operations run in hardware-isolated AWS Nitro Enclaves. Clients verify a signed attestation (PCRs, COSE, AWS root CA) before trusting them.",
  },
  {
    icon: LuSearch,
    title: "Searchable encryption",
    desc: "Query encrypted data directly. Deterministic tokens power equality lookups; order-preserving encryption enables sorting and ranges.",
  },
  {
    icon: LuKeyRound,
    title: "4-level key hierarchy",
    desc: "KMS → tenant master key → per-object DEKs → your data. Envelope encryption means rotation without re-encrypting everything.",
  },
  {
    icon: LuDatabase,
    title: "Hybrid DynamoDB + S3",
    desc: "Small encrypted objects live inline in DynamoDB; large blobs go to S3 with wrapped DEKs in metadata. Scalable by default.",
  },
  {
    icon: LuGitBranch,
    title: "Type-safe Go SDK",
    desc: "An ORM-like API built on Go generics, inspired by Mongoose and GORM — plus optimistic locking for safe concurrent writes.",
  },
];

const FeaturesSection: React.FC = () => {
  return (
    <section id="features" className="relative py-20 sm:py-28">
      <div
        className="absolute left-[10%] top-1/3 -z-10 h-72 w-72 rounded-full bg-accent-3/10 blur-[120px]"
        aria-hidden
      />
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto mb-14 max-w-3xl text-center sm:mb-16">
          <span className="text-sm font-semibold uppercase tracking-widest text-accent-2">
            Capabilities
          </span>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl md:text-5xl">
            Real encryption, real database ergonomics
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-base text-muted sm:text-lg">
            Zero-trust security usually means giving up querying, sorting, and developer experience.
            Gardbase keeps all three.
          </p>
        </div>

        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {features.map(f => (
            <div
              key={f.title}
              className="glass group relative overflow-hidden rounded-2xl p-6 transition-all duration-300 hover:-translate-y-1 hover:border-accent/40"
            >
              <div className="absolute inset-x-0 -top-px h-px bg-gradient-to-r from-transparent via-accent/50 to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
              <div className="mb-4 inline-flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-accent/25 to-accent-3/20 text-accent-2 ring-1 ring-fg/10">
                <f.icon className="h-5 w-5" />
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fg">{f.title}</h3>
              <p className="text-sm leading-relaxed text-muted">{f.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default FeaturesSection;
