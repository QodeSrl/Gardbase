import React from "react";
import { LuCircleCheck } from "react-icons/lu";

const pricingPlans = [
  {
    name: "Startup",
    description: "Perfect for early-stage startups and small projects",
    price: "€79+",
    features: [
      "Up to 20GB storage",
      "25M API Calls/month",
      "Full GDPR compliance",
      "Email support",
    ],
  },
  {
    name: "Growth",
    description: "For growing businesses and teams",
    price: "€349+",
    features: [
      "Up to 100GB storage",
      "150M API Calls/month",
      "Full GDPR compliance",
      "Priority support",
    ],
  },
  {
    name: "Scale",
    description: "For large organizations with custom needs",
    price: "Custom",
    features: ["Custom storage", "Custom API Calls limit", "Dedicated support", "SLA guarantees"],
  },
];

const PricingSection: React.FC = () => {
  /* Could be subject to change */
  return (
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
          {pricingPlans.map((plan, index) => (
            <div
              key={index}
              className={`relative rounded-2xl border ${
                index === 1 ? "border-brand bg-white shadow-lg" : "border-gray-200 bg-white"
              } p-6 sm:p-8`}
            >
              {index === 1 && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2 transform sm:-top-4">
                  <span className="bg-brand rounded-full px-3 py-1 text-xs font-medium text-white sm:px-4 sm:py-2 sm:text-sm">
                    Most Popular
                  </span>
                </div>
              )}
              <h3 className="text-brand mb-2 text-xl font-bold sm:text-2xl">{plan.name}</h3>
              <p className="mb-4 text-sm text-gray-600 sm:mb-6 sm:text-base">{plan.description}</p>
              <div className="text-brand mb-4 text-3xl font-bold sm:mb-6 sm:text-4xl">
                {plan.price}
                <span className="text-lg font-normal text-gray-600 sm:text-xl">/month</span>
              </div>
              <ul className="mb-6 space-y-2 sm:mb-8 sm:space-y-3">
                {plan.features.map((feature, featureIndex) => (
                  <li key={featureIndex} className="flex items-center">
                    <LuCircleCheck className="mr-2 h-4 w-4 flex-shrink-0 text-green-500 sm:mr-3 sm:h-5 sm:w-5" />
                    <span className="text-sm text-gray-600 sm:text-base">{feature}</span>
                  </li>
                ))}
              </ul>
              {plan.name === "Scale" && (
                <button className="bg-brand hover:bg-brand/90 w-full rounded-lg py-2.5 text-sm font-semibold text-white transition-colors sm:py-3 sm:text-base">
                  Contact Sales
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default PricingSection;
