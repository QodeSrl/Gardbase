import React from "react";

const ProblemStatementSection: React.FC = () => {
  return (
    <section className="lg:py-25 mb-20 bg-white py-12 sm:py-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mb-20 text-center sm:mb-16">
          <h2 className="text-brand mb-3 text-2xl font-bold tracking-tight sm:mb-4 sm:text-3xl md:text-4xl">
            GDPR Compliance Shouldn't Be This Hard
          </h2>
          <p className="mx-auto max-w-3xl text-base text-gray-600 sm:text-lg md:text-xl">
            Traditional databases make you choose between developer experience and compliance. With
            complex configurations, shared responsibility models, and constant legal uncertainty.
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
              SMEs face disproportionate compliance costs, with some experiencing over 8% reduction
              in profits due to regulatory overhead.
            </p>
          </div>

          <div className="rounded-2xl bg-yellow-50 p-6 sm:p-8">
            <div className="mb-2 text-xs font-medium uppercase tracking-wider text-amber-500 group-hover:text-amber-400 sm:text-sm">
              Technical Burden
            </div>
            <h3 className="text-brand mb-2 text-lg font-semibold sm:text-xl">Complex Tools</h3>
            <p className="text-sm text-gray-600 sm:text-base">
              Current solutions offer compliance as a tool you must configure correctly, leaving all
              responsibility and risk with you.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
};

export default ProblemStatementSection;
