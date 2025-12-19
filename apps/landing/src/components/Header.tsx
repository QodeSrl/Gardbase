import React, { useState } from "react";
import Logo from "../assets/logo.svg";
import { LuMenu, LuX } from "react-icons/lu";

const navLinks = [
  { name: "Features", href: "#features" },
  { name: "How it Works", href: "#how-it-works" },
  { name: "Pricing", href: "#pricing" },
];

const Header: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  return (
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
            {navLinks.map(link => (
              <a
                key={link.name}
                href={link.href}
                className="hover:text-brand text-sm text-gray-600 transition-colors lg:text-base"
              >
                {link.name}
              </a>
            ))}
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
            {navLinks.map(link => (
              <a
                key={link.name}
                href={link.href}
                className="hover:text-brand block rounded-md px-3 py-2 text-base font-medium text-gray-600 hover:bg-gray-50"
                onClick={() => setMobileMenuOpen(false)}
              >
                {link.name}
              </a>
            ))}
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
  );
};

export default Header;
