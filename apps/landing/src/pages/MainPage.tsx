import React, { useState } from "react";
import {
  LuShield,
  LuDatabase,
  LuLock,
  LuArrowRight,
  LuCircleCheck,
  LuMenu,
  LuX,
} from "react-icons/lu";
import Logo from "../assets/logo.svg";
import LogoWhite from "../assets/logo-white.svg";

const MainPage: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-gray-100 bg-white">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center">
              <a href="/#">
                <img draggable={false} src={Logo} alt="Gardbase Logo" className="h-6 sm:h-8" />
              </a>
            </div>

            {/* Desktop Navigation */}
            <nav className="hidden space-x-4 md:flex lg:space-x-8">
              <a
                href="#features"
                className="hover:text-brand text-sm text-gray-600 transition-colors lg:text-base"
              >
                Features
              </a>
              <a
                href="#how-it-works"
                className="hover:text-brand text-sm text-gray-600 transition-colors lg:text-base"
              >
                How it Works
              </a>
              <a
                href="#pricing"
                className="hover:text-brand text-sm text-gray-600 transition-colors lg:text-base"
              >
                Pricing
              </a>
            </nav>

            {/* Desktop Actions */}
            <div className="hidden items-center space-x-4 md:flex">
              <button className="hover:text-brand text-sm text-gray-600 transition-colors lg:text-base">
                Sign In
              </button>
              <button className="bg-brand hover:bg-brand/90 rounded-lg px-3 py-1.5 text-sm text-white transition-colors lg:px-4 lg:py-2 lg:text-base">
                Get Started
              </button>
            </div>

            {/* Mobile menu button */}
            <button
              onClick={() => setMobileMenuOpen(prev => !prev)}
              aria-label={mobileMenuOpen ? "Close menu" : "Open menu"}
              aria-expanded={mobileMenuOpen}
              className="focus:ring-brand inline-flex items-center justify-center rounded-md p-2 text-gray-600 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:ring-2 md:hidden"
            >
              {mobileMenuOpen ? <LuX className="h-6 w-6" /> : <LuMenu className="h-6 w-6" />}
            </button>
          </div>
        </div>

        {/* Mobile menu */}
        {mobileMenuOpen && (
          <div className="border-t border-gray-100 md:hidden">
            <div className="space-y-1 px-4 pb-3 pt-2">
              <a
                href="#features"
                className="hover:text-brand block rounded-md px-3 py-2 text-base font-medium text-gray-600 hover:bg-gray-50"
                onClick={() => setMobileMenuOpen(false)}
              >
                Features
              </a>
              <a
                href="#how-it-works"
                className="hover:text-brand block rounded-md px-3 py-2 text-base font-medium text-gray-600 hover:bg-gray-50"
                onClick={() => setMobileMenuOpen(false)}
              >
                How it Works
              </a>
              <a
                href="#pricing"
                className="hover:text-brand block rounded-md px-3 py-2 text-base font-medium text-gray-600 hover:bg-gray-50"
                onClick={() => setMobileMenuOpen(false)}
              >
                Pricing
              </a>
              <div className="border-t border-gray-100 pt-4">
                <button className="hover:text-brand block w-full rounded-md px-3 py-2 text-left text-base font-medium text-gray-600 hover:bg-gray-50">
                  Sign In
                </button>
                <button className="bg-brand hover:bg-brand/90 mt-2 block w-full rounded-lg px-3 py-2 text-base font-medium text-white transition-colors">
                  Get Started
                </button>
              </div>
            </div>
          </div>
        )}
      </header>

      {/* Hero Section */}
      <section className="py-22 lg:py-25 relative bg-gradient-to-b from-slate-50 to-white sm:py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <div className="text-brand mb-4 inline-flex items-center rounded-full bg-blue-50 px-3 py-1.5 text-xs font-medium sm:mb-6 sm:px-4 sm:py-2 sm:text-sm">
              <LuShield className="mr-2 h-3 w-3 sm:h-4 sm:w-4" />
              GDPR-Native Database Platform
            </div>
            <h1 className="text-brand mb-4 text-3xl font-bold sm:mb-6 sm:text-4xl md:text-5xl lg:text-6xl">
              The Database That <span className="block sm:inline md:block">Can't Be Breached</span>
            </h1>
            <p className="mx-auto mb-6 max-w-3xl text-base text-gray-600 sm:mb-8 sm:text-lg md:text-xl">
              Traditional databases store everything. When breached, they expose everything.
              Gardbase flips the script—storing only encrypted data, makes catastrophic data
              breaches <strong>architecturally impossible</strong>.
            </p>
            <div className="flex flex-col gap-3 sm:flex-row sm:justify-center sm:gap-4">
              <button className="bg-brand hover:bg-brand/90 flex items-center justify-center rounded-lg px-6 py-3 text-base font-semibold text-white transition-colors sm:px-8 sm:py-4 sm:text-lg">
                See How It Works
                <LuArrowRight className="ml-2 h-4 w-4 sm:h-5 sm:w-5" />
              </button>
              <button className="text-brand rounded-lg border border-gray-300 px-6 py-3 text-base font-semibold transition-colors hover:bg-gray-50 sm:px-8 sm:py-4 sm:text-lg">
                Try Live Demo
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Problem Statement */}
      <section className="lg:py-25 mb-20 bg-white py-12 sm:py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-20 text-center sm:mb-16">
            <h2 className="text-brand mb-3 text-2xl font-bold tracking-tight sm:mb-4 sm:text-3xl md:text-4xl">
              GDPR Compliance Shouldn't Be This Hard
            </h2>
            <p className="mx-auto max-w-3xl text-base text-gray-600 sm:text-lg md:text-xl">
              Traditional databases make you choose between developer experience and compliance.
              With complex configurations, shared responsibility models, and constant legal
              uncertainty.
            </p>
          </div>

          <div className="grid gap-6 sm:gap-8 md:grid-cols-3">
            <div className="rounded-2xl bg-red-50 p-6 sm:p-8">
              <div className="mb-2 text-xs font-medium uppercase tracking-wider text-red-500 group-hover:text-red-400 sm:text-sm">
                Financial Impact
              </div>
              <h3 className="text-brand mb-2 text-lg font-semibold sm:text-xl">€5.88B in Fines</h3>
              <p className="text-sm text-gray-600 sm:text-base">
                GDPR fines continue to escalate, with enforcement becoming more aggressive and
                personal liability extending to executives.
              </p>
            </div>

            <div className="rounded-2xl bg-orange-50 p-6 sm:p-8">
              <div className="mb-2 text-xs font-medium uppercase tracking-wider text-orange-500 group-hover:text-orange-400 sm:text-sm">
                Business Cost
              </div>
              <h3 className="text-brand mb-2 text-lg font-semibold sm:text-xl">8% Profit Drop</h3>
              <p className="text-sm text-gray-600 sm:text-base">
                SMEs face disproportionate compliance costs, with some experiencing over 8%
                reduction in profits due to regulatory overhead.
              </p>
            </div>

            <div className="rounded-2xl bg-yellow-50 p-6 sm:p-8">
              <div className="mb-2 text-xs font-medium uppercase tracking-wider text-amber-500 group-hover:text-amber-400 sm:text-sm">
                Technical Burden
              </div>
              <h3 className="text-brand mb-2 text-lg font-semibold sm:text-xl">Complex Tools</h3>
              <p className="text-sm text-gray-600 sm:text-base">
                Current solutions offer compliance as a tool you must configure correctly, leaving
                all responsibility and risk with you.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Solution */}
      <section id="features" className="lg:py-30 bg-slate-50 py-12 sm:py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-12 text-center sm:mb-16">
            <h2 className="text-brand mb-3 text-2xl font-bold tracking-tight sm:mb-4 sm:text-3xl md:text-4xl">
              Compliance as a Service, Not a Tool
            </h2>
            <p className="mx-auto max-w-3xl text-base text-gray-600 sm:text-lg md:text-xl">
              Gardbase shifts the paradigm from complex configuration to automatic compliance. Our
              zero-knowledge architecture makes data breaches technically impossible.
            </p>
          </div>

          <div className="grid items-center gap-8 sm:gap-12 md:grid-cols-2">
            <div>
              <div className="space-y-6 sm:space-y-8">
                <div className="flex items-start space-x-3 sm:space-x-4">
                  <div className="bg-brand flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl sm:h-10 sm:w-10">
                    <LuShield className="h-4 w-4 text-white sm:h-5 sm:w-5" />
                  </div>
                  <div>
                    <h3 className="text-brand mb-1.5 text-lg font-semibold tracking-tight sm:mb-2 sm:text-xl">
                      Breach-Proof Architecture
                    </h3>
                    <p className="text-sm text-gray-600 sm:text-base">
                      Client-side encryption with zero-knowledge design. Even if our infrastructure
                      is compromised, your data remains useless ciphertext.
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3 sm:space-x-4">
                  <div className="bg-brand flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl sm:h-10 sm:w-10">
                    <LuCircleCheck className="h-4 w-4 text-white sm:h-5 sm:w-5" />
                  </div>
                  <div>
                    <h3 className="text-brand mb-1.5 text-lg font-semibold tracking-tight sm:mb-2 sm:text-xl">
                      Automated Compliance
                    </h3>
                    <p className="text-sm text-gray-600 sm:text-base">
                      GDPR-conformant audit logs, automated data retention, and built-in portability
                      features work out of the box.
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3 sm:space-x-4">
                  <div className="bg-brand flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl sm:h-10 sm:w-10">
                    <LuDatabase className="h-4 w-4 text-white sm:h-5 sm:w-5" />
                  </div>
                  <div>
                    <h3 className="text-brand mb-1.5 text-lg font-semibold tracking-tight sm:mb-2 sm:text-xl">
                      Developer-First Experience
                    </h3>
                    <p className="text-sm text-gray-600 sm:text-base">
                      Seamless integration with your existing stack. Simple SDKs, predictable
                      pricing, and familiar database interfaces.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* TODO: this section will be modified when the actual SDK(s) is (are) ready */}
            <div className="rounded-2xl bg-white p-4 shadow-lg sm:p-6 lg:p-8">
              <div className="overflow-x-auto rounded-lg bg-gray-800 p-4 font-mono text-xs text-green-400 sm:p-6 sm:text-sm">
                <div className="mb-2">
                  <span className="text-gray-500">// Simple integration</span>
                </div>
                <div className="space-y-1">
                  <div>
                    <span className="text-blue-400">import</span> {"{"} GardbaseClient {"}"}{" "}
                    <span className="text-blue-400">from</span>{" "}
                    <span className="text-yellow-400">'@gardbase/client'</span>
                  </div>
                  <div className="mt-4">
                    <span className="text-blue-400">const</span>{" "}
                    <span className="text-white">gardb</span> ={" "}
                    <span className="text-blue-400">new</span>{" "}
                    <span className="text-white">GardbaseClient</span>({"{"}
                  </div>
                  <div className="ml-4">
                    <span className="text-white">apiKey</span>:{" "}
                    <span className="text-yellow-400">'your-api-key'</span>,
                  </div>
                  <div className="ml-4">
                    <span className="text-white">autoCompliance</span>:{" "}
                    <span className="text-orange-400">true</span>{" "}
                    <span className="text-gray-500">// Always on</span>
                  </div>
                  <div>{"}"});</div>
                  <div className="mt-4">
                    <span className="text-gray-500">// Data is encrypted client-side</span>
                  </div>
                  <div>
                    <span className="text-blue-400">await</span>{" "}
                    <span className="text-white">gardb</span>.
                    <span className="text-white">users</span>.
                    <span className="text-white">create</span>({"{"}{" "}
                    <span className="text-white">email</span>,{" "}
                    <span className="text-white">name</span> {"}"});
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="lg:py-25 mb-25 bg-white py-12 sm:py-16">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-12 text-center sm:mb-16">
            <h2 className="text-brand mb-3 text-2xl font-bold tracking-tight sm:mb-4 sm:text-3xl md:text-4xl">
              How Gardbase Works
            </h2>
            <p className="mx-auto max-w-3xl text-base text-gray-600 sm:text-lg md:text-xl">
              Three simple steps to breach-proof, compliant data storage
            </p>
          </div>

          {/* These details will eventually be subject to change */}
          <div className="grid gap-8 md:grid-cols-3">
            <div className="text-center">
              <div className="bg-brand mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl sm:mb-6 sm:h-16 sm:w-16">
                <LuLock className="h-6 w-6 text-white sm:h-8 sm:w-8" />
              </div>
              <h3 className="text-brand mb-3 text-lg font-semibold tracking-tight sm:mb-4 sm:text-xl">
                1. Client-Side Encryption
              </h3>
              <p className="text-sm text-gray-600 sm:text-base">
                Your data is encrypted on your servers before it ever reaches Gardbase. Even if our
                infrastructure is compromised, your data remains useless ciphertext to attackers.
              </p>
            </div>

            <div className="text-center">
              <div className="bg-brand mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl sm:mb-6 sm:h-16 sm:w-16">
                <LuDatabase className="h-6 w-6 text-white sm:h-8 sm:w-8" />
              </div>
              <h3 className="text-brand mb-3 text-lg font-semibold tracking-tight sm:mb-4 sm:text-xl">
                2. Automated Compliance
              </h3>
              <p className="text-sm text-gray-600 sm:text-base">
                All GDPR requirements are handled automatically—audit logs, data retention,
                portability, and deletion rights—without manual configuration or oversight.
              </p>
            </div>

            <div className="text-center">
              <div className="bg-brand mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl sm:mb-6 sm:h-16 sm:w-16">
                <LuShield className="h-6 w-6 text-white sm:h-8 sm:w-8" />
              </div>
              <h3 className="text-brand mb-3 text-lg font-semibold tracking-tight sm:mb-4 sm:text-xl">
                3. Seamless Integration
              </h3>
              <p className="text-sm text-gray-600 sm:text-base">
                Deploy through developer-friendly SDKs with simple, predictable pricing that
                includes all compliance features in your existing stack.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Pricing */}
      {/* Could be subject to change */}
      <section id="pricing" className="bg-slate-50 py-12 sm:py-16 lg:py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-12 text-center sm:mb-16">
            <h2 className="text-brand mb-3 text-2xl font-bold tracking-tight sm:mb-4 sm:text-3xl md:text-4xl">
              Simple, Predictable Pricing
            </h2>
            <p className="mx-auto max-w-3xl text-base text-gray-600 sm:text-lg md:text-xl">
              No hidden costs, no surprise bills. All compliance features included by default.
            </p>
          </div>

          <div className="grid gap-6 sm:gap-8 md:grid-cols-3">
            <div className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm sm:p-8">
              <h3 className="text-brand mb-2 text-xl font-bold sm:text-2xl">Startup</h3>
              <p className="mb-4 text-sm text-gray-600 sm:mb-6 sm:text-base">
                Perfect for early-stage Startups and small projects
              </p>
              <div className="text-brand mb-4 text-3xl font-bold sm:mb-6 sm:text-4xl">
                €79+<span className="text-lg font-normal text-gray-600 sm:text-xl">/month</span>
              </div>
              <ul className="mb-6 space-y-2 sm:mb-8 sm:space-y-3">
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Up to 20GB storage</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">25M API Calls/month</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Full GDPR compliance</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Email support</span>
                </li>
              </ul>
            </div>

            <div className="border-brand relative rounded-2xl border-2 bg-white p-6 shadow-lg sm:p-8">
              <div className="absolute -top-3 left-1/2 -translate-x-1/2 transform sm:-top-4">
                <span className="bg-brand rounded-full px-3 py-1 text-xs font-medium text-white sm:px-4 sm:py-2 sm:text-sm">
                  Most Popular
                </span>
              </div>
              <h3 className="text-brand mb-2 text-xl font-bold sm:text-2xl">Growth</h3>
              <p className="mb-4 text-sm text-gray-600 sm:mb-6 sm:text-base">
                For growing businesses and teams
              </p>
              <div className="text-brand mb-4 text-3xl font-bold sm:mb-6 sm:text-4xl">
                €349+<span className="text-lg font-normal text-gray-600 sm:text-xl">/month</span>
              </div>
              <ul className="mb-6 space-y-2 sm:mb-8 sm:space-y-3">
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Up to 100GB storage</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">150M API Calls/month</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Full GDPR compliance</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Priority support</span>
                </li>
              </ul>
            </div>

            <div className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm sm:p-8">
              <h3 className="text-brand mb-2 text-xl font-bold sm:text-2xl">Scale</h3>
              <p className="mb-4 text-sm text-gray-600 sm:mb-6 sm:text-base">
                For large organizations with custom needs
              </p>
              <div className="text-brand mb-4 text-3xl font-bold sm:mb-6 sm:text-4xl">Custom</div>
              <ul className="mb-6 space-y-2 sm:mb-8 sm:space-y-3">
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Custom storage</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Custom API Calls limit</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">Dedicated support</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                  <span className="text-sm text-gray-600 sm:text-base">SLA guarantees</span>
                </li>
              </ul>
              <button className="bg-brand hover:bg-brand/90 w-full rounded-lg py-2.5 text-sm font-semibold text-white transition-colors sm:py-3 sm:text-base">
                Contact Sales
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="bg-brand border-b border-gray-800 py-12 sm:py-16 lg:py-20">
        <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
          <h2 className="mb-3 text-2xl font-bold text-white sm:mb-4 sm:text-3xl md:text-4xl">
            Ready to Build with Confidence?
          </h2>
          <p className="mx-auto mb-6 max-w-2xl text-base text-blue-200 sm:mb-8 sm:text-lg md:text-xl">
            Join the developers who've chosen to focus on building great products instead of
            managing compliance complexity.
          </p>
          <div className="flex flex-col gap-3 sm:flex-row sm:justify-center sm:gap-4">
            <button className="text-brand rounded-lg bg-white px-6 py-3 text-base font-semibold transition-colors hover:bg-gray-100 sm:px-8 sm:py-4 sm:text-lg">
              Start Free Trial
            </button>
            <button className="rounded-lg border border-blue-200 px-6 py-3 text-base font-semibold text-white transition-colors hover:bg-white/10 sm:px-8 sm:py-4 sm:text-lg">
              Schedule Demo
            </button>
          </div>
        </div>
      </section>

      {/* Footer */}
      {/* All links will be subject to change here */}
      <footer className="bg-brand py-8 text-white sm:py-12">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="grid gap-8 sm:grid-cols-2 md:grid-cols-4">
            <div className="sm:col-span-2 md:col-span-1">
              <div className="mb-4 flex items-center space-x-2">
                <img draggable={false} src={LogoWhite} alt="Gardbase Logo" className="h-6 sm:h-8" />
              </div>
              <p className="text-sm text-gray-400 sm:text-base">
                The GDPR-native database that protects by design.
              </p>
            </div>

            <div>
              <h3 className="mb-3 text-sm font-semibold sm:mb-4 sm:text-base">Product</h3>
              <ul className="space-y-2 text-xs text-gray-400 sm:text-sm">
                <li>
                  <a href="#features" className="transition-colors hover:text-white">
                    Features
                  </a>
                </li>
                <li>
                  <a href="#pricing" className="transition-colors hover:text-white">
                    Pricing
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Documentation
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    API Reference
                  </a>
                </li>
              </ul>
            </div>

            <div>
              <h3 className="mb-3 text-sm font-semibold sm:mb-4 sm:text-base">Company</h3>
              <ul className="space-y-2 text-xs text-gray-400 sm:text-sm">
                <li>
                  <a href="https://qodesrl.com/" className="transition-colors hover:text-white">
                    About
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Blog
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Careers
                  </a>
                </li>
                <li>
                  <a href="https://qodesrl.com/" className="transition-colors hover:text-white">
                    Contact
                  </a>
                </li>
              </ul>
            </div>

            <div>
              <h3 className="mb-3 text-sm font-semibold sm:mb-4 sm:text-base">Legal</h3>
              <ul className="space-y-2 text-xs text-gray-400 sm:text-sm">
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Privacy Policy
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Terms of Service
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Security
                  </a>
                </li>
                <li>
                  <a href="#" className="transition-colors hover:text-white">
                    Compliance
                  </a>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-8 border-t border-gray-800 pt-6 text-center text-xs text-gray-400 sm:pt-8 sm:text-sm">
            <p>&copy; 2025 QodeSrl - Gardbase. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default MainPage;
