import React from "react";
import Header from "@/components/Header";
import HeroSection from "@/components/HeroSection";
import ProblemStatementSection from "@/components/ProblemStatementSection";
import FeaturesSection from "@/components/FeaturesSection";
import HowItWorksSection from "@/components/HowItWorksSection";
import PricingSection from "@/components/PricingSection";
import CTASection from "@/components/CTASection";

const MainPage: React.FC = () => {
  return (
    <div className="min-h-screen bg-white">
      <Header />
      <HeroSection />
      <ProblemStatementSection />
      <FeaturesSection />
      <HowItWorksSection />
      <PricingSection />
      <CTASection />
    </div>
  );
};

export default MainPage;
