import React from "react";
import { LuShield, LuCircleCheck, LuDatabase } from "react-icons/lu";

const FeaturesSection: React.FC = () => {
  return (
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
                    Client-side encryption with zero-knowledge design. Even if our infrastructure is
                    compromised, your data remains useless ciphertext.
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
                    Seamless integration with your existing stack. Simple SDKs, predictable pricing,
                    and familiar database interfaces.
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
  );
};

export default FeaturesSection;
