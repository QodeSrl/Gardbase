# Contributing to Gardbase

Thanks for helping improve Gardbase. This guide covers how changes move through the repository and what a reviewer needs to see.

## Before you start

- **Bugs:** search [existing issues](https://github.com/QodeSrl/Gardbase/issues) first, then open one with the bug report template.
- **Larger changes:** open an issue to agree on the approach before writing code, especially for routes, storage formats, key handling, or anything in the [security model](README.md#security-model).
- **Security problems:** never report them in a public issue. Follow [SECURITY.md](SECURITY.md).

## Branches

| Branch        | Purpose                                               |
| ------------- | ----------------------------------------------------- |
| `development` | Integration branch. Open pull requests against it.    |
| `test`        | Staging. Promoted from `development` by pull request. |
| `main`        | Released code. Promoted from `test` by pull request.  |

Create a topic branch from `development`, keep it focused on one change, and open a pull request back into `development`.

## Local setup

You need Node.js 22.14 or newer, pnpm 12.6 (see `packageManager` in [`package.json`](package.json)), and the Go toolchain declared in [`go.work`](go.work).

```bash
git clone --branch development https://github.com/QodeSrl/Gardbase.git
cd Gardbase
pnpm install --frozen-lockfile
```

## Checks

Run the same checks as CI before opening a pull request:

```bash
# Go modules
go vet ./apps/api/... ./apps/enclave-service/... ./pkg/api/... ./pkg/crypto/... ./pkg/enclaveproto/... ./pkg/models/...
go build ./apps/api/... ./apps/enclave-service/... ./pkg/api/... ./pkg/crypto/... ./pkg/enclaveproto/... ./pkg/models/...

# Landing page
pnpm check:landing
pnpm --dir apps/landing exec eslint src
pnpm --dir apps/landing exec prettier --check src
pnpm build:landing
```

The crypto integration tests need a deployed enclave and AWS credentials, so CI does not run them. If your change touches `pkg/crypto` or the enclave service, describe how you validated it.

## Pull requests

- Explain **why** the change is needed, not only what it does.
- Update documentation when you change routes, configuration, encryption formats, or security assumptions.
- Keep product and documentation claims consistent with the implementation. Describe index leakage and deployment requirements plainly.
- Never include AWS credentials, API keys, plaintext production data, or Terraform state.

## License

By contributing, you agree that your contributions are licensed under the [Apache License 2.0](LICENSE).
