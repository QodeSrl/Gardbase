import {
  LuArrowRight,
  LuArrowUpRight,
  LuDatabase,
  LuFingerprint,
  LuLaptop,
  LuLockKeyhole,
  LuServer,
} from "react-icons/lu";
import { site } from "@/lib/site";

const steps = [
  {
    n: "01",
    title: "Verify the enclave",
    text: "Establish a session and verify the signed attestation, expected code measurements, nonce, and public-key binding.",
  },
  {
    n: "02",
    title: "Encrypt in your app",
    text: "Receive session-sealed key material and encrypt object contents locally with a per-object data encryption key.",
  },
  {
    n: "03",
    title: "Store and query",
    text: "Send ciphertext and index tokens to the API. Retrieve encrypted records, then decrypt them in your application.",
  },
];

export default function HowItWorksSection() {
  return (
    <section id="how-it-works" className="section architecture-section">
      <div className="container">
        <div className="section-heading">
          <div>
            <p className="eyebrow">02 / THE ARCHITECTURE</p>
            <h2>
              Clear boundaries.
              <br />
              From client to storage.
            </h2>
          </div>
          <p>
            Object contents and key operations follow separate paths. Each component has a specific
            job.
          </p>
        </div>
        <div
          className="architecture-map"
          aria-label="Architecture: the application sends ciphertext through the API to storage. The API relays key operations to an isolated Nitro Enclave backed by KMS."
        >
          <div className="architecture-flow">
            <div className="architecture-node client-node">
              <LuLaptop />
              <span className="mono">YOUR APPLICATION</span>
              <h3>Plaintext lives here.</h3>
              <p>
                Encrypt and decrypt objects.
                <br />
                Generate query tokens.
              </p>
              <span className="node-tag">Client trust boundary</span>
            </div>
            <div className="flow-connector">
              <span>ciphertext + tokens</span>
              <span className="flow-line" aria-hidden="true">
                <LuArrowRight />
              </span>
            </div>
            <div className="architecture-node">
              <LuServer />
              <span className="mono">GARDBASE API</span>
              <h3>Route. Authorize. Store.</h3>
              <p>
                Tenant permissions.
                <br />
                Object and query endpoints.
              </p>
              <span className="node-tag">EC2 parent instance</span>
            </div>
            <div className="flow-connector">
              <span>encrypted objects</span>
              <span className="flow-line" aria-hidden="true">
                <LuArrowRight />
              </span>
            </div>
            <div className="architecture-node">
              <LuDatabase />
              <span className="mono">DYNAMODB + S3</span>
              <h3>Ciphertext at rest.</h3>
              <p>
                Inline objects and indexes.
                <br />
                Large encrypted blobs.
              </p>
              <span className="node-tag">AWS storage</span>
            </div>
          </div>
          <div className="enclave-connection">
            <span className="mono">KEY OPERATIONS VIA VSOCK</span>
            <span className="enclave-line" aria-hidden="true" />
          </div>
          <div className="enclave-node">
            <div className="enclave-icon">
              <LuFingerprint />
            </div>
            <div>
              <span className="mono">AWS NITRO ENCLAVE</span>
              <h3>Attested, isolated key operations</h3>
              <p>KMS recipient encryption · session-sealed keys · no persistent storage</p>
            </div>
            <LuLockKeyhole className="enclave-lock" />
          </div>
        </div>
        <ol className="steps">
          {steps.map(step => (
            <li key={step.n}>
              <span className="step-number">{step.n}</span>
              <h3>{step.title}</h3>
              <p>{step.text}</p>
            </li>
          ))}
        </ol>
        <div className="architecture-note">
          <span>
            Encrypted indexes disclose patterns: equality tokens reveal repeated values, and range
            indexes reveal order. Choose indexes to match your threat model.
          </span>
          <a href={site.security} className="text-link">
            Review the tradeoffs <LuArrowUpRight />
          </a>
        </div>
      </div>
    </section>
  );
}
