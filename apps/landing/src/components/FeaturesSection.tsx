import { LuDatabase, LuGitBranch, LuKeyRound, LuSearch } from "react-icons/lu";

const features = [
  {
    n: "01",
    icon: LuSearch,
    title: "Query encrypted indexes",
    text: "Equality lookups use deterministic tokens. Range queries use order-preserving tokens for fields where revealing order is acceptable.",
    detail: "Equality · Ranges · Pagination",
  },
  {
    n: "02",
    icon: LuDatabase,
    title: "One object model. Two storage paths.",
    text: "Keep small encrypted objects inline in DynamoDB. Upload larger payloads directly to S3 using presigned URLs, then confirm their metadata through the API.",
    detail: "DynamoDB · S3 · Presigned uploads",
  },
  {
    n: "03",
    icon: LuKeyRound,
    title: "Keys with defined responsibilities",
    text: "KMS, tenant master keys, per-table index keys, and per-object data keys separate storage access from cryptographic operations.",
    detail: "Tenant isolation · Session-sealed keys",
  },
  {
    n: "04",
    icon: LuGitBranch,
    title: "Build on explicit primitives",
    text: "Use the REST API and shared Go types for versioned writes, soft deletion, recovery, and tenant-scoped read, write, and crypto permissions.",
    detail: "REST endpoints · Go crypto packages",
  },
];

export default function FeaturesSection() {
  return (
    <section id="features" className="section capabilities-section">
      <div className="container">
        <div className="section-heading">
          <div>
            <p className="eyebrow">03 / The building blocks</p>
            <h2>
              Useful storage.
              <br />
              Deliberate encryption.
            </h2>
          </div>
          <p>The pieces to build an encrypted data workflow on infrastructure you control.</p>
        </div>
        <div className="feature-grid">
          {features.map(feature => (
            <article key={feature.n} className="feature-card">
              <div className="feature-card-top">
                <span className="feature-icon">
                  <feature.icon aria-hidden="true" />
                </span>
                <span className="feature-number">{feature.n}</span>
              </div>
              <h3>{feature.title}</h3>
              <p>{feature.text}</p>
              <ul className="feature-tags" aria-label="Includes">
                {feature.detail.split(" · ").map(tag => (
                  <li key={tag}>{tag}</li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
