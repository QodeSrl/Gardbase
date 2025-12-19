import React from "react";
import { LuShield, LuArrowRight } from "react-icons/lu";

const HeroSection: React.FC = () => {
  return (
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
            Traditional databases store everything. When breached, they expose everything. Gardbase
            flips the script—storing only encrypted data, makes catastrophic data breaches{" "}
            <strong>architecturally impossible</strong>.
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
  );
};

export default HeroSection;
