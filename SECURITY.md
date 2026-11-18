# Security policy

Gardbase handles encryption keys and sensitive data, so we take reports seriously and appreciate responsible disclosure.

## Reporting a vulnerability

**Please do not open a public issue, discussion, or pull request for a security problem.**

Report it privately through GitHub: go to the repository's **Security** tab and select **[Report a vulnerability](https://github.com/QodeSrl/Gardbase/security/advisories/new)**.

Include as much of the following as you can:

- The affected component (`apps/api`, `apps/enclave-service`, `pkg/crypto`, infrastructure, or the landing page) and commit or branch.
- A description of the issue and its impact.
- Steps to reproduce, or a proof of concept.
- Any suggested fix or mitigation.

We will acknowledge your report, keep you updated while we investigate, and credit you in the advisory if you would like.

## Scope

In scope:

- Weaknesses in object encryption, key wrapping, session sealing, or attestation verification.
- Ways to read another tenant's data or bypass API-key permissions.
- Paths that expose plaintext keys or object contents outside the intended boundary.
- Vulnerabilities in the Terraform configuration or Docker images as shipped.

Not vulnerabilities in themselves:

- The documented leakage of encrypted indexes: equality tokens reveal repeated values, and order-preserving tokens reveal ordering and distribution. See the [security model](README.md#security-model).
- Issues that require a deployment configured against the guidance, such as attestation PCR checks disabled.
- Weaknesses in the browser-only demo on the landing page, which does not connect to Gardbase.

## Supported versions

Gardbase does not publish versioned releases yet. Fixes land on `development` and are promoted to `test` and `main`.
