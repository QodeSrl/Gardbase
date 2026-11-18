import { useState } from "react";
import { LuArrowDown, LuArrowRight, LuCheck, LuLockKeyhole, LuRotateCcw } from "react-icons/lu";
import { decryptDemoRecord, encryptDemoRecord, type DemoEnvelope } from "@/lib/encryptionDemo";

const initialRecord =
  '{\n  "customer": "Ada",\n  "note": "Keep this between us.",\n  "project": "Something good"\n}';

export default function EncryptionDemo() {
  const [source, setSource] = useState(initialRecord);
  const [envelope, setEnvelope] = useState<DemoEnvelope | null>(null);
  const [revealed, setRevealed] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function encrypt() {
    setBusy(true);
    setError("");
    setRevealed(null);
    setEnvelope(null);
    try {
      setEnvelope(await encryptDemoRecord(source));
    } catch {
      setError(
        "Encryption is unavailable here. Open this page over HTTPS or localhost in a browser with Web Crypto support."
      );
    } finally {
      setBusy(false);
    }
  }

  async function decrypt() {
    if (!envelope) return;
    setBusy(true);
    setError("");
    try {
      setRevealed(await decryptDemoRecord(envelope));
    } catch {
      setError("Could not decrypt this sample. Encrypt the record again to create a new key.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="encryption-demo">
      <div className="demo-title">
        <span className="demo-dots" aria-hidden="true">
          <i />
          <i />
          <i />
        </span>
        <span>An encryption boundary, in action</span>
        <span className="demo-badge">
          <span className="status-dot" /> Local demo
        </span>
      </div>
      <div className="demo-source">
        <label className="panel-label" htmlFor="demo-record">
          <span>01 / Your application</span>
          <span>plaintext</span>
        </label>
        <textarea
          id="demo-record"
          value={source}
          spellCheck={false}
          maxLength={2048}
          disabled={busy}
          aria-describedby="demo-privacy"
          onChange={event => {
            setSource(event.target.value);
            setEnvelope(null);
            setRevealed(null);
            setError("");
          }}
        />
      </div>
      <div className="demo-boundary">
        <span className="boundary-line" />
        <LuArrowDown aria-hidden="true" />
        <button
          className="button button-small"
          onClick={() => void encrypt()}
          disabled={busy || !source.trim()}
        >
          <LuLockKeyhole />
          {busy ? "Working…" : envelope ? "Encrypt again" : "Encrypt locally"}
          <LuArrowRight />
        </button>
        <span className="boundary-line" />
      </div>
      <div className="demo-result">
        <div className="panel-label">
          <span>{revealed !== null ? "03 / Decrypted locally" : "02 / Encrypted payload"}</span>
          <span>{revealed !== null ? "plaintext" : "AES-256-GCM"}</span>
        </div>
        <div
          className="demo-output"
          role="region"
          aria-label={revealed !== null ? "Decrypted sample" : "Encrypted payload"}
          aria-live="polite"
          aria-atomic="true"
          tabIndex={0}
        >
          {error ? (
            <p className="demo-error" role="alert">
              {error}
            </p>
          ) : revealed !== null ? (
            <pre key="revealed">{revealed}</pre>
          ) : envelope ? (
            <code key={envelope.encoded}>{envelope.encoded}</code>
          ) : (
            <div className="demo-empty">
              <span className="demo-ghost" aria-hidden="true">
                <i />
                <i />
                <i />
              </span>
              <span>
                <LuLockKeyhole /> Edit the sample, then encrypt it to see what leaves your app.
              </span>
            </div>
          )}
        </div>
        <div className="demo-success">
          {envelope ? (
            <>
              <span>
                <LuCheck /> {envelope.ciphertext.byteLength} encrypted bytes
              </span>
              <button
                onClick={revealed !== null ? () => setRevealed(null) : () => void decrypt()}
                disabled={busy}
              >
                <LuRotateCcw /> {revealed !== null ? "View ciphertext" : "Decrypt locally"}
              </button>
            </>
          ) : (
            <span className="demo-ready">{busy ? "Encrypting locally…" : "Ready to encrypt"}</span>
          )}
        </div>
      </div>
      <p className="demo-privacy" id="demo-privacy">
        Runs entirely in your browser. No data is sent to Gardbase. This demo illustrates object
        encryption; it does not connect to an enclave.
      </p>
    </div>
  );
}
