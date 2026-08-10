---
schema_version: 1
id: 8c8b40ec-59f8-4a9f-bec7-a415b6114fff
name: landing
node: apps/landing
category: app
---
## Purpose

`landing` is the marketing site for Gardbase — a single-page React application that explains what the product is, why it exists, and how the zero-trust architecture works, then drives visitors to the GitHub repository and the docs.

It is a purely presentational node. It has no backend calls, no analytics, no forms, and no dependency on any other node in the repo: all copy, links, and product claims are hardcoded in the components and in one small config module. Its content mirrors the architecture implemented by the `api`, `enclave-service`, and `crypto-sdk` nodes (client-side AES-256-GCM, attested Nitro Enclaves, searchable encryption, a four-level key hierarchy), so changes to those guarantees should be reflected here.

## Structure

```
apps/landing/
  index.html               # document head, favicons, meta description, pre-hydration theme script
  src/
    main.tsx               # ReactDOM root: StrictMode → ThemeProvider → BrowserRouter → App
    components/
      App.tsx              # router: path "*" → MainPage
      Header.tsx           # sticky nav, anchor links, theme toggle, mobile menu
      HeroSection.tsx      # headline, trust badges, primary CTAs
      ProblemStatementSection.tsx  # "#why" — traditional DB vs Gardbase comparison
      FeaturesSection.tsx          # "#features" — six feature cards
      HowItWorksSection.tsx        # "#how-it-works" — three steps + key hierarchy
      OpenSourceSection.tsx        # "#open-source" — auditability, self-hosting, license
      PricingSection.tsx           # "#pricing" — Open Source vs Managed Cloud (early access)
      CTASection.tsx               # closing spotlight panel
      Footer.tsx                   # three link columns + logo
    pages/MainPage.tsx     # composes Header + all sections + Footer
    lib/site.ts            # single source of truth for name, repo, docs, license, company URLs
    lib/themeContext.ts    # Theme type, storage key, React context, useTheme hook
    lib/ThemeProvider.tsx  # theme state, <html> class sync, localStorage persistence
    assets/                # logo.svg, logo-white.svg
    index.css              # Tailwind v4 theme tokens, light/dark variables, animations
  public/                  # favicons and apple-touch-icon
  vite.config.ts           # React SWC + Tailwind plugins, "@" → /src alias, dev server on 0.0.0.0:3000
  project.json             # Nx project; build output goes to dist/apps/landing
  package.json             # @gardbase/landing
```

## Behavior

**Rendering.** A conventional Vite SPA. `main.tsx` mounts into `#root` inside `React.StrictMode`, wrapped by `ThemeProvider` and `BrowserRouter`. The router has a single catch-all route, so every path renders `MainPage`; navigation within the page is anchor-based (`#why`, `#features`, `#how-it-works`, `#open-source`, `#pricing`) rather than route-based.

**Theming.** Dark is the default. An inline script in `index.html` runs before hydration, reads `localStorage["gardbase-theme"]`, and applies the `light` or `dark` class plus `color-scheme` to `<html>` — this is what prevents a flash of the wrong theme. `ThemeProvider` then reads that class as its initial state, keeps `<html>` in sync on every change, and writes the choice back to `localStorage` inside a `try/catch` so private-mode browsers degrade gracefully. `Header` and `Footer` consume `useTheme()` to toggle and to swap between the dark and light logo assets.

Styling uses Tailwind CSS v4 configured entirely in `index.css`. A `@custom-variant dark` drives the `dark:` variant off the class toggle instead of `prefers-color-scheme`. Fixed brand colors (`brand`, `accent`, `accent-2`, `accent-3`, the always-dark `ink`/`surface` surfaces) live in `@theme`, while semantic tokens (`bg`, `card`, `fg`, `muted`, `subtle`, `line`) are declared with `@theme inline` and remapped per theme, so components reference semantic names and both themes follow automatically.

**Header.** Sticky, with a scroll listener that switches to a glass/blurred treatment past 8px. Desktop shows nav links, a theme toggle, a GitHub "Star" link, and a "Get Started" button pointing at the docs. Under `md` it collapses into a hamburger menu with `aria-expanded` and per-link close handling.

**Content sections.** Each section is a self-contained component with its copy declared as a local array or object literal at the top of the file — feature cards, comparison bullets, the three how-it-works steps, the key hierarchy (KMS key → tenant master key → per-object DEKs → your data), and the two pricing plans. Pricing presents "Open Source / Free forever" against a "Managed Cloud / Early access" plan badged as in development, whose CTA points at the company site rather than a signup flow.

**Shared configuration.** `lib/site.ts` centralizes the repo URL, docs URL, license name and URL, company name and URL, tagline, and copyright year. Every external link in the header, sections, and footer reads from it, so a repo move or license change is a one-line edit.

**Build and dev.** `pnpm dev` runs `nx serve` (Vite dev server, bound to `0.0.0.0:3000`); `pnpm build` runs `tsc` then `nx vite:build`, emitting to `dist/apps/landing`. Linting and formatting go through the workspace's Nx-managed ESLint and Prettier (with the Tailwind class-sorting plugin).

## Dependencies

- **Runtime:** React 19 with `react-dom`, `react-router-dom` v7, `react-icons` (Lucide `lu*` and Simple Icons `si*` sets), and `tailwindcss` v4 with `@tailwindcss/vite`.
- **Build:** Vite 7 with `@vitejs/plugin-react-swc`; TypeScript extending the workspace `tsconfig.base.json`; the `@/*` path alias is declared in both `tsconfig.json` and `vite.config.ts`.
- **Workspace:** Nx orchestrates targets and caching; pnpm workspaces manage installation; ESLint and Prettier config is shared from the repo root.
- **External assets:** Poppins and JetBrains Mono are fetched from Google Fonts via an `@import` at the top of `index.css`.
- **No internal dependencies:** the site does not consume any Go module or call the Gardbase API.

## Notes

- Everything about this node is static content. The product claims it renders (encryption modes, attestation steps, key hierarchy, licensing) are copy, not behavior — when the underlying architecture changes, nothing here breaks, so the copy has to be updated deliberately.
- The catch-all route means unknown paths render the landing page rather than a 404. Hosting must fall back to `index.html` for the SPA to work on deep links.
- `src/types.ts` is empty, and `src/vite-env.d.ts` carries only the Vite client types.
- The `CTASection` panel is intentionally dark in both themes and uses fixed brand colors rather than semantic tokens; the inline comment in the file says so explicitly.
- The pricing "Managed Cloud" plan is aspirational — it is badged "In development" and its CTA links to the company website, not to a product signup.
- `project.json` sets `outputPath` to `../../dist/apps/landing` while `vite.config.ts` sets `build.outDir` to the same location; the two must stay in agreement for Nx caching to track outputs correctly.
- There is no deployment configuration in the repo for this node — no CI workflow, Dockerfile, or hosting manifest. Publishing the built `dist/apps/landing` directory is currently a manual step.
