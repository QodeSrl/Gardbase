import React from "react";
import { LuShield, LuDatabase, LuLock, LuArrowRight, LuCircleCheck } from "react-icons/lu";
import Logo from "../assets/logo.svg";
import LogoWhite from "../assets/logo-white.svg";

const MainPage: React.FC = () => {
  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-gray-100 bg-white">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center">
              <a href="/#">
                <img draggable={false} src={Logo} alt="Gardbase Logo" className="h-8" />
              </a>
            </div>
            <nav className="hidden space-x-8 md:flex">
              <a href="#features" className="hover:text-brand text-gray-600 transition-colors">
                Features
              </a>
              <a href="#how-it-works" className="hover:text-brand text-gray-600 transition-colors">
                How it Works
              </a>
              <a href="#pricing" className="hover:text-brand text-gray-600 transition-colors">
                Pricing
              </a>
            </nav>
            <div className="flex items-center space-x-4">
              <button className="hover:text-brand text-gray-600 transition-colors">Sign In</button>
              <button className="bg-brand hover:bg-brand/90 rounded-lg px-4 py-2 text-white transition-colors">
                Get Started
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative bg-gradient-to-b from-slate-50 to-white py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <div className="text-brand mb-6 inline-flex items-center rounded-full bg-blue-50 px-4 py-2 text-sm font-medium">
              <LuShield className="mr-2 h-4 w-4" />
              GDPR-Native Database Platform
            </div>
            <h1 className="text-brand mb-6 text-5xl font-bold md:text-6xl">
              The Database That <span className="md:block">Can't Be Breached</span>
            </h1>
            <p className="mx-auto mb-8 max-w-3xl text-xl text-gray-600">
              Traditional databases store everything. When breached, they expose everything. Gardbase flips the script—storing only encrypted data, makes catastrophic data breaches <strong>architecturally impossible</strong>.
            </p>
            <div className="flex flex-col justify-center gap-4 sm:flex-row">
              <button className="bg-brand hover:bg-brand/90 flex items-center justify-center rounded-lg px-8 py-4 text-lg font-semibold text-white transition-colors">
                See How It Works
                <LuArrowRight className="ml-2 h-5 w-5" />
              </button>
              <button className="text-brand rounded-lg border border-gray-300 px-8 py-4 text-lg font-semibold transition-colors hover:bg-gray-50">
                Try Live Demo
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Problem Statement */}
      <section className="bg-white py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-16 text-center">
            <h2 className="text-brand mb-4 text-3xl font-bold tracking-tight md:text-4xl">
              GDPR Compliance Shouldn't Be This Hard
            </h2>
            <p className="mx-auto max-w-3xl text-xl text-gray-600">
              Traditional databases make you choose between developer experience and compliance.
              With complex configurations, shared responsibility models, and constant legal
              uncertainty.
            </p>
          </div>

          <div className="grid gap-8 md:grid-cols-3">
            <div className="rounded-2xl bg-red-50 p-8">
              <div className="mb-2 text-sm font-medium tracking-wider text-red-500 uppercase group-hover:text-red-400">
                Financial Impact
              </div>
              <h3 className="text-brand mb-2 text-xl font-semibold">€5.88B in Fines</h3>
              <p className="text-gray-600">
                GDPR fines continue to escalate, with enforcement becoming more aggressive and
                personal liability extending to executives.
              </p>
            </div>

            <div className="rounded-2xl bg-orange-50 p-8">
              <div className="mb-2 text-sm font-medium tracking-wider text-orange-500 uppercase group-hover:text-orange-400">
                Business Cost
              </div>
              <h3 className="text-brand mb-2 text-xl font-semibold">8% Profit Drop</h3>
              <p className="text-gray-600">
                SMEs face disproportionate compliance costs, with some experiencing over 8%
                reduction in profits due to regulatory overhead.
              </p>
            </div>

            <div className="rounded-2xl bg-yellow-50 p-8">
              <div className="mb-2 text-sm font-medium tracking-wider text-amber-500 uppercase group-hover:text-amber-400">
                Technical Burden
              </div>
              <h3 className="text-brand mb-2 text-xl font-semibold">Complex Tools</h3>
              <p className="text-gray-600">
                Current solutions offer compliance as a tool you must configure correctly, leaving
                all responsibility and risk with you.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Solution */}
      <section id="features" className="bg-slate-50 py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-16 text-center">
            <h2 className="text-brand mb-4 text-3xl font-bold tracking-tight md:text-4xl">
              Compliance as a Service, Not a Tool
            </h2>
            <p className="mx-auto max-w-3xl text-xl text-gray-600">
              Gardbase shifts the paradigm from complex configuration to automatic compliance. Our
              zero-knowledge architecture makes data breaches technically impossible.
            </p>
          </div>

          <div className="grid items-center gap-12 md:grid-cols-2">
            <div>
              <div className="space-y-8">
                <div className="flex items-start space-x-4">
                  <div className="bg-brand flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl">
                    <LuShield className="h-5 w-5 text-white" />
                  </div>
                  <div>
                    <h3 className="text-brand mb-2 text-xl font-semibold tracking-tight">
                      Breach-Proof Architecture
                    </h3>
                    <p className="text-gray-600">
                      Client-side encryption with zero-knowledge design. Even if our infrastructure
                      is compromised, your data remains useless ciphertext.
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-4">
                  <div className="bg-brand flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl">
                    <LuCircleCheck className="h-5 w-5 text-white" />
                  </div>
                  <div>
                    <h3 className="text-brand mb-2 text-xl font-semibold tracking-tight">
                      Automated Compliance
                    </h3>
                    <p className="text-gray-600">
                      GDPR-conformant audit logs, automated data retention, and built-in portability
                      features work out of the box.
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-4">
                  <div className="bg-brand flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl">
                    <LuDatabase className="h-5 w-5 text-white" />
                  </div>
                  <div>
                    <h3 className="text-brand mb-2 text-xl font-semibold tracking-tight">
                      Developer-First Experience
                    </h3>
                    <p className="text-gray-600">
                      Seamless integration with your existing stack. Simple SDKs, predictable
                      pricing, and familiar database interfaces.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* TODO: this section will be modified when the actual SDK(s) is (are) ready */}
            <div className="rounded-2xl bg-white p-8 shadow-lg">
              <div className="rounded-lg bg-gray-800 p-6 font-mono text-sm text-green-400">
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
      <section id="how-it-works" className="bg-white py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-16 text-center">
            <h2 className="text-brand mb-4 text-3xl font-bold tracking-tight md:text-4xl">
              How Gardbase Works
            </h2>
            <p className="mx-auto max-w-3xl text-xl text-gray-600">
              Three simple steps to breach-proof, compliant data storage
            </p>
          </div>

          {/* These details will eventually be subject to change */}
          <div className="grid gap-8 md:grid-cols-3">
            <div className="text-center">
              <div className="bg-brand mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl">
                <LuLock className="h-8 w-8 text-white" />
              </div>
              <h3 className="text-brand mb-4 text-xl font-semibold tracking-tight">
                1. Client-Side Encryption
              </h3>
              <p className="text-gray-600">
                Your data is encrypted on your servers before it ever reaches Gardbase. Even if our
                infrastructure is compromised, your data remains useless ciphertext to attackers.
              </p>
            </div>

            <div className="text-center">
              <div className="bg-brand mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl">
                <LuDatabase className="h-8 w-8 text-white" />
              </div>
              <h3 className="text-brand mb-4 text-xl font-semibold tracking-tight">
                2. Automated Compliance
              </h3>
              <p className="text-gray-600">
                All GDPR requirements are handled automatically—audit logs, data retention,
                portability, and deletion rights—without manual configuration or oversight.
              </p>
            </div>

            <div className="text-center">
              <div className="bg-brand mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl">
                <LuShield className="h-8 w-8 text-white" />
              </div>
              <h3 className="text-brand mb-4 text-xl font-semibold tracking-tight">
                3. Seamless Integration
              </h3>
              <p className="text-gray-600">
                Deploy through developer-friendly SDKs with simple, predictable pricing that
                includes all compliance features in your existing stack.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Pricing */}
      {/* Could be subject to change */}
      <section id="pricing" className="bg-slate-50 py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mb-16 text-center">
            <h2 className="text-brand mb-4 text-3xl font-bold tracking-tight md:text-4xl">
              Simple, Predictable Pricing
            </h2>
            <p className="mx-auto max-w-3xl text-xl text-gray-600">
              No hidden costs, no surprise bills. All compliance features included by default.
            </p>
          </div>

          <div className="grid gap-8 md:grid-cols-3">
            <div className="rounded-2xl border border-gray-200 bg-white p-8 shadow-sm">
              <h3 className="text-brand mb-2 text-2xl font-bold">Startup</h3>
              <p className="mb-6 text-gray-600">
                Perfect for early-stage Startups and small projects
              </p>
              <div className="text-brand mb-6 text-4xl font-bold">
                €79+<span className="text-xl font-normal text-gray-600">/month</span>
              </div>
              <ul className="mb-8 space-y-3">
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Up to 20GB storage</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">25M API Calls/month</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Full GDPR compliance</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Email support</span>
                </li>
              </ul>
            </div>

            <div className="border-brand relative rounded-2xl border-2 bg-white p-8 shadow-lg">
              <div className="absolute -top-4 left-1/2 -translate-x-1/2 transform">
                <span className="bg-brand rounded-full px-4 py-2 text-sm font-medium text-white">
                  Most Popular
                </span>
              </div>
              <h3 className="text-brand mb-2 text-2xl font-bold">Growth</h3>
              <p className="mb-6 text-gray-600">For growing businesses and teams</p>
              <div className="text-brand mb-6 text-4xl font-bold">
                €349+<span className="text-xl font-normal text-gray-600">/month</span>
              </div>
              <ul className="mb-8 space-y-3">
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Up to 100GB storage</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">150M API Calls/month</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Full GDPR compliance</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Priority support</span>
                </li>
              </ul>
            </div>

            <div className="rounded-2xl border border-gray-200 bg-white p-8 shadow-sm">
              <h3 className="text-brand mb-2 text-2xl font-bold">Scale</h3>
              <p className="mb-6 text-gray-600">For large organizations with custom needs</p>
              <div className="text-brand mb-6 text-4xl font-bold">Custom</div>
              <ul className="mb-8 space-y-3">
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Custom storage</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Custom API Calls limit</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">Dedicated support</span>
                </li>
                <li className="flex items-center">
                  <LuCircleCheck className="mr-3 h-5 w-5 text-green-500" />
                  <span className="text-gray-600">SLA guarantees</span>
                </li>
              </ul>
              <button className="bg-brand hover:bg-brand/90 w-full rounded-lg py-3 font-semibold text-white transition-colors">
                Contact Sales
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="bg-brand border-b border-gray-800 py-20">
        <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
          <h2 className="mb-4 text-3xl font-bold text-white md:text-4xl">
            Ready to Build with Confidence?
          </h2>
          <p className="mx-auto mb-8 max-w-2xl text-xl text-blue-200">
            Join the developers who've chosen to focus on building great products instead of
            managing compliance complexity.
          </p>
          <div className="flex flex-col justify-center gap-4 sm:flex-row">
            <button className="text-brand rounded-lg bg-white px-8 py-4 text-lg font-semibold transition-colors hover:bg-gray-100">
              Start Free Trial
            </button>
            <button className="rounded-lg border border-blue-200 px-8 py-4 text-lg font-semibold text-white transition-colors hover:bg-white/10">
              Schedule Demo
            </button>
          </div>
        </div>
      </section>

      {/* Footer */}
      {/* All links will be subject to change here */}
      <footer className="bg-brand py-12 text-white">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="grid gap-8 md:grid-cols-4">
            <div>
              <div className="mb-4 flex items-center space-x-2">
                <img draggable={false} src={LogoWhite} alt="Gardbase Logo" className="h-8" />
              </div>
              <p className="text-gray-400">The GDPR-native database that protects by design.</p>
            </div>

            <div>
              <h3 className="mb-4 font-semibold">Product</h3>
              <ul className="space-y-2 text-gray-400">
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
              <h3 className="mb-4 font-semibold">Company</h3>
              <ul className="space-y-2 text-gray-400">
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
              <h3 className="mb-4 font-semibold">Legal</h3>
              <ul className="space-y-2 text-gray-400">
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

          <div className="mt-8 border-t border-gray-800 pt-8 text-center text-gray-400">
            <p>&copy; 2025 QodeSrl - Gardbase. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default MainPage;
