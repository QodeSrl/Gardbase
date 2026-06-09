import React from "react";
import { LuFingerprint, LuLockKeyhole, LuServer } from "react-icons/lu";

const steps = [
  {
    n: "01",
    icon: LuFingerprint,
    title: "Attest the enclave",
    text: "Your client opens a session and receives a signed attestation document. It verifies the AWS certificate chain, COSE signatures, and PCR code measurements — proving genuine Gardbase code is running before any trust is extended.",
  },
  {
    n: "02",
    icon: LuLockKeyhole,
    title: "Encrypt client-side",
    text: "A fresh data encryption key is unwrapped inside the enclave and sealed to your session. Your app encrypts each object with AES-256-GCM locally. Plaintext and master keys never touch the server.",
  },
  {
    n: "03",
    icon: LuServer,
    title: "Store & query encrypted",
    text: "Ciphertext lands in DynamoDB or S3 with envelope-wrapped keys. Encrypted index tokens, generated in the enclave, let you run equality and range queries — without ever decrypting on the backend.",
  },
];

const hierarchy = [
  { k: "AWS KMS Key", d: "per environment" },
  { k: "Tenant Master Key", d: "decrypted only inside the enclave" },
  { k: "Data Encryption Keys", d: "fresh, per object" },
  { k: "Your Data", d: "AES-256-GCM ciphertext" },
];

const HowItWorksSection: React.FC = () => {
  return (
    <section id="how-it-works" className="relative py-20 sm:py-28">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto mb-14 max-w-3xl text-center sm:mb-16">
          <span className="text-sm font-semibold uppercase tracking-widest text-accent-2">
            How it works
          </span>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl md:text-5xl">
            Trust you can verify, not just assume
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-base text-muted sm:text-lg">
            Three steps from your application to provably confidential storage.
          </p>
        </div>

        <div className="relative grid gap-6 md:grid-cols-3 lg:gap-8">
          <div
            className="absolute left-0 right-0 top-[3.25rem] hidden h-px bg-gradient-to-r from-transparent via-accent/30 to-transparent md:block"
            aria-hidden
          />
          {steps.map(s => (
            <div key={s.n} className="glass relative rounded-2xl p-7">
              <div className="mb-5 flex items-center justify-between">
                <div className="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-accent to-accent-3 text-white shadow-lg shadow-accent/30">
                  <s.icon className="h-6 w-6" />
                </div>
                <span className="font-mono text-3xl font-bold text-fg/10">{s.n}</span>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fg sm:text-xl">{s.title}</h3>
              <p className="text-sm leading-relaxed text-muted">{s.text}</p>
            </div>
          ))}
        </div>

        {/* Key hierarchy */}
        <div className="glass mx-auto mt-12 max-w-3xl rounded-2xl p-6 sm:p-8">
          <p className="mb-4 text-center text-sm font-medium uppercase tracking-widest text-subtle">
            4-level key hierarchy
          </p>
          <div className="flex flex-col items-center gap-2 font-mono text-xs text-muted sm:text-sm">
            {hierarchy.map((row, i) => (
              <React.Fragment key={row.k}>
                <div className="flex w-full max-w-md items-center justify-between rounded-lg border border-line bg-fg/5 px-4 py-2.5">
                  <span className="font-semibold text-fg">{row.k}</span>
                  <span className="text-subtle">{row.d}</span>
                </div>
                {i < hierarchy.length - 1 && <span className="text-accent-2">↓</span>}
              </React.Fragment>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
};

export default HowItWorksSection;
