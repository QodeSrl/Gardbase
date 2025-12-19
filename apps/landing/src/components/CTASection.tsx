import React from "react";

const CTASection: React.FC = () => {
  return (
    <section className="bg-brand border-b border-gray-800 py-12 sm:py-16 lg:py-20">
      <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
        <h2 className="mb-3 text-2xl font-bold text-white sm:mb-4 sm:text-3xl md:text-4xl">
          Ready to Build with Confidence?
        </h2>
        <p className="mx-auto mb-6 max-w-2xl text-base text-blue-200 sm:mb-8 sm:text-lg md:text-xl">
          Join the developers who've chosen to focus on building great products instead of managing
          compliance complexity.
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
  );
};

export default CTASection;
