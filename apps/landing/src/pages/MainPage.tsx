import React from "react";
import Header from "@/components/Header";
import HeroSection from "@/components/HeroSection";
import ProblemStatementSection from "@/components/ProblemStatementSection";
import FeaturesSection from "@/components/FeaturesSection";
import HowItWorksSection from "@/components/HowItWorksSection";
import OpenSourceSection from "@/components/OpenSourceSection";
import PricingSection from "@/components/PricingSection";
import CTASection from "@/components/CTASection";
import Footer from "@/components/Footer";

const MainPage: React.FC = () => {
  return (
    <div className="min-h-screen bg-bg">
      <Header />
      <main>
        <HeroSection />
        <ProblemStatementSection />
        <FeaturesSection />
        <HowItWorksSection />
        <OpenSourceSection />
        <PricingSection />
        <CTASection />
      </main>
      <Footer />
    </div>
  );
};

export default MainPage;
