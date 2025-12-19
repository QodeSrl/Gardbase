import React from "react";
import { LuLock, LuDatabase, LuShield } from "react-icons/lu";

const howItWorksParagraphs = [
  {
    title: "Client-Side Encryption",
    text: "Your data is encrypted on your servers before it ever reaches Gardbase. Even if our infrastructure is compromised, your data remains useless ciphertext to attackers.",
    icon: <LuLock className="h-6 w-6 text-white sm:h-8 sm:w-8" />,
  },
  {
    title: "Automated Compliance",
    text: "All GDPR requirements are handled automatically—audit logs, data retention, portability, and deletion rights—without manual configuration or oversight.",
    icon: <LuDatabase className="h-6 w-6 text-white sm:h-8 sm:w-8" />,
  },
  {
    title: "Seamless Integration",
    text: "Deploy through developer-friendly SDKs with simple, predictable pricing that includes all compliance features in your existing stack.",
    icon: <LuShield className="h-6 w-6 text-white sm:h-8 sm:w-8" />,
  },
];

const HowItWorksSection: React.FC = () => {
  return (
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
          {howItWorksParagraphs.map((item, i) => (
            <div key={i} className="text-center">
              <div className="bg-brand mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl sm:mb-6 sm:h-16 sm:w-16">
                {item.icon}
              </div>
              <h3 className="text-brand mb-3 text-lg font-semibold tracking-tight sm:mb-4 sm:text-xl">
                {item.title}
              </h3>
              <p className="text-sm text-gray-600 sm:text-base">{item.text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorksSection;
