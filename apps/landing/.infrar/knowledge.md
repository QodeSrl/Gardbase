---
schema_version: 1
id: c5a22c53-f9fd-4a8b-9f13-0f311f2831c6
name: landing
node: apps/landing
category: app
---

## Purpose
Marketing/landing website for the product. A static single-page React application presenting the value proposition through hero, features, pricing, and call-to-action sections, with theming (light/dark) and open-source messaging.

## Structure
Vite + React + TypeScript app. Entry at src/main.tsx mounts src/components/App.tsx. UI is composed of section components (HeroSection, FeaturesSection, HowItWorksSection, ProblemStatementSection, OpenSourceSection, PricingSection, CTASection) plus Header and Footer, assembled in src/pages/MainPage.tsx. src/lib holds cross-cutting concerns: site.ts (site metadata/config), themeContext.ts and ThemeProvider.tsx (theme state). src/types.ts holds shared TS types; src/index.css global styles; src/assets logos. index.html is the Vite HTML entry; public/ holds favicons and PWA icons. Config: vite.config.ts, tsconfig.json/tsconfig.node.json, eslint.config.js, project.json (Nx target definitions), package.json.

## Behavior
main.tsx bootstraps React and renders App wrapped in ThemeProvider, exposing theme via themeContext consumed by components. MainPage lays out the static marketing sections in order. Theme selection (light/dark) is toggled through the provider, likely persisting/reading a preference and applying a class or attribute for CSS. No dynamic data fetching implied; content is largely static and driven by site.ts configuration. Built and served as a static SPA via Vite.

## Dependencies
External: React, ReactDOM, Vite, TypeScript, ESLint. Build/orchestration via Nx (project.json). Internal: components depend on lib/themeContext and lib/site; ThemeProvider provides themeContext; MainPage aggregates section components. Assets and public icons referenced from HTML/components.

## Notes
Purely presentational app with no backend coupling shown. Adding new sections means creating a component and wiring it into MainPage. Site copy/links should be centralized in lib/site.ts. Ensure favicon/PWA assets in public/ stay in sync with branding in src/assets.
