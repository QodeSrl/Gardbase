# Gardbase landing page

The public-facing introduction to Gardbase: its architecture, encrypted storage primitives, source code, and self-hosting workflow. Built with React 19, TypeScript, Vite, and Tailwind CSS.

[Project overview](../../README.md) · [API reference](../api/README.md) · [Deployment guide](../../infrastructure/README.md)

## Run locally

Use Node.js 22.14.0 or newer and the root package manager version (`pnpm@12.6.0`). Run all commands from the repository root:

```bash
pnpm install --frozen-lockfile
pnpm dev:landing
```

Vite normally serves the page at `http://localhost:3000`. AWS credentials and the Go services are not required.

## Check and build

```bash
pnpm check:landing
pnpm build:landing
pnpm preview:landing
```

The build writes static assets to `dist/apps/landing`. The preview command prints its local URL. Serve that directory with a static host; if deep links are supported, configure them to return `index.html`.

## Where to edit

| File                                                     | Purpose                                                                                                  |
| -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| [`src/pages/MainPage.tsx`](src/pages/MainPage.tsx)       | Section order and page landmarks                                                                         |
| [`src/components`](src/components)                       | Navigation, encryption demo, architecture, capabilities, developer examples, deployment, FAQ, and footer |
| [`src/index.css`](src/index.css)                         | Theme tokens, layout, responsive rules, focus states, and reduced-motion support                         |
| [`src/lib/site.ts`](src/lib/site.ts)                     | Shared repository, documentation, and deployment links                                                   |
| [`src/lib/encryptionDemo.ts`](src/lib/encryptionDemo.ts) | Standalone browser encryption/decryption helpers                                                         |
| [`src/lib/highlight.tsx`](src/lib/highlight.tsx)         | Minimal syntax highlighting for the Go and HTTP samples in the developer section                         |
| [`index.html`](index.html)                               | Page title, description, favicon references, and initial theme                                           |

The existing Gardbase SVG logos are retained. Typography uses Geist and Geist Mono, bundled from `@fontsource-variable` packages so the page makes no requests to a third-party font service. Documentation links point to the `development` branch; update the shared source URL when the documentation is promoted to another release branch.

## Encryption demo

The hero demonstrates AES-256-GCM with the browser's Web Crypto API:

1. Edit the sample and select **Encrypt locally**.
2. The browser generates a fresh, non-extractable 256-bit key and a random 12-byte nonce.
3. The page displays the ciphertext and authentication tag as base64.
4. Select **Decrypt locally** to show the decrypted sample in the same result area, using the key and nonce held in this tab. Select **View ciphertext** to switch back.

The result area keeps a fixed height through each step. Long ciphertext and plaintext scroll inside the preview, so the panel and surrounding page stay in place.

Editing the sample clears the prior result. The sample, key, and ciphertext are not sent to a backend or saved to persistent storage. The demo requires HTTPS or localhost and a browser with Web Crypto support. The ciphertext display is illustrative: it does not include Gardbase's complete envelope or attestation/session protocol, and it is not a database request.

## Interaction and accessibility checks

When changing the page, check:

- Light and dark themes, including persistence after reloading. Only the `gardbase-theme` preference is stored in local storage.
- Mobile navigation with touch, keyboard activation, Escape dismissal, and closing after choosing a section.
- Skip link, visible focus indicators, heading order, and the sample textarea label.
- Encryption, decryption, empty input, Unicode input, and editing after encryption.
- Both developer examples, clone-command feedback, and expandable FAQ answers.
- Narrow layouts, horizontal scrolling inside code examples, and reduced-motion preferences.

Keep product copy aligned with the implementation. Examples should use actual functions or explicitly marked request templates. Describe index leakage and deployment requirements without claiming certification, guaranteed breach resistance, or unimplemented services.
