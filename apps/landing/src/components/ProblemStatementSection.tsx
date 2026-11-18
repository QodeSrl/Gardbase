import { LuArrowUpRight, LuCheck, LuMinus } from "react-icons/lu";
import { site } from "@/lib/site";

const rows = [
  {
    label: "Objects are encrypted",
    atRest: "By the storage service",
    gardbase: "In your application",
  },
  { label: "The database returns", atRest: "Readable records", gardbase: "Ciphertext" },
  {
    label: "Key operations run",
    atRest: "Inside the storage provider",
    gardbase: "In an attested Nitro Enclave",
  },
];

export default function ProblemStatementSection() {
  return (
    <section id="why" className="section intro-section">
      <div className="intro-layout container">
        <div className="intro-copy">
          <p className="eyebrow">01 / The idea</p>
          <h2>Make confidentiality part of the architecture.</h2>
          <p>
            Encryption at rest protects stored bytes. An application that needs readable records can
            still give its database a path to plaintext.
          </p>
          <p>
            Gardbase moves object encryption into your application and key operations into an
            isolated enclave. The API handles encrypted payloads and query tokens, while your
            application controls when records become readable.
          </p>
          <a className="text-link" href={site.security}>
            Read the security model <LuArrowUpRight />
          </a>
        </div>
        <div
          className="compare"
          role="table"
          aria-label="Encryption at rest compared with Gardbase"
        >
          <div className="compare-head" role="row">
            <span role="columnheader" />
            <span role="columnheader">Encryption at rest</span>
            <span role="columnheader" className="compare-ours">
              With Gardbase
            </span>
          </div>
          {rows.map(row => (
            <div className="compare-row" role="row" key={row.label}>
              <span role="rowheader">{row.label}</span>
              <span role="cell">
                <LuMinus aria-hidden="true" />
                {row.atRest}
              </span>
              <span role="cell" className="compare-ours">
                <LuCheck aria-hidden="true" />
                {row.gardbase}
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
