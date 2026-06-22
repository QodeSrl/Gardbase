---
schema_version: 1
id: ced3d59b-5c02-4640-88e1-3cc81d9d5635
name: landing
node: apps/landing
category: app
---

## Purpose
Public-facing marketing/landing website for the product. Presents hero, problem statement, features, how-it-works, pricing, open-source, and CTA sections to convert visitors. Static single-page React app served by Vite.

## Structure
Standard Vite + React + TypeScript app within an Nx monorepo (project.json, eslint.config.js). Entry: src/main.tsx mounts App.tsx. App.tsx composes the single MainPage (src/pages/MainPage.tsx), which renders the section components (Header, HeroSection, ProblemStatementSection, FeaturesSection, HowItWorksSection, OpenSourceSection, PricingSection, CTASection, Footer). src/lib holds shared concerns: site.ts (site metadata/config), themeContext.ts + ThemeProvider.tsx (theme/dark-mode context). src/types.ts for shared types. Styling via src/index.css. Static assets in public/ (favicons, touch icon) and src/assets/ (logos). HTML shell in index.html. Build config in vite.config.ts; TS config split into tsconfig.json / tsconfig.node.json.

## Behavior
main.tsx bootstraps React, wraps the tree in ThemeProvider for light/dark theming via React context (themeContext.ts), and renders App -> MainPage -> ordered marketing sections. Header likely provides navigation/theme toggle; Footer provides links. Content/links sourced from site.ts. No routing beyond a single page; primarily presentational with theme state being the main interactive concern. Built and served as a static SPA by Vite.

## Dependencies
React + ReactDOM, Vite (build/dev server), TypeScript, ESLint (flat config). Nx tooling (project.json) for monorepo task orchestration. No backend calls implied; self-contained static content from src/lib/site.ts and local assets.

## Notes
Theme state is local to this app via React context, not shared from a workspace lib. site.ts centralizes copy/links — update there for content changes. Favicons/icons duplicated across public/ for various platforms. Section components are stateless presentational units; adding/reordering sections is done in MainPage.tsx.
