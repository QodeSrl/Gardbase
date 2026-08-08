---
schema_version: 1
id: b02f5975-df24-4215-a61a-edfe59296091
name: landing
node: apps/landing
category: app
---
## Purpose

`landing` is the public marketing site for Gardbase — a single-page React application that explains the product (open-source, zero-trust encrypted NoSQL database), walks through how client-side encryption and Nitro Enclave attestation work, presents the pricing tiers, and drives visitors to the GitHub repository.

It is purely presentational. It has no backend, no API calls, no authentication, and no runtime dependency on any other node in this repo. It ships as static assets.

## Structure

Nx/pnpm workspace package `@gardbase/landing`, rooted at `apps/landing`.

- `index.html` — the Vite entry document. Carries the favicon set, SEO title/description, per-color-scheme `theme-color` meta tags, and a small inline script that reads the persisted theme from `localStorage` and applies the `light`/`dark` class to `<html>` *before* React mounts, so there is no flash of the wrong theme.
- `src/main.tsx` — mounts `<App>` inside `StrictMode` → `ThemeProvider` → `BrowserRouter`.
- `src/components/App.tsx` — the router: a single catch-all `path="*"` rendering `MainPage`.
- `src/pages/MainPage.tsx` — composes the page in fixed order: `Header`, then `HeroSection`, `ProblemStatementSection`, `FeaturesSection`, `HowItWorksSection`, `OpenSourceSection`, `PricingSection`, `CTASection`, then `Footer`.
- `src/components/` — one file per section, each self-contained with its content declared as a local array of objects at the top of the file (feature cards, steps, plans, footer link groups). `Header.tsx` is the only stateful one: mobile menu toggle, scroll-shadow state, and the theme toggle button.
- `src/lib/` — `site.ts` (single source of truth for repo URL, docs URL, license, company links, tagline, year), `themeContext.ts` (`Theme` type, storage key `gardbase-theme`, context + `useTheme` hook), `ThemeProvider.tsx` (state, DOM class sync, `localStorage` persistence).
- `src/index.css` — Tailwind v4 configured entirely in CSS: `@custom-variant dark` driving the `dark:` variant off the class toggle rather than `prefers-color-scheme`, an `@theme` block of fixed brand/accent colors and fonts (Poppins, JetBrains Mono via Google Fonts), an `@theme inline` block mapping semantic tokens (`bg`, `card`, `fg`, `muted`, `subtle`, `line`) to CSS variables, and light/dark variable sets plus glass/gradient/animation utilities.
- `src/assets/` — `logo.svg` and `logo-white.svg` (swapped by active theme). `public/` — favicons and apple-touch icon.
- `src/types.ts` is present but empty; `vite-env.d.ts` holds Vite's ambient types.

Config: `vite.config.ts` (React SWC + Tailwind plugins, `@` → `/src` alias, dev server on `0.0.0.0:3000`, output to `../../dist/apps/landing`), `tsconfig.json` extending the workspace `tsconfig.base.json` (strict, `noUncheckedIndexedAccess`, `noUnusedLocals`/`noUnusedParameters`), and `eslint.config.js` which simply re-exports the root config.

## Behavior

**Rendering.** Everything renders client-side. There is no data fetching, no SSR, no prerendering — all copy is hard-coded in the component files. Navigation is anchor-based (`#why`, `#features`, `#how-it-works`, `#open-source`, `#pricing`); React Router is present but only serves the catch-all route, so any deep path renders the same page (which also means the host must be configured for SPA fallback, or every URL 404s on refresh).

**Theming.** Dark is the default. The inline script in `index.html` applies the stored class synchronously on first paint; `ThemeProvider` then reads that class as its initial state (`readInitial`), and on every change rewrites the `<html>` class, sets `style.colorScheme`, and writes back to `localStorage` inside a try/catch so private-mode storage failures are non-fatal. The header's toggle button flips between sun and moon icons and swaps the logo asset. Because the dark variant is bound to the class, the toggle is authoritative — the OS preference is not consulted.

**Header.** Sticky at `top-0`. A scroll listener (passive) sets a `scrolled` flag past 8px, which swaps a transparent border for the glass/blurred treatment. Below `md`, nav collapses into a toggled mobile menu with `aria-expanded`/`aria-label` on the trigger; selecting a link closes it.

**Content sections.** `FeaturesSection` renders six cards (client-side encryption, attested Nitro Enclaves, searchable encryption, 4-level key hierarchy, hybrid DynamoDB + S3, type-safe Go SDK). `HowItWorksSection` presents three steps: attest the enclave, encrypt client-side, store & query encrypted. `OpenSourceSection` covers auditability, self-hosting on your own AWS, the Apache 2.0 license, and building in the open. `PricingSection` has three plans — Open Source (free forever), Managed Cloud (early access, badged "In development", the highlighted tier), and Enterprise (custom) — with CTAs pointing at the repo or the company site. `Footer` groups links under Product / Open Source / Company. External links consistently use `target="_blank" rel="noreferrer"`.

**Build.** `nx vite:build` (the package's `build` script runs `tsc` first as a typecheck, then the Vite build) emits static assets to `dist/apps/landing` with `emptyOutDir`. `nx serve` runs the Vite dev server on port 3000 bound to all interfaces.

## Dependencies

- **Runtime**: React 19 + React DOM, `react-router-dom` 7, `react-icons` 5 (the `lu` Lucide and `si` Simple Icons sets), `tailwindcss` 4 with `@tailwindcss/vite`.
- **Build**: Vite 7, `@vitejs/plugin-react-swc`, TypeScript 5, ESLint 9 + typescript-eslint, Prettier with `prettier-plugin-tailwindcss` — all hoisted from the workspace root except the React/Vite pins declared locally.
- **Workspace**: Nx 21 (`@nx/vite` plugin supplies the `serve`/`vite:build`/`vite:preview`/`typecheck` targets; `project.json` only overrides the build output path), pnpm 10 workspace covering `apps/*`.
- **Network at runtime**: Google Fonts (Poppins, JetBrains Mono) is fetched by the CSS `@import`; the page otherwise makes no requests.
- **Other nodes**: none. This node shares nothing with the Go modules or the Terraform workspaces and can be built, run, and deployed entirely on its own.

## Notes

- The site is a static bundle with no deployment automation in this repo — unlike `api` and `enclave-service`, there is no Dockerfile, ECR target, or Terraform resource for it. How `dist/apps/landing` gets published is external to the codebase.
- All copy lives inline in the components; there is no CMS, i18n, or content collection. Editing marketing text means editing TSX, and recent history shows exactly that (the landing copy has been rewritten several times).
- `site.ts` is the right place to change any outbound URL, the license label, or the copyright year (currently hard-coded to 2026) — the components read it rather than duplicating strings.
- `src/types.ts` is empty and unused.
- `pnpm-workspace.yaml` lists `packages/*` alongside `apps/*`, but no `packages/` directory exists — the Go shared code lives in `pkg/` and is outside the pnpm workspace entirely.
- The package's own `packageManager` field pins pnpm 10.12.4 while the root pins 10.13.1; the root is what governs in practice.
- Because routing is a client-side catch-all, static hosting needs an SPA rewrite rule for anything other than `/`.
