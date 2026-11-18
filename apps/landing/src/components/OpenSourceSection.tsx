import { useEffect, useRef, useState } from "react";
import { LuArrowUpRight, LuCheck, LuCopy } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";
import { highlight } from "@/lib/highlight";

const cloneCommand = `git clone --branch development ${site.repo}.git`;
const examples = [
  {
    label: "Encrypt an object",
    file: "encrypt.go",
    description:
      "A real function from pkg/crypto. Supply a 32-byte data key obtained through your verified enclave session.",
    source: `${site.source}/pkg/crypto/probabilistic.go`,
    code: `package example\n\nimport (\n    gardcrypto "github.com/qodesrl/gardbase/pkg/crypto"\n)\n\nfunc encrypt(plaintext, dek []byte) ([]byte, error) {\n    return gardcrypto.EncryptObjectProbabilistic(\n        plaintext,\n        dek,\n    )\n}\n\n// AES-256-GCM: nonce || encrypted content || tag\n// Send the encrypted bytes to the object API.`,
  },
  {
    label: "Query an index",
    file: "query.http",
    description:
      "Request template: replace the tenant, key, table hash, and base64 token with values from your integration. range_op: 0 is an equality query.",
    source: `${site.source}/pkg/api/objects/requests.go`,
    code: `POST /api/objects/query\nContent-Type: application/json\nX-Tenant-ID: <tenant-id>\nX-API-Key: <api-key>\n\n{\n  "table_hash": "<table-hash>",\n  "index": {\n    "name": { "hash_field": "customer" },\n    "token_hash": "<base64-token>"\n  },\n  "range_op": 0,\n  "limit": 20\n}`,
  },
];

export default function OpenSourceSection() {
  const [selected, setSelected] = useState(0);
  const [copyStatus, setCopyStatus] = useState("");
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const example = examples[selected] ?? examples[0]!;

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    []
  );

  async function copyCommand() {
    try {
      await navigator.clipboard.writeText(cloneCommand);
      setCopyStatus("Copied to clipboard");
    } catch {
      setCopyStatus("Select and copy the command below");
    }
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopyStatus(""), 3500);
  }

  return (
    <section id="open-source" className="section developer-section">
      <div className="container">
        <div className="section-heading">
          <div>
            <p className="eyebrow">04 / BUILT IN THE OPEN</p>
            <h2>
              Start with the code.
              <br />
              Understand the system.
            </h2>
          </div>
          <a className="text-link" href={site.repo}>
            <SiGithub /> {site.repoLabel} <LuArrowUpRight />
          </a>
        </div>
        <div className="developer-layout">
          <div className="developer-copy">
            <p>
              Inspect the API, enclave service, and crypto packages together. Integrate through REST
              and shared Go types, with the security decisions visible in source.
            </p>
            <div className="source-links">
              <a href={site.apiDocs}>
                <span className="mono">apps/api</span>
                <span>
                  API reference <LuArrowUpRight />
                </span>
              </a>
              <a href={`${site.source}/apps/enclave-service/README.md`}>
                <span className="mono">apps/enclave-service</span>
                <span>
                  Enclave service <LuArrowUpRight />
                </span>
              </a>
              <a href={`${site.source}/pkg/crypto/README.md`}>
                <span className="mono">pkg/crypto</span>
                <span>
                  Crypto primitives <LuArrowUpRight />
                </span>
              </a>
            </div>
            <div className="clone-block">
              <div>
                <span className="mono">GET THE SOURCE</span>
                <button
                  className="icon-button"
                  aria-label="Copy clone command"
                  onClick={() => void copyCommand()}
                >
                  {copyStatus === "Copied to clipboard" ? <LuCheck /> : <LuCopy />}
                </button>
              </div>
              <code>{cloneCommand}</code>
              <span className="copy-status" role="status">
                {copyStatus}
              </span>
            </div>
          </div>
          <div className="code-example">
            <div className="example-controls">
              <div className="segmented" role="group" aria-label="Code examples">
                {examples.map((item, i) => (
                  <button
                    key={item.file}
                    onClick={() => setSelected(i)}
                    aria-pressed={selected === i}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
              <span className="mono example-file">{example.file}</span>
            </div>
            <pre tabIndex={0} aria-label={example.file}>
              <code>{highlight(example.code)}</code>
            </pre>
            <div className="code-caption">
              <p>{example.description}</p>
              <a href={example.source} className="text-link">
                View implementation <LuArrowUpRight />
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
