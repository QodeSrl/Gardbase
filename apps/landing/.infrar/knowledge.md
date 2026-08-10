---
schema_version: 1
id: 61e96bce-c3d9-4025-9135-1aae0926f7b7
name: landing
node: apps/landing
category: app
---
## Purpose

`landing` is the public marketing site for Gardbase — a single-page React application that explains the product, positions it against conventional databases, and pushes visitors toward the GitHub repository.

It is pure presentation. It makes no API calls, holds no user state, and has no dependency on the Go services in this repo; the only thing it shares with them is the story it tells about them (client-side encryption, Nitro Enclave attestation, the four-level key hierarchy, searchable encrypted indexes). It ships as a static bundle to `dist/apps/landing`.

## Structure

The node root is `apps/landing`; the Nx `sourceRoot` is `apps/landing/src`.

- `index.html` — Vite entry document. Carries the favicon set, description/`theme-color` meta, and a small inline script that reads `localStorage["gardbase-theme"]` and sets the `light`/`dark` class on `<html>` before React mounts.
- `src/main.tsx` — mounts `<StrictMode><ThemeProvider><BrowserRouter><App/>`.
- `src/components/App.tsx` — the router; a single catch-all `path="*"` route rendering `MainPage`.
- `src/pages/MainPage.tsx` — the page skeleton: `Header`, then `HeroSection` → `ProblemStatementSection` → `FeaturesSection` → `HowItWorksSection` → `OpenSourceSection` → `PricingSection` → `CTASection`, then `Footer`.
- `src/components/*.tsx` — one self-contained section per file. Each keeps its copy in a module-level array or object at the top of the file and renders it below, so editing content rarely means touching JSX.
- `src/lib/site.ts` — the single source of truth for external links and brand strings (repo URL, docs URL, license name and URL, company URL, tagline, year). Sections import from it rather than hardcoding URLs.
- `src/lib/themeContext.ts` / `src/lib/ThemeProvider.tsx` — the light/dark theme context, split so the context and hook live apart from the provider component.
- `src/index.css` — the entire design system: Google Fonts import, `@import "tailwindcss"`, a custom `dark` variant, fixed brand tokens in `@theme`, semantic tokens in `@theme inline`, the light/dark CSS variable sets, base layer resets, and the utility/keyframe definitions (`glass`, `bg-grid`, `mask-fade`, `text-gradient`, `animate-*`).
- `src/assets/` — `logo.svg` and `logo-white.svg` (swapped by theme). `public/` — favicons and the apple-touch icon.
- `vite.config.ts`, `tsconfig.json`, `tsconfig.node.json`, `package.json`, `project.json`, `eslint.config.js` (a one-line re-export of the workspace config).

## Behavior

**Build and dev.** Vite 7 with `@vitejs/plugin-react-swc` and `@tailwindcss/vite`. The `@` alias maps to `/src`. The dev server binds `0.0.0.0:3000`. `build.outDir` is `../../dist/apps/landing` with `emptyOutDir`, and `project.json` mirrors that path in the Nx `outputs` so caching works. The package `build` script runs `tsc` for type checking before `nx vite:build`.

**Theming.** Dark is the default. The inline script in `index.html` applies the stored class synchronously, which avoids a flash of the wrong theme; `ThemeProvider` then reads that class as its initial state, and on every change swaps the `light`/`dark` class on `<html>`, sets `style.colorScheme`, and writes back to `localStorage` (wrapped in a `try` so private-mode storage failures are ignored). Components consume `useTheme()` for the toggle button in `Header` and the logo swap in `Header` and `Footer`. Everything else themes itself through the CSS variables — components reference semantic classes like `bg-bg`, `text-fg`, `text-muted`, `border-line`, so no component branches on theme for color.

**Sections.** `Header` is sticky and switches to the `glass` treatment once `window.scrollY > 8`, with a collapsible mobile menu; nav links are in-page anchors (`#why`, `#features`, `#how-it-works`, `#open-source`, `#pricing`) served by `scroll-behavior: smooth`. `CTASection` is deliberately a dark spotlight panel in both themes and uses the `text-gradient-bright` utility instead of the theme-aware one. `PricingSection` renders a typed `Plan[]` with two tiers — a free self-hosted open-source plan and a "Managed Cloud" tier badged as in development.

**Routing.** The catch-all route means every path renders the same page; there is no 404 view and no code splitting.

**Linting and formatting.** `apps/landing/eslint.config.js` re-exports the workspace config, which enables type-checked `typescript-eslint` rules (its `parserOptions.project` list includes `./apps/*/tsconfig.json`) plus the React hooks and react-refresh plugins. Formatting is Prettier with `prettier-plugin-tailwindcss` for class sorting, configured at the repo root.

## Dependencies

**Runtime:** React 19 + React DOM, `react-router-dom` 7, `react-icons` (the `Lu` Lucide set for UI icons and `Si` for the GitHub mark), `tailwindcss` 4 with `@tailwindcss/vite`.

**Build/dev:** Vite, `@vitejs/plugin-react-swc`, TypeScript, `@types/react`, `@types/react-dom`.

**Workspace:** pnpm workspace member (`apps/*`), managed by Nx 21 with the `@nx/vite` and `@nx/eslint` plugins — those supply the `serve`, `vite:build`, `vite:preview`, `typecheck`, and `eslint:lint` targets, while `project.json` only overrides the build output path. `tsconfig.json` extends the strict repo-root `tsconfig.base.json` (`strict`, `noUncheckedIndexedAccess`, `noUnusedLocals`, `noUnusedParameters`).

**External at runtime:** the Google Fonts CDN for Poppins and JetBrains Mono, imported from `index.css`. Nothing else — no analytics, no backend.

**No dependency** on `apps/api`, `apps/enclave-service`, `pkg/*`, or any Terraform output.

## Notes

- The landing site is not deployed by anything in this repo. `infrastructure/bootstrap` and `infrastructure/main` provision only the API/enclave stack; there is no bucket, CDN, or pipeline here for the built `dist/apps/landing` output.
- `src/types.ts` and `src/vite-env.d.ts` exist but `types.ts` is empty; shared types would go there.
- `App.tsx` lives under `src/components/` rather than at the `src` root, and `MainPage.tsx` is imported as `HomePage` — a naming mismatch worth knowing when searching.
- Because the router matches `*`, deep links resolve to the landing page rather than a 404; any static host serving this bundle should be configured with an SPA fallback for the same reason.
- All product claims in the copy (AES-256-GCM, PCR/COSE attestation verification, deterministic and order-preserving index encryption, the four-level key hierarchy) describe behavior actually implemented in `pkg/crypto` and `apps/enclave-service`; if that behavior changes, the copy in `FeaturesSection`, `HowItWorksSection`, and `ProblemStatementSection` goes stale.
- `site.ts` hardcodes `year: 2026` for the footer copyright rather than deriving it.
