import React, { useEffect, useState } from "react";
import Logo from "../assets/logo.svg";
import LogoWhite from "../assets/logo-white.svg";
import { LuMenu, LuX, LuStar, LuSun, LuMoon } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";
import { useTheme } from "@/lib/themeContext";

const navLinks = [
  { name: "Why Gardbase", href: "#why" },
  { name: "Features", href: "#features" },
  { name: "How it Works", href: "#how-it-works" },
  { name: "Open Source", href: "#open-source" },
  { name: "Pricing", href: "#pricing" },
];

const Header: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const { theme, toggle } = useTheme();

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const ThemeToggle = (
    <button
      onClick={toggle}
      aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
      title={theme === "dark" ? "Light mode" : "Dark mode"}
      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-line bg-fg/5 text-muted transition-colors hover:bg-fg/10 hover:text-fg"
    >
      {theme === "dark" ? <LuSun className="h-4 w-4" /> : <LuMoon className="h-4 w-4" />}
    </button>
  );

  return (
    <header
      className={`sticky top-0 z-50 transition-all duration-300 ${
        scrolled ? "glass border-b" : "border-b border-transparent bg-transparent"
      }`}
    >
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center justify-between">
          <a href="#" className="flex items-center">
            <img
              draggable={false}
              src={theme === "dark" ? LogoWhite : Logo}
              alt="Gardbase"
              className="h-6 sm:h-7"
            />
          </a>

          {/* Desktop Navigation */}
          <nav className="hidden items-center space-x-1 md:flex lg:space-x-2">
            {navLinks.map(link => (
              <a
                key={link.name}
                href={link.href}
                className="rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-fg/5 hover:text-fg"
              >
                {link.name}
              </a>
            ))}
          </nav>

          {/* Desktop Actions */}
          <div className="hidden items-center space-x-2 md:flex lg:space-x-3">
            {ThemeToggle}
            <a
              href={site.repo}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 rounded-lg border border-line bg-fg/5 px-3 py-2 text-sm text-muted transition-colors hover:bg-fg/10 hover:text-fg"
            >
              <SiGithub className="h-4 w-4" />
              <span className="hidden lg:inline">Star</span>
              <LuStar className="h-3.5 w-3.5 text-amber-400" />
            </a>
            <a
              href={site.docs}
              target="_blank"
              rel="noreferrer"
              className="rounded-lg bg-gradient-to-r from-accent to-accent-3 px-4 py-2 text-sm font-semibold text-white shadow-lg shadow-accent/25 transition-transform hover:scale-[1.03]"
            >
              Get Started
            </a>
          </div>

          {/* Mobile actions */}
          <div className="flex items-center gap-2 md:hidden">
            {ThemeToggle}
            <button
              onClick={() => setMobileMenuOpen(prev => !prev)}
              aria-label={mobileMenuOpen ? "Close menu" : "Open menu"}
              aria-expanded={mobileMenuOpen}
              className="inline-flex items-center justify-center rounded-md p-2 text-muted hover:bg-fg/10 focus:outline-none focus:ring-2 focus:ring-accent"
            >
              {mobileMenuOpen ? <LuX className="h-6 w-6" /> : <LuMenu className="h-6 w-6" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile menu */}
      {mobileMenuOpen && (
        <div className="glass border-t md:hidden">
          <div className="space-y-1 px-4 pb-4 pt-2">
            {navLinks.map(link => (
              <a
                key={link.name}
                href={link.href}
                className="block rounded-md px-3 py-2 text-base font-medium text-muted hover:bg-fg/5 hover:text-fg"
                onClick={() => setMobileMenuOpen(false)}
              >
                {link.name}
              </a>
            ))}
            <div className="mt-3 grid grid-cols-1 gap-2 border-t border-line pt-4">
              <a
                href={site.repo}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-line bg-fg/5 px-3 py-2.5 text-base font-medium text-fg"
              >
                <SiGithub className="h-4 w-4" /> View on GitHub
              </a>
              <a
                href={site.docs}
                target="_blank"
                rel="noreferrer"
                className="rounded-lg bg-gradient-to-r from-accent to-accent-3 px-3 py-2.5 text-center text-base font-semibold text-white"
              >
                Get Started
              </a>
            </div>
          </div>
        </div>
      )}
    </header>
  );
};

export default Header;
