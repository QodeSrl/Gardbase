---
schema_version: 1
id: 087ee911-b7bc-465f-b64d-22d324e055ca
name: landing
node: apps/landing
category: app
---
## Purpose

`landing` is the public marketing site for Gardbase — a single-page React application that explains what the product is, why zero-trust encryption matters, how the enclave architecture works, and where to get the code.

It is entirely presentational. There is no API client, no authentication, no form submission, and no state beyond a theme toggle: every call to action links out to the GitHub repository or to the parent company's site. It shares no code with the Go services and has no runtime dependency on them.

## Structure

A standard Vite + React + TypeScript single-page app.

```
apps/landing/
├── index.html              # HTML shell; includes an inline anti-FOUC theme script
├── vite.config.ts          # React SWC + Tailwind v4 plugins, "@" → /src alias
├── package.json            # deps and pnpm scripts
├── project.json            # Nx target: build → dist/apps/landing
├── tsconfig.json           # extends the workspace base; "@/*" paths
├── eslint.config.js
├── public/                 # favicons, apple-touch-icon
└── src/
    ├── main.tsx            # ReactDOM root: StrictMode → ThemeProvider → BrowserRouter → App
    ├── index.css           # Tailwind v4 entry + the entire design-token system
    ├── types.ts            # (empty)
    ├── components/
    │   ├── App.tsx         # router: every path renders MainPage
    │   ├── Header.tsx      # sticky nav, mobile menu, theme toggle
    │   ├── HeroSection.tsx
    │   ├── ProblemStatementSection.tsx
    │   ├── FeaturesSection.tsx
    │   ├── HowItWorksSection.tsx
    │   ├── OpenSourceSection.tsx
    │   ├── PricingSection.tsx
    │   ├── CTASection.tsx
    │   └── Footer.tsx
    ├── pages/MainPage.tsx  # composes Header + the seven sections + Footer
    ├── lib/
    │   ├── site.ts         # single source of truth for names, URLs, license, year
    │   ├── themeContext.ts # Theme type, storage key, context, useTheme hook
    │   └── ThemeProvider.tsx
    └── assets/             # logo.svg, logo-white.svg
```

The composition is deliberately flat: `MainPage` renders the sections in order inside a `<main>`, and each section component is self-contained — its own copy, its own local data arrays (feature lists, pricing plans, steps), its own Tailwind classes. There is no shared UI primitive library; visual consistency comes from the design tokens in `index.css` rather than from shared components.

## Behavior

**Rendering.** `main.tsx` mounts into `#root` wrapped in `StrictMode`, then `ThemeProvider`, then `BrowserRouter`. `App` declares a single catch-all route (`path="*"`) rendering `MainPage`, so the router is present for future expansion but currently every URL resolves to the same page. Navigation between sections is done with in-page anchor links (`#why`, `#features`, `#how-it-works`, `#open-source`, `#pricing`) rather than routes.

**Theming.** Light/dark is a class on `<html>`, not a media query. The flow:

1. An inline script in `index.html` runs before React and reads `localStorage["gardbase-theme"]` (defaulting to `"dark"`), adding the class and setting `style.colorScheme` immediately. This prevents a flash of the wrong theme.
2. `ThemeProvider` reads the initial value back off `document.documentElement.classList` — so React inherits whatever the inline script decided rather than re-deciding.
3. Any change to `theme` runs an effect that swaps the class, updates `colorScheme`, and writes back to `localStorage` inside a `try/catch` (storage may be unavailable).
4. `useTheme()` exposes `{ theme, toggle, setTheme }`; `Header` renders a sun/moon toggle button with an appropriate `aria-label`.

`index.css` sets up the token system: `@custom-variant dark (&:where(.dark, .dark *))` rewires Tailwind's `dark:` variant to the class toggle instead of `prefers-color-scheme`. A fixed `@theme` block defines brand colors, always-dark surfaces (used for code windows and the CTA spotlight regardless of theme), and the Poppins / JetBrains Mono font stacks. An `@theme inline` block maps semantic tokens (`--color-bg`, `--color-fg`, `--color-muted`, `--color-line`, …) onto CSS custom properties that are redefined per theme, so components write `bg-bg` / `text-fg` and get the right value automatically.

