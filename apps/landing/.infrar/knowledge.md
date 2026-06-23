---
schema_version: 1
id: 98c59bf1-2386-45a6-9684-efa148a277fc
name: landing
node: apps/landing
category: app
---

## Purpose
Marketing/landing website for the product. A static single-page React application presenting hero, features, how-it-works, pricing, open-source, CTA, and footer sections to drive user acquisition.

## Structure
Vite + React + TypeScript app. Entry at src/main.tsx mounting src/components/App.tsx, which renders src/pages/MainPage.tsx composed of section components (HeroSection, FeaturesSection, HowItWorksSection, ProblemStatementSection, PricingSection, OpenSourceSection, CTASection) plus Header and Footer. src/lib holds site configuration (site.ts), theme context (themeContext.ts), and ThemeProvider.tsx for light/dark theming. src/types.ts for shared types, src/index.css for global styles, src/assets for logos. public/ holds favicons and touch icons. Config: vite.config.ts, tsconfig.json/tsconfig.node.json, eslint.config.js, project.json (Nx target wiring), package.json. index.html is the Vite HTML shell.

## Behavior
Built and served as a static SPA via Vite. main.tsx bootstraps React into the index.html root element. ThemeProvider wraps the app to supply theme state via context; section components render content largely driven by site.ts config constants. No backend/runtime logic beyond client-side rendering; output is static assets for hosting/CDN. Nx project.json defines build/serve/lint targets.

## Dependencies
React + ReactDOM, Vite (build/dev server), TypeScript, ESLint (flat config). Part of an Nx monorepo (project.json). No apparent internal workspace package dependencies referenced in the file list; self-contained content from src/lib/site.ts.

## Notes
.infrar/knowledge.md present (generated knowledge artifact). Content is config-driven via site.ts, so copy/marketing changes should be made there. Theme handling is local to this app (own ThemeProvider/context) rather than a shared library.
