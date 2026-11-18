import { LuPlus } from "react-icons/lu";
import { site } from "@/lib/site";

const questions = [
  {
    question: "What do I need to run Gardbase?",
    answer: (
      <>
        The full data path uses AWS services and a Nitro Enclaves-capable EC2 host. The API connects
        to the enclave over vsock. The landing page runs independently with Node.js and pnpm. See
        the <a href={site.quickStart}>quick start</a> for the two workflows.
      </>
    ),
  },
  {
    question: "What does the server still learn?",
    answer: (
      <>
        Encrypted object contents are separate from operational metadata. The service can observe
        identifiers, sizes, timing, access patterns, and index tokens. Equality tokens reveal
        repeated values; order-preserving tokens reveal ordering and distributions. Review the{" "}
        <a href={site.security}>security model</a> before choosing indexes.
      </>
    ),
  },
  {
    question: "Is there a complete application SDK?",
    answer: (
      <>
        This repository provides REST endpoints, Go request and response types, and crypto
        primitives. Your application assembles the session, attestation, key handling, and object
        workflow. An ORM-style collection API is not included.
      </>
    ),
  },
  {
    question: "Does the interactive demo use a live database?",
    answer: (
      <>
        The demo uses your browser’s Web Crypto API to encrypt and decrypt the sample with
        AES-256-GCM. Its key stays in memory in this tab. It does not contact Gardbase, AWS, or an
        enclave, and does not represent the complete attestation and key-management protocol.
      </>
    ),
  },
];

export default function FAQSection() {
  return (
    <section id="faq" className="section faq-section">
      <div className="faq-layout container">
        <div>
          <p className="eyebrow">A FEW PRACTICAL DETAILS</p>
          <h2>Before you build.</h2>
          <p>
            The boundaries matter.
            <br />
            Here’s where to start.
          </p>
        </div>
        <div className="faq-list">
          {questions.map(item => (
            <details key={item.question}>
              <summary>
                {item.question}
                <LuPlus aria-hidden="true" />
              </summary>
              <div className="faq-answer">{item.answer}</div>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}
