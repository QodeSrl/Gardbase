---
schema_version: 1
id: 028999ee-c197-41aa-944f-2a2a6836ea9a
name: landing
node: apps/landing
category: app
---
## Purpose

`landing` is the public marketing site for Gardbase — a single-page React app that explains what the product is, why zero-trust encrypted storage matters, how the enclave attestation flow works, and where the code lives. It exists to convert visitors into GitHub stars and early-access sign-ups; it is entirely static and has no runtime dependency on the `api` or `enclave-service` nodes.

## Structure

```
apps/landing/
  index.html                # Vite entry; favicons, SEO meta, pre-paint theme script
  vite.config.ts            # React SWC + Tailwind v4 plugins, "@" → /src alias
  package.json              # @gardbase/landing
  project.json              # Nx wiring (build output → dist/apps/landing)
  tsconfig.json             # extends the workspace tsconfig.base.json
  eslint.config.js
  public/                   # favicon set + apple-touch-icon
  src/
    main.tsx                # ReactDOM root: StrictMode → ThemeProvider → BrowserRouter → App
    index.css               # Tailwind v4 config-in-CSS: fonts, @theme tokens, light/dark vars, utilities
    types.ts                # currently empty
    components/
      App.tsx               # router: every path renders MainPage
      Header.tsx            # sticky nav, scroll-aware glass, mobile menu, theme toggle
      HeroSection.tsx       # headline, CTAs, trust badges
      ProblemStatementSection.tsx   # "traditional vs Gardbase" contrast  (#why)
      FeaturesSection.tsx           # six feature cards                   (#features)
      HowItWorksSection.tsx         # three steps + key hierarchy         (#how-it-works)
      OpenSourceSection.tsx         # four open-source pitches            (#open-source)
      PricingSection.tsx            # three plans                         (#pricing)
      CTASection.tsx                # closing call to action
      Footer.tsx                    # three link columns + legal
    pages/MainPage.tsx      # composes Header + all sections + Footer
    lib/
      site.ts               # single source of truth for repo/docs/license/company URLs
      themeContext.ts       # Theme type, storage key, React context, useTheme hook
      ThemeProvider.tsx     # state + <html> class/colorScheme sync + localStorage persistence
    assets/                 # logo.svg, logo-white.svg
```

## Behavior

**Rendering.** Client-side only. `main.tsx` mounts into `#root` and wraps the tree in `ThemeProvider` and `BrowserRouter`. Routing is nominal: `App.tsx` maps `path="*"` to `MainPage`, so every URL renders the same page. Navigation is anchor-based (`#why`, `#features`, `#how-it-works`, `#open-source`, `#pricing`) with `scroll-behavior: smooth` from CSS.

**Theming.** Dark is the default. An inline script in `index.html` runs *before paint*, reads `gardbase-theme` from localStorage, and sets the matching class and `colorScheme` on `<html>` — this is what prevents a light flash on load. `ThemeProvider` then takes over: it reads the initial theme from the class the script already applied, and on every change swaps the `light`/`dark` class, updates `style.colorScheme`, and writes back to localStorage (failing silently if storage is unavailable). `useTheme()` exposes `{theme, toggle, setTheme}`; `Header` renders the sun/moon toggle and `Footer` uses it to pick the light or dark logo variant.

**Styling.** Tailwind v4, configured entirely in `index.css` rather than a JS config file. A `@custom-variant dark (&:where(.dark, .dark *))` binds the `dark:` variant to the class toggle instead of `prefers-color-scheme`. Fixed brand colors, accents, always-dark surfaces, and the Poppins/JetBrains Mono font stacks live in `@theme`; semantic tokens (`bg`, `card`, `fg`, `muted`, `subtle`, `line`) are declared in `@theme inline` and repointed by the `:root.dark` / `:root.light` blocks, so components reference semantic names (`bg-bg`, `text-muted`, `border-line`) and never hardcode a palette. Custom utilities include `.text-gradient`, `.text-gradient-bright` (for panels that stay dark in both themes), and `.glass`. Icons come from `react-icons` (`lu` and `si` sets).

**Content.** Sections are driven by local literal arrays at the top of each component (`features`, `steps`, `hierarchy`, `plans`, `points`, `footerCols`), so copy edits are data edits. Every external URL — repo, docs, license, company — resolves through `lib/site.ts`. The pricing tiers are Open Source (free, self-hosted), Managed Cloud (marked "In development", links to the company site for early access), and Enterprise (custom).

**Header interaction.** A scroll listener flips a `scrolled` flag past 8px to swap in the glass/border treatment; the mobile menu is local `useState`. Both are self-contained — there is no global state beyond theme.

**Build.** `nx vite:build` (via `pnpm build`, which runs `tsc` first) emits to `dist/apps/landing` at the workspace root with `emptyOutDir`. Dev server binds `0.0.0.0:3000`. From the repo root: `pnpm dev:landing`.

## Dependencies

- **Framework:** React 19 + `react-dom`, `react-router-dom` 7.
- **Build:** Vite 7, `@vitejs/plugin-react-swc`, `@tailwindcss/vite` + `tailwindcss` 4.
- **UI:** `react-icons` (Lucide + Simple Icons).
- **Fonts:** Poppins and JetBrains Mono, loaded from Google Fonts via `@import` at the top of `index.css` (a render-blocking external request).
- **Workspace:** part of the pnpm workspace and the Nx graph as `@gardbase/landing`; shares the root ESLint and Prettier config (Prettier runs with `prettier-plugin-tailwindcss` for class sorting). No dependency on any Go module or on the deployed API.

## Notes

- Nothing here is deployed by the Terraform in `infrastructure/` — that stack provisions only the API/enclave host. Hosting for the landing build output is external to this repo.
- The router exists but serves no distinct routes; `path="*"` → `MainPage` means deep links and 404s alike render the home page. Adding real routes means adding entries in `App.tsx`.
- `src/types.ts` is empty; per-component types are declared inline.
- The pre-paint theme script in `index.html` and `ThemeProvider.readInitial()` are two halves of one mechanism. Changing the storage key or the class names requires editing both, plus `THEME_STORAGE_KEY` in `themeContext.ts`.
- `site.year` is hardcoded to 2026 rather than derived from the current date.
- Marketing copy makes concrete security claims (client-side AES-256-GCM, Nitro attestation, the 4-level key hierarchy) that mirror the actual implementation in `pkg/crypto` and `enclave-service` — a change to the security model should be reflected here too.