**Header.** Sticky at `top-0`, with a scroll listener (passive, cleaned up on unmount) that adds a glass/blur treatment once `scrollY > 8`. Holds the desktop nav, a mobile hamburger menu backed by `useState`, the theme toggle, and a GitHub star link. The logo swaps between `logo.svg` and `logo-white.svg` based on the active theme.

**Content.** Section content is hardcoded in each component as typed local arrays — for example `PricingSection` declares a `Plan` type and a `plans` array with two entries: "Open Source" (free, self-hosted, links to the repo) and "Managed Cloud" (marked "In development", links to the company site). Anything that could drift — product name, repo URL, license name and URL, company URL, tagline, copyright year — is centralized in `src/lib/site.ts` and interpolated.

**Icons.** `react-icons` — the `lu` (Lucide) set for UI affordances and `si` (Simple Icons) for the GitHub mark.

## Dependencies

**Runtime:**

- `react` ^19.1.0 / `react-dom` ^19.1.0
- `react-router-dom` ^7.6.3
- `react-icons` ^5.5.0
- `tailwindcss` ^4.1.11 with `@tailwindcss/vite` ^4.1.11 (the Vite plugin, not PostCSS — Tailwind v4 style)

**Build:**

- `vite` ^7.0.2 with `@vitejs/plugin-react-swc` ^3.10.2
- TypeScript, extending `tsconfig.base.json` at the workspace root
- `@types/react`, `@types/react-dom`

**External at runtime:** Google Fonts, imported at the top of `index.css` for Poppins and JetBrains Mono.

**Configuration:**

- Path alias `@` → `/src`, declared in both `vite.config.ts` and `tsconfig.json`.
- Dev server on `0.0.0.0:3000`.
- Build output to `../../dist/apps/landing` with `emptyOutDir: true`; `project.json` declares the same path as `outputPath` and as the Nx `outputs` entry.

**Scripts** (`package.json`, run through pnpm/Nx): `dev` → `nx serve`, `build` → `tsc && nx vite:build`, `lint` → ESLint with `--max-warnings 0`, `preview`, `format` / `format:check` → Prettier.

**Workspace:** part of the pnpm workspace (`pnpm-workspace.yaml`) and the Nx graph (`nx.json`) as `@gardbase/landing`. It is the only JavaScript node in an otherwise Go and Terraform repository.

**No deployment infrastructure exists in this repo.** Neither `infrastructure/bootstrap` nor `infrastructure/main` provisions hosting for the landing site — no CloudFront distribution, no static-site S3 bucket, no domain. It builds to a static directory and is presumably deployed by something outside this repository.

## Notes

- `src/types.ts` is an empty file.
- `project.json` declares only a `build` target with an `options.outputPath` but no `executor` or `command`, relying on Nx inference from the Vite config. The `package.json` scripts (`nx serve`, `nx vite:build`, `nx eslint:lint`) likewise assume inferred targets.
- The build output path is configured in two places — `vite.config.ts` (`build.outDir`) and `project.json` (`options.outputPath`) — which must stay in sync manually.
- `build` runs `tsc` before Vite, so type errors fail the build even though Vite itself would transpile past them.
- The router is effectively vestigial: one catch-all route, no route-based navigation, no 404 handling. A URL like `/anything` renders the landing page rather than an error.
- The theme default is dark in three independent places — the inline script in `index.html`, `readInitial()` in `ThemeProvider.tsx`, and the `createContext` default in `themeContext.ts`. The storage key `"gardbase-theme"` is likewise duplicated as a literal in `index.html` and as `THEME_STORAGE_KEY` in `themeContext.ts`.
- `site.ts` hardcodes `year: 2026`, so the footer copyright needs a manual bump.
- Marketing copy is inlined per component rather than extracted to a content file, so copy changes mean editing TSX. Recent git history (`Rewrite hero/CTA/problem copy`, `Clean up landing copy`) suggests this churns.
- No tests, no test target, and no accessibility or visual-regression tooling configured for this app.
